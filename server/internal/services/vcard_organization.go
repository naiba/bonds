package services

import (
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// reconcileVCardOrganization runs inside the caller's contact transaction.
// ORG must populate the same jobs that normal UI writes and vCard exports use.
// Departments are not company names; split components before interpreting text.
// Derive both compatibility fields from one coordinated job. card.Value(TITLE)
// may belong to a secondary ORG when the selected company has no title.
func reconcileVCardOrganization(tx *gorm.DB, card vcard.Card, contactID, vaultID string, replace bool) error {
	organizations := effectiveVCardOrganizations(card)
	if len(organizations) != len(card[vcard.FieldOrganization]) {
		// Empty ORG (including department-only components) is not a company.
		// Still reject an explicit clear of existing employment before normalizing
		// it, rather than silently discarding Jobs or changing their identity.
		if len(organizations) == 0 {
			if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
				return err
			}
			contact, err := loadContactEmployment(tx, contactID, vaultID)
			if err != nil {
				return err
			}
			if contact.CompanyID != nil || len(contact.ContactCompanies) > 0 {
				return ErrVCardUnsafeReplacement
			}
		}
		normalized := make(vcard.Card, len(card))
		for key, fields := range card {
			normalized[key] = fields
		}
		normalized[vcard.FieldOrganization] = organizations
		card = normalized
	}

	if len(card[vcard.FieldOrganization]) == 0 && len(card[vcard.FieldTitle]) == 0 {
		return nil
	}
	preserveStandaloneSelection := false
	if len(card[vcard.FieldOrganization]) == 0 {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		contact, err := loadContactEmployment(tx, contactID, vaultID)
		if err != nil {
			return err
		}
		// Several TITLE-only fields can be existing Jobs whose employer names
		// are blank. Match those identities below instead of choosing one title.
		titleOnlyJobs := hasTitleOnlyJobs(contact)
		if !titleOnlyJobs && len(card[vcard.FieldTitle]) > 1 {
			return ErrVCardUnsafeReplacement
		}
		// Clearing the independent title must not select and overwrite a remaining
		// Job merely because the cleared title no longer appears in the projection.
		if hasStandaloneEmployment(contact) && isUngroupedTitleEdit(card) {
			contact.JobPosition = strPtrOrNil(strings.TrimSpace(card.Value(vcard.FieldTitle)))
			return updateContactProfile(tx, contact, vaultID, "job_position")
		}
		preserveStandaloneSelection = hasStandaloneEmployment(contact)
	}
	if err := reconcileVCardEmploymentRecords(tx, card, contactID, vaultID, replace); err != nil {
		return err
	}
	// Grouped records already reconciled the independent field, including its
	// omission during a full replacement. Do not fill it from a retained Job.
	if preserveStandaloneSelection {
		return nil
	}
	contact, err := loadContactEmployment(tx, contactID, vaultID)
	if err != nil {
		return err
	}
	employment := ContactVCardEmployments(contact)
	if len(employment) > 0 && employment[0].JobID != 0 {
		for _, job := range contact.ContactCompanies {
			if job.ID == employment[0].JobID {
				id := job.CompanyID
				contact.PreferredJobID = &job.ID
				contact.CompanyID = &id
				contact.JobPosition = job.JobPosition
				break
			}
		}
	} else if len(card[vcard.FieldOrganization]) == 0 && !hasStandaloneEmployment(contact) {
		// Standalone titles use the same blank-value semantics as the projection.
		contact.JobPosition = strPtrOrNil(strings.TrimSpace(card.Value(vcard.FieldTitle)))
	}
	return updateContactProfile(tx, contact, vaultID, "company_id", "job_position", "preferred_job_id")
}

func reconcileVCardEmploymentRecords(tx *gorm.DB, card vcard.Card, contactID, vaultID string, replace bool) error {
	organizations := card[vcard.FieldOrganization]
	if len(organizations) == 0 && len(card[vcard.FieldTitle]) == 0 {
		return nil
	}
	if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
		return err
	}
	// Read both associated and automatic candidates under the same name lock.
	// Loading the projection before waiting could reuse a pre-rename name.
	if len(organizations) > 0 {
		if err := lockCompanyResolution(tx, vaultID); err != nil {
			return err
		}
	}
	contact, err := loadContactEmployment(tx, contactID, vaultID)
	if err != nil {
		return err
	}
	if err := materializeLegacyEmployment(tx, contact); err != nil {
		return err
	}
	titleOnlyJobs := hasTitleOnlyJobs(contact)
	projected := ContactVCardEmployments(contact)
	titleRecordsOnly := len(organizations) == 0 && !isUngroupedTitleEdit(card)
	if isUngroupedTitleEdit(card) {
		// A single TITLE without ORG still edits the selected named Job, even
		// when a secondary blank employer also exports TITLE alone (#297).
		if len(projected) > 0 && projected[0].JobID != 0 {
			job := models.ContactCompany{ID: projected[0].JobID, ContactID: contactID, JobPosition: strPtrOrNil(card.Value(vcard.FieldTitle))}
			return updateContactRecord(tx, &job, contactID, vaultID, ErrContactJobNotFound, "job_position")
		}
		return nil
	}
	// Resolve the entire input before writing. A greedy per-field match lets an
	// inserted/edited first role consume an unchanged later role's identity.
	type incomingEmployment struct {
		name, title string
		jobID       uint
		matched     bool
		standalone  bool
	}
	var incoming []incomingEmployment
	matchedTitles := map[*vcard.Field]bool{}
	for _, field := range organizations {
		components := field.Organization()
		if len(components) == 0 || strings.TrimSpace(components[0]) == "" {
			continue
		}
		title := ""
		fieldTitle, err := vcardOrganizationTitle(card, field.Group)
		if err != nil {
			return err
		}
		if fieldTitle != nil {
			// A TITLE belongs to one ORG only, even when both lack a group.
			if matchedTitles[fieldTitle] {
				return ErrVCardUnsafeReplacement
			}
			title = strings.TrimSpace(fieldTitle.Value)
			matchedTitles[fieldTitle] = true
		}
		// Missing TITLE is an empty title, not permission to keep an old title.
		incoming = append(incoming, incomingEmployment{name: strings.TrimSpace(components[0]), title: title})
	}
	// A TITLE without ORG may still belong to a real Job with a blank company
	// name (#297). Feed it through the same identity matching as named Jobs;
	// otherwise an unchanged DAV PUT copies it into an independent title.
	for _, title := range card[vcard.FieldTitle] {
		if matchedTitles[title] || (strings.TrimSpace(title.Value) == "" && !titleOnlyJobs) {
			continue
		}
		if len(organizations) > 0 && title.Group == "" {
			return ErrVCardUnsafeReplacement
		}
		for _, org := range organizations {
			if strings.EqualFold(org.Group, title.Group) {
				return ErrVCardUnsafeReplacement
			}
		}
		incoming = append(incoming, incomingEmployment{title: strings.TrimSpace(title.Value)})
	}
	if len(incoming) == 0 && (len(contact.ContactCompanies) > 0 || contact.CompanyID != nil) {
		return ErrVCardUnsafeReplacement
	}
	// A grouped TITLE can be an unchanged secondary record (#297). Group
	// labels are not row IDs; if an omitted duplicate could be either the
	// independent title or a Job, deletion/edit identity is unknowable.
	if titleRecordsOnly {
		for _, entry := range incoming {
			candidates, supplied := 0, 0
			standalone := false
			for _, job := range projected {
				if job.Company == entry.name && job.Title == entry.title {
					candidates++
					standalone = standalone || job.JobID == 0
				}
			}
			for _, other := range incoming {
				if other.name == entry.name && other.title == entry.title {
					supplied++
				}
			}
			if standalone && candidates > supplied {
				return ErrVCardUnsafeReplacement
			}
		}
	}
	used := map[uint]bool{}
	for i := range incoming {
		for _, job := range projected {
			if !used[job.JobID] && job.Company == incoming[i].name && job.Title == incoming[i].title {
				incoming[i].jobID = job.JobID
				incoming[i].matched = true
				incoming[i].standalone = job.JobID == 0
				used[job.JobID] = true
				break
			}
		}
	}
	// Exact matches (including empty titles) are reserved first. Within a company
	// only one remaining input and one remaining job identify an edit. Multiple
	// candidates have no stable wire identity: reject rather than guess by order.
	for i := range incoming {
		if incoming[i].matched {
			continue
		}
		var remaining []VCardEmployment
		for _, job := range projected {
			if !used[job.JobID] && job.Company == incoming[i].name {
				remaining = append(remaining, job)
			}
		}
		if len(remaining) == 0 {
			continue
		}
		unmatched := 0
		for _, entry := range incoming {
			if !entry.matched && entry.name == incoming[i].name {
				unmatched++
			}
		}
		if len(remaining) != 1 || unmatched != 1 {
			return ErrVCardUnsafeReplacement
		}
		// If an identical old role was reserved above, the wire cannot tell which
		// of those IDs was edited. Reserving one arbitrarily must not manufacture
		// a unique residual identity for the other.
		for _, job := range projected {
			if used[job.JobID] && job.Company == remaining[0].Company && job.Title == remaining[0].Title {
				return ErrVCardUnsafeReplacement
			}
		}
		incoming[i].jobID = remaining[0].JobID
		incoming[i].matched = true
		incoming[i].standalone = remaining[0].JobID == 0
		used[remaining[0].JobID] = true
	}
	// Only an unmatched company-less value can create a new independent title.
	// At most one can be represented by the legacy profile field; reject the
	// input before updating matched records if it requires guessing or copying.
	var standaloneTitle *string
	for i := range incoming {
		// An unmatched grouped title must not silently replace another role.
		// New contacts can still import a standalone title without an employer.
		if titleRecordsOnly && len(projected) > 0 && !incoming[i].matched {
			return ErrVCardUnsafeReplacement
		}
		if !incoming[i].matched && incoming[i].name == "" {
			incoming[i].standalone = true
		}
		if incoming[i].standalone {
			if standaloneTitle != nil {
				return ErrVCardUnsafeReplacement
			}
			standaloneTitle = &incoming[i].title
		}
	}
	// Reordering must retain the selected company when it is still present.
	// Replacing its ORG, however, selects the new company while preserving the
	// old employment, as the shared import/subscription contract requires.
	var primaryCompanyID, primaryJobID *uint
	for _, job := range projected {
		if job.CompanyID == 0 {
			continue
		}
		if contact.CompanyID != nil && job.CompanyID != *contact.CompanyID {
			continue
		}
		// A selected job with neither visible company nor title has no wire
		// property to omit. Preserve that identity rather than selecting a
		// secondary job on an otherwise unchanged round trip.
		if job.Company == "" && job.Title == "" && contact.PreferredJobID != nil && *contact.PreferredJobID == job.JobID {
			companyID, jobID := job.CompanyID, job.JobID
			primaryCompanyID, primaryJobID = &companyID, &jobID
			break
		}
		for _, entry := range incoming {
			if entry.name == job.Company {
				id := job.CompanyID
				primaryCompanyID = &id
				jobID := job.JobID
				primaryJobID = &jobID
				break
			}
		}
		if primaryCompanyID != nil {
			break
		}
	}
	for _, entry := range incoming {
		if entry.standalone {
			continue
		}
		var companyID uint
		if entry.jobID != 0 {
			for _, job := range projected {
				if job.JobID != entry.jobID {
					continue
				}
				companyID = job.CompanyID
				if job.Title != entry.title {
					updated := models.ContactCompany{ID: job.JobID, ContactID: contactID, JobPosition: strPtrOrNil(entry.title)}
					if err := updateContactRecord(tx, &updated, contactID, vaultID, ErrContactJobNotFound, "job_position"); err != nil {
						return err
					}
				}
				break
			}
		} else {
			// Company names are not identities. Reuse the associated company even
			// when its stored name has whitespace; never choose between two local
			// same-name company identities for an unidentifiable new role.
			for _, job := range projected {
				if job.Company != entry.name {
					continue
				}
				if companyID != 0 && companyID != job.CompanyID {
					return ErrVCardUnsafeReplacement
				}
				companyID = job.CompanyID
			}
			if companyID == 0 {
				// A new association has no prior identity to disambiguate equal
				// names. Never attach it to an arbitrary company (or make a copy).
				var err error
				companyID, err = resolveVCardCompany(tx, vaultID, entry.name)
				if err != nil {
					return err
				}
			}
			job := models.ContactCompany{ContactID: contactID, CompanyID: companyID, JobPosition: strPtrOrNil(entry.title)}
			if err := createContactRecord(tx, &job, contactID, vaultID); err != nil {
				return err
			}
			entry.jobID = job.ID
		}
		if primaryCompanyID == nil {
			id := companyID
			primaryCompanyID = &id
			jobID := entry.jobID
			primaryJobID = &jobID
		}
	}
	if len(incoming) == 0 {
		primaryCompanyID = nil
	}
	if titleRecordsOnly && len(projected) > 0 && standaloneTitle == nil {
		// Omitting a Job preserves both its row and the selected employment.
		// Only full replacements remove an omitted independent title; partial
		// imports have no authority to delete that field (#297).
		if replace && hasStandaloneEmployment(contact) {
			contact.JobPosition = nil
			return updateContactProfile(tx, contact, vaultID, "job_position")
		}
		return nil
	}
	contact.CompanyID = primaryCompanyID
	contact.PreferredJobID = primaryJobID
	if standaloneTitle != nil {
		contact.CompanyID, contact.PreferredJobID = nil, nil
		contact.JobPosition = strPtrOrNil(*standaloneTitle)
		return updateContactProfile(tx, contact, vaultID, "company_id", "preferred_job_id", "job_position")
	}
	return updateContactProfile(tx, contact, vaultID, "company_id", "preferred_job_id")
}

// vCard group names are case-insensitive, just like property names.
func vcardOrganizationTitle(card vcard.Card, group string) (*vcard.Field, error) {
	var match *vcard.Field
	for _, title := range card[vcard.FieldTitle] {
		if strings.EqualFold(title.Group, group) {
			if match != nil {
				return nil, ErrVCardUnsafeReplacement
			}
			match = title
		}
	}
	if match != nil {
		return match, nil
	}
	// Some clients label only one side of a unique ORG/TITLE pair.
	// Distinct explicit groups identify an independent TITLE, even when there
	// is just one employer without a title. Do not pair those on a round trip.
	if len(card[vcard.FieldOrganization]) == 1 && len(card[vcard.FieldTitle]) == 1 &&
		(group == "" || card[vcard.FieldTitle][0].Group == "") {
		return card[vcard.FieldTitle][0], nil
	}
	return nil, nil
}

// effectiveVCardOrganizations interprets the same structured company component
// for full replacements and incremental imports. The caller's card is immutable.
func effectiveVCardOrganizations(card vcard.Card) []*vcard.Field {
	var organizations []*vcard.Field
	for _, field := range card[vcard.FieldOrganization] {
		components := field.Organization()
		if len(components) > 0 && strings.TrimSpace(components[0]) != "" {
			organizations = append(organizations, field)
		}
	}
	return organizations
}

// Match the exact Unicode whitespace normalization used by the export
// projection. SQL TRIM differs between databases and misses Unicode whitespace;
// normalizing only incoming ORG would create copies of existing companies.
// Read just identity/name in bounded batches; stop at the second match rather
// than loading every company or retaining an unbounded candidate collection.
// The caller holds lockCompanyResolution for the entire transaction.
func resolveVCardCompany(tx *gorm.DB, vaultID, name string) (uint, error) {
	const batchSize = 128
	var afterID, candidateID uint
	for {
		var companies []models.Company
		if err := tx.Select("id", "name").Where("vault_id = ? AND id > ?", vaultID, afterID).Order("id").Limit(batchSize).Find(&companies).Error; err != nil {
			return 0, err
		}
		for _, company := range companies {
			if strings.TrimSpace(company.Name) == name {
				if candidateID != 0 {
					return 0, ErrVCardUnsafeReplacement
				}
				candidateID = company.ID
			}
		}
		if len(companies) < batchSize {
			break
		}
		afterID = companies[len(companies)-1].ID
	}
	if candidateID != 0 {
		return candidateID, nil
	}
	company := models.Company{VaultID: vaultID, Name: name}
	if err := createResolvedCompany(tx, &company); err != nil {
		return 0, err
	}
	return company.ID, nil
}

// A lone ungrouped TITLE is the established legacy selected-role edit. Grouped
// or multiple TITLE fields describe employment records and must match first.
func isUngroupedTitleEdit(card vcard.Card) bool {
	return len(card[vcard.FieldOrganization]) == 0 && len(card[vcard.FieldTitle]) == 1 && card[vcard.FieldTitle][0].Group == ""
}
