package services

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errContactMergeIntroducerUnavailable = errors.New("contact introducer is unavailable in this vault")

var ErrContactMergeBlocked = errors.New("contact merge has unresolved safety restrictions")
var ErrContactMergeReviewChanged = errors.New("contact merge requires a current review and a choice for every conflict")

type contactMergeReview struct {
	*dto.ContactMergePreview
	relationshipChanges []contactMergeRelationshipChange
}

func (s *ContactService) PreviewContactMerge(vaultID, userID string, req dto.MergeContactsRequest) (*dto.ContactMergePreview, error) {
	ids := append([]string{req.TargetContactID}, req.SourceContactIDs...)
	if len(ids) < 2 || len(ids) > 50 {
		return nil, ErrContactMergeSelection
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || seen[id] {
			return nil, ErrContactMergeSelection
		}
		seen[id] = true
	}
	var result *dto.ContactMergePreview
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := NewVaultService(tx).CheckUserVaultAccess(userID, vaultID, models.PermissionEditor); err != nil {
			return err
		}
		var contacts []models.Contact
		if err := tx.Where("vault_id = ? AND id IN ?", vaultID, ids).Order("id ASC").Find(&contacts).Error; err != nil {
			return err
		}
		if len(contacts) != len(ids) {
			return ErrContactNotFound
		}
		review, err := buildContactMergePreview(tx, vaultID, userID, req, contacts)
		if err != nil {
			return err
		}
		result = review.ContactMergePreview
		return nil
	})
	return result, err
}

func buildContactMergePreview(tx *gorm.DB, vaultID, userID string, req dto.MergeContactsRequest, contacts []models.Contact) (*contactMergeReview, error) {
	ids := append([]string{req.TargetContactID}, req.SourceContactIDs...)
	byID := map[string]models.Contact{}
	review := &contactMergeReview{ContactMergePreview: &dto.ContactMergePreview{Contacts: []dto.ContactMergeCandidate{}, Fields: []dto.ContactMergeField{}, Effects: map[string]int64{"removed_contacts": int64(len(req.SourceContactIDs))}, Blockers: []string{}}}
	for _, contact := range contacts {
		byID[contact.ID] = contact
	}
	formatter, err := newContactNameFormatter(tx, userID)
	if err != nil {
		return nil, err
	}
	fields := map[string]int{}
	identities := map[string]bool{}
	for _, id := range ids {
		contact := byID[id]
		if id != req.TargetContactID && !contact.CanBeDeleted {
			review.Blockers = append(review.Blockers, "protected")
		}
		if ptrToStr(contact.DistantURI) != "" {
			review.Blockers = append(review.Blockers, "dav_linked")
		}
		if ptrToStr(contact.Vcard) != "" {
			review.Blockers = append(review.Blockers, "raw_vcard")
		}
		template := ""
		if contact.TemplateID != nil {
			var row models.VaultContactTemplate
			if err := tx.First(&row, *contact.TemplateID).Error; err != nil {
				return nil, err
			}
			template = row.Name
		}
		name, err := formatter.format(&contact, ptrToStr(contact.Nickname))
		if err != nil {
			return nil, err
		}
		review.Contacts = append(review.Contacts, dto.ContactMergeCandidate{ID: id, Name: name, Listed: contact.Listed, Template: template})
		values, err := contactMergeProfileFields(tx, contact)
		if errors.Is(err, errContactMergeIntroducerUnavailable) {
			review.Blockers = append(review.Blockers, "introducer_unavailable")
		} else if err != nil {
			return nil, err
		}
		for _, value := range values {
			if value.value == "" {
				continue
			}
			index, ok := fields[value.key]
			if !ok {
				index = len(review.Fields)
				fields[value.key] = index
				review.Fields = append(review.Fields, dto.ContactMergeField{Key: value.key, Options: []dto.ContactMergeOption{}})
			}
			field := &review.Fields[index]
			// Equal values need no decision. The first option is the target, or first
			// nonempty source, and remains the documented fill order.
			selectedProfile := models.Contact{}
			value.apply(&selectedProfile)
			identityJSON, _ := json.Marshal(selectedProfile)
			identity := value.key + string(identityJSON)
			duplicate := identities[identity]
			identities[identity] = true
			if !duplicate {
				field.Options = append(field.Options, dto.ContactMergeOption{ContactID: id, Value: value.value})
			}
			field.Conflict = len(field.Options) > 1
		}
	}
	// Pull replaces contact fields and remote deletion is not transactional with
	// local merging. Even disabled/read-only links can resume later. Do not erase
	// their mapping or attempt DELETE-before-PUT as a best-effort merge.
	var linked, pushing int64
	if err := tx.Model(&models.ContactSubscriptionState{}).Where("contact_id IN ?", ids).Count(&linked).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&models.AddressBookSubscription{}).Where("vault_id = ? AND active = ? AND (sync_way & ?) != 0", vaultID, true, SyncWayPush).Count(&pushing).Error; err != nil {
		return nil, err
	}
	if linked > 0 {
		review.Blockers = append(review.Blockers, "dav_linked")
	}
	if pushing > 0 {
		review.Blockers = append(review.Blockers, "dav_push")
	}
	var relationships []models.Relationship
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("contact_id IN ? OR related_contact_id IN ?", ids, ids).Order("id ASC").Find(&relationships).Error; err != nil {
		return nil, err
	}
	// Lock existing rows and carry this exact plan into the write phase. A
	// second query after authorization must not expand the set of edited owners.
	changes := planContactMergeRelationships(relationships, req.TargetContactID, req.SourceContactIDs)
	review.relationshipChanges = changes
	ownerIDs := []string{}
	for _, change := range changes {
		ownerIDs = append(ownerIDs, change.original.ContactID)
		if change.remove {
			if change.contactID == change.relatedID {
				review.Effects["self_relationships"]++
			} else {
				review.Effects["duplicate_relationships"]++
			}
		} else {
			review.Effects["redirected_relationships"]++
		}
	}
	// Check the original owning vault of every row that will change, including
	// soft-deleted owners. Unchanged target references do not grant or require
	// permission. Only live introducer references are actually redirected.
	var owners []models.Contact
	if err := tx.Unscoped().Where("id IN ? OR (deleted_at IS NULL AND first_met_through_contact_id IN ?)", ownerIDs, req.SourceContactIDs).Order("id ASC").Find(&owners).Error; err != nil {
		return nil, err
	}

	checked := map[string]bool{vaultID: true}
	for _, owner := range owners {
		if checked[owner.VaultID] {
			continue
		}
		if err := NewVaultService(tx).CheckUserVaultAccess(userID, owner.VaultID, models.PermissionEditor); err != nil {
			if errors.Is(err, ErrVaultForbidden) || errors.Is(err, ErrInsufficientPerm) {
				review.Blockers = append(review.Blockers, "incoming_permission")
			} else {
				return nil, err
			}
		}
		checked[owner.VaultID] = true
	}
	for _, entry := range []struct{ key, table, column string }{
		{"notes", "notes", "contact_id"}, {"contact_information", "contact_information", "contact_id"},
		{"addresses", "contact_address", "contact_id"}, {"reminders", "contact_reminders", "contact_id"},
		{"dates", "contact_important_dates", "contact_id"}, {"tasks", "task_contacts", "contact_id"},
		{"activities", "activity_participants", "contact_id"}, {"files", "files", "ufileable_id"},
		{"groups", "contact_group", "contact_id"}, {"calls", "calls", "contact_id"}, {"pets", "pets", "contact_id"},
		{"goals", "goals", "contact_id"}, {"gifts", "gifts", "contact_id"}, {"quick_facts", "quick_facts", "contact_id"},
		{"jobs", "contact_companies", "contact_id"}, {"life_events", "contact_life_metric", "contact_id"}, {"history", "contact_feed_items", "contact_id"}, {"payers", "activities", "paid_by_contact_id"}, {"posts", "contact_post", "contact_id"}, {"labels", "contact_label", "contact_id"},
	} {
		var count int64
		if err := tx.Table(entry.table).Where(entry.column+" IN ?", req.SourceContactIDs).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			review.Effects[entry.key] = count
		}
	}
	for _, entry := range []struct {
		key   string
		model any
	}{{"loans", &models.ContactLoan{}}, {"gift_parties", &models.ContactGift{}}} {
		var count int64
		if err := tx.Model(entry.model).Where("loaner_id IN ? OR loanee_id IN ?", req.SourceContactIDs, req.SourceContactIDs).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			review.Effects[entry.key] = count
		}
	}
	var groups []models.ContactGroup
	if err := tx.Where("contact_id IN ?", ids).Order("id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	roles := map[uint]uint{}
	for _, group := range groups {
		if group.GroupTypeRoleID == nil {
			continue
		}
		if previous, ok := roles[group.GroupID]; ok && previous != *group.GroupTypeRoleID {
			review.Blockers = append(review.Blockers, "group_roles")
		}
		roles[group.GroupID] = *group.GroupTypeRoleID
	}

	if err := tx.Model(&models.Contact{}).Where("first_met_through_contact_id IN ? AND id NOT IN ?", req.SourceContactIDs, ids).Count(&linked).Error; err != nil {
		return nil, err
	}
	if linked > 0 {
		review.Effects["introducers"] = linked
	}
	for _, contact := range contacts {
		if contact.FirstMetThroughContactID != nil {
			if _, ok := byID[*contact.FirstMetThroughContactID]; ok {
				review.Effects["self_introducers"]++
			}
		}
	}
	var dates []models.ContactImportantDate
	if err := tx.Where("contact_id IN ?", ids).Order("id ASC").Find(&dates).Error; err != nil {
		return nil, err
	}
	review.Dates = []dto.ImportantDateResponse{}
	for i := range dates {
		review.Dates = append(review.Dates, toImportantDateResponse(&dates[i]))
	}
	// Hashes are only concurrency guards, never persisted snapshots. Include all
	// contact versions and affected reference contents, even if counts are equal.
	payload, err := json.Marshal([]any{review, contacts, owners, groups, relationships})
	if err != nil {
		return nil, err
	}
	review.ReviewToken = fmt.Sprintf("%x", sha256.Sum256(payload))
	return review, nil
}

type contactMergeProfileField struct {
	key, value string
	apply      func(*models.Contact)
}

func applyContactMergeChoices(target *models.Contact, contacts map[string]models.Contact, fields []dto.ContactMergeField, choices map[string]string) error {
	for key := range choices {
		found := false
		for _, field := range fields {
			if field.Key == key {
				found = true
			}
		}
		if !found {
			return ErrContactMergeReviewChanged
		}
	}
	for _, field := range fields {
		selected := field.Options[0].ContactID
		if field.Conflict {
			selected = choices[field.Key]
		}
		valid := false
		for _, option := range field.Options {
			if option.ContactID == selected {
				valid = true
			}
		}
		if !valid {
			return ErrContactMergeReviewChanged
		}
		// The allowlist below is shared with the preview. Internal metadata can
		// never be selected, serialized into a note, or mass-assigned by the client.
		values, _ := contactMergeProfileFields(nil, contacts[selected])
		for _, value := range values {
			if value.key == field.Key {
				value.apply(target)
			}
		}
	}
	return nil
}

func contactMergeProfileFields(tx *gorm.DB, c models.Contact) ([]contactMergeProfileField, error) {
	fields := []contactMergeProfileField{}
	var referenceErr error
	add := func(key, value string, apply func(*models.Contact)) {
		fields = append(fields, contactMergeProfileField{key, strings.TrimSpace(value), apply})
	}
	add("first_name", ptrToStr(c.FirstName), func(t *models.Contact) { t.FirstName = c.FirstName })
	add("middle_name", ptrToStr(c.MiddleName), func(t *models.Contact) { t.MiddleName = c.MiddleName })
	add("last_name", ptrToStr(c.LastName), func(t *models.Contact) { t.LastName = c.LastName })
	add("nickname", ptrToStr(c.Nickname), func(t *models.Contact) { t.Nickname = c.Nickname })
	add("maiden_name", ptrToStr(c.MaidenName), func(t *models.Contact) { t.MaidenName = c.MaidenName })
	add("prefix", ptrToStr(c.Prefix), func(t *models.Contact) { t.Prefix = c.Prefix })
	add("suffix", ptrToStr(c.Suffix), func(t *models.Contact) { t.Suffix = c.Suffix })
	add("description", ptrToStr(c.Description), func(t *models.Contact) { t.Description = c.Description })
	add("food_preferences", ptrToStr(c.FoodPreferences), func(t *models.Contact) { t.FoodPreferences = c.FoodPreferences })
	if c.GenderID != nil {
		value := fmt.Sprint(*c.GenderID)
		if tx != nil {
			var row models.Gender
			if err := tx.First(&row, *c.GenderID).Error; err != nil {
				return nil, err
			}
			value = ptrToStr(row.Name) + " (#" + value + ")"
		}
		add("gender", value, func(t *models.Contact) { t.GenderID = c.GenderID })
	}
	if c.PronounID != nil {
		value := fmt.Sprint(*c.PronounID)
		if tx != nil {
			var row models.Pronoun
			if err := tx.First(&row, *c.PronounID).Error; err != nil {
				return nil, err
			}
			value = ptrToStr(row.Name) + " (#" + value + ")"
		}
		add("pronoun", value, func(t *models.Contact) { t.PronounID = c.PronounID })
	}
	if c.ReligionID != nil {
		value := fmt.Sprint(*c.ReligionID)
		if tx != nil {
			var row models.Religion
			if err := tx.First(&row, *c.ReligionID).Error; err != nil {
				return nil, err
			}
			value = ptrToStr(row.Name) + " (#" + value + ")"
		}
		add("religion", value, func(t *models.Contact) { t.ReligionID = c.ReligionID })
	}
	if c.FileID != nil {
		value := fmt.Sprint(*c.FileID)
		if tx != nil {
			var row models.File
			if err := tx.First(&row, *c.FileID).Error; err != nil {
				return nil, err
			}
			value = row.Name + " (#" + value + ")"
		}
		add("avatar", value, func(t *models.Contact) { t.FileID = c.FileID })
	}
	if c.CompanyID != nil || ptrToStr(c.JobPosition) != "" {
		value := ptrToStr(c.JobPosition)
		if c.CompanyID != nil && tx != nil {
			var row models.Company
			if err := tx.First(&row, *c.CompanyID).Error; err != nil {
				return nil, err
			}
			value = row.Name + " / " + value
		}
		add("job", value, func(t *models.Contact) { t.CompanyID = c.CompanyID; t.JobPosition = c.JobPosition })
	}
	if c.FirstMetAt != nil || c.FirstMetYear != nil || c.FirstMetMonth != nil || c.FirstMetDay != nil {
		parts := []any{c.FirstMetYear, c.FirstMetMonth, c.FirstMetDay, c.FirstMetAt, c.FirstMetDatePrecision}
		value, _ := json.Marshal(parts)
		add("first_met", string(value), func(t *models.Contact) {
			t.FirstMetAt = c.FirstMetAt
			t.FirstMetYear = c.FirstMetYear
			t.FirstMetMonth = c.FirstMetMonth
			t.FirstMetDay = c.FirstMetDay
			t.FirstMetDatePrecision = c.FirstMetDatePrecision
		})
	}
	if c.FirstMetThroughContactID != nil {
		value := *c.FirstMetThroughContactID
		if tx != nil {
			var row models.Contact
			if err := tx.First(&row, "id = ? AND vault_id = ?", value, c.VaultID).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, err
				}
				// Existing invalid/cross-vault references must be resolved explicitly;
				// hiding an inaccessible name is not permission to discard its link.
				referenceErr = errContactMergeIntroducerUnavailable
				value = ""
			} else {
				value = strings.TrimSpace(ptrToStr(row.FirstName) + " " + ptrToStr(row.LastName))
				// Nickname-only and unnamed contacts are still meaningful references.
				if value == "" {
					value = ptrToStr(row.Nickname)
				}
				if value == "" {
					value = row.ID
				}
			}
		}
		add("introducer", value, func(t *models.Contact) { t.FirstMetThroughContactID = c.FirstMetThroughContactID })
	}
	if c.StayInTouchFrequencyDays != nil {
		add("stay_in_touch", fmt.Sprint(*c.StayInTouchFrequencyDays), func(t *models.Contact) { t.StayInTouchFrequencyDays = c.StayInTouchFrequencyDays })
	}
	return fields, referenceErr
}
