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
func reconcileVCardOrganization(tx *gorm.DB, card vcard.Card, contactID, vaultID string) error {
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
	// Group labels without ORG do not identify local jobs. Never partially
	// accept multiple title-only properties by choosing the first one.
	if len(card[vcard.FieldOrganization]) == 0 && len(card[vcard.FieldTitle]) > 1 {
		return ErrVCardUnsafeReplacement
	}
	if len(card[vcard.FieldOrganization]) == 0 {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		contact, err := loadContactEmployment(tx, contactID, vaultID)
		if err != nil {
			return err
		}
		// Clearing the independent title must not select and overwrite a remaining
		// Job merely because the cleared title no longer appears in the projection.
		if hasStandaloneEmployment(contact) {
			contact.JobPosition = strPtrOrNil(strings.TrimSpace(card.Value(vcard.FieldTitle)))
			return updateContactProfile(tx, contact, vaultID, "job_position")
		}
	}
	if err := reconcileVCardEmploymentRecords(tx, card, contactID, vaultID); err != nil {
		return err
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
	} else if len(card[vcard.FieldOrganization]) == 0 {
		// Standalone titles use the same blank-value semantics as the projection.
		contact.JobPosition = strPtrOrNil(strings.TrimSpace(card.Value(vcard.FieldTitle)))
	}
	return updateContactProfile(tx, contact, vaultID, "company_id", "job_position", "preferred_job_id")
}

func reconcileVCardEmploymentRecords(tx *gorm.DB, card vcard.Card, contactID, vaultID string) error {
	organizations := card[vcard.FieldOrganization]
	if len(organizations) == 0 && len(card[vcard.FieldTitle]) == 0 {
		return nil
	}
	if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
		return err
	}
	contact, err := loadContactEmployment(tx, contactID, vaultID)
	if err != nil {
		return err
	}
	// One transaction-scoped lock per Vault serializes automatic lookup/create
	// across contacts and processes. A name uniqueness constraint would forbid
	// legitimate explicit duplicates; per-name locks could deadlock when batch
	// imports encounter several names in different orders. SQLite serializes writes.
	if len(organizations) > 0 && tx.Dialector.Name() == "postgres" {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "bonds:vcard-company:"+vaultID).Error; err != nil {
			return err
		}
	}
	if err := materializeLegacyEmployment(tx, contact); err != nil {
		return err
	}
	if len(organizations) == 0 {
		// A title-only edit targets the first projected job, not a stale profile copy.
		employment := ContactVCardEmployments(contact)
		if len(employment) > 0 && employment[0].JobID != 0 {
			job := models.ContactCompany{ID: employment[0].JobID, ContactID: contactID, JobPosition: strPtrOrNil(card.Value(vcard.FieldTitle))}
			return updateContactRecord(tx, &job, contactID, vaultID, ErrContactJobNotFound, "job_position")
		}
		return nil
	}
	// Resolve the entire input before writing. A greedy per-field match lets an
	// inserted/edited first role consume an unchanged later role's identity.
	type incomingEmployment struct {
		name, title string
		jobID       uint
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
	// A distinct group can identify one company-less TITLE alongside Jobs.
	// Ungrouped or multiple unmatched titles remain ambiguous; never guess which
	// company they belong to. Export emits the same distinct groups on round trips.
	var standaloneTitle *string
	if len(incoming) > 0 {
		for _, title := range card[vcard.FieldTitle] {
			if matchedTitles[title] || strings.TrimSpace(title.Value) == "" {
				continue
			}
			if title.Group == "" || standaloneTitle != nil {
				return ErrVCardUnsafeReplacement
			}
			for _, org := range organizations {
				if strings.EqualFold(org.Group, title.Group) {
					return ErrVCardUnsafeReplacement
				}
			}
			value := strings.TrimSpace(title.Value)
			standaloneTitle = &value
		}
	}
	if len(incoming) == 0 && (len(contact.ContactCompanies) > 0 || contact.CompanyID != nil) {
		return ErrVCardUnsafeReplacement
	}
	projected := ContactVCardEmployments(contact)
	used := map[uint]bool{}
	for i := range incoming {
		for _, job := range projected {
			if job.JobID != 0 && !used[job.JobID] && job.Company == incoming[i].name && job.Title == incoming[i].title {
				incoming[i].jobID = job.JobID
				used[job.JobID] = true
				break
			}
		}
	}
	// Exact matches (including empty titles) are reserved first. Within a company
	// only one remaining input and one remaining job identify an edit. Multiple
	// candidates have no stable wire identity: reject rather than guess by order.
	for i := range incoming {
		if incoming[i].jobID != 0 {
			continue
		}
		var remaining []VCardEmployment
		for _, job := range projected {
			if job.JobID != 0 && !used[job.JobID] && job.Company == incoming[i].name {
				remaining = append(remaining, job)
			}
		}
		if len(remaining) == 0 {
			continue
		}
		unmatched := 0
		for _, entry := range incoming {
			if entry.jobID == 0 && entry.name == incoming[i].name {
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
		used[remaining[0].JobID] = true
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
				var companies []models.Company
				if err := tx.Where("vault_id = ? AND name = ?", vaultID, entry.name).Limit(2).Find(&companies).Error; err != nil {
					return err
				}
				if len(companies) > 1 {
					return ErrVCardUnsafeReplacement
				}
				if len(companies) == 0 {
					company := models.Company{VaultID: vaultID, Name: entry.name}
					if err := tx.Create(&company).Error; err != nil {
						return err
					}
					companyID = company.ID
				} else {
					companyID = companies[0].ID
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
	contact.CompanyID = primaryCompanyID
	contact.PreferredJobID = primaryJobID
	if standaloneTitle != nil {
		contact.CompanyID, contact.PreferredJobID = nil, nil
		contact.JobPosition = standaloneTitle
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
