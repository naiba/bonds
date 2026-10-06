package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrContactMergeSelection = errors.New("select a target and between 1 and 49 distinct source contacts")

// MergeContacts deliberately uses existing notes for original profiles: merging
// needs neither a second contact identity system nor an ever-growing undo schema.
func (s *ContactService) MergeContacts(vaultID, userID string, req dto.MergeContactsRequest) (*dto.ContactResponse, error) {
	ids := append([]string{req.TargetContactID}, req.SourceContactIDs...)
	seen := make(map[string]bool, len(ids))
	if len(ids) < 2 || len(ids) > 50 {
		return nil, ErrContactMergeSelection
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || seen[id] {
			return nil, ErrContactMergeSelection
		}
		seen[id] = true
	}
	var releaseDAV func()
	if s.davPushService != nil {
		releaseDAV = lockMovedContactDAVOperations(&s.davPushService.operationLocks, ids)
	}
	workerOwnsLocks := false
	defer func() {
		if releaseDAV != nil && !workerOwnsLocks {
			releaseDAV()
		}
	}()
	var target models.Contact
	var result dto.ContactResponse
	var remoteTargets []contactRemoteDeletionTarget
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := NewVaultService(tx).CheckUserVaultAccess(userID, vaultID, models.PermissionEditor); err != nil {
			return err
		}
		// Stable row locking prevents overlapping PostgreSQL merges from consuming
		// the same source twice. SQLite serializes writes and rolls back a loser.
		var contacts []models.Contact
		if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("vault_id = ? AND id IN ?", vaultID, ids).Order("id ASC").Find(&contacts).Error; err != nil {
			return err
		}
		if len(contacts) != len(ids) {
			return ErrContactNotFound
		}
		byID := make(map[string]models.Contact, len(contacts))
		for _, contact := range contacts {
			byID[contact.ID] = contact
		}
		target = byID[req.TargetContactID]
		for _, sourceID := range req.SourceContactIDs {
			source := byID[sourceID]
			if !source.CanBeDeleted {
				return ErrContactCannotBeDeleted
			}
			if err := recordMergedContactProfile(tx, source, target.ID, userID); err != nil {
				return err
			}
		}
		// Capture every source before redirecting any relationships; otherwise
		// later snapshots would describe partially merged data instead of originals.
		for _, sourceID := range req.SourceContactIDs {
			source := byID[sourceID]
			fillMergedContactProfile(&target, &source)
			if err := mergeContactAssociations(tx, source.ID, target.ID); err != nil {
				return err
			}
		}
		if target.FirstMetThroughContactID != nil && seen[*target.FirstMetThroughContactID] {
			target.FirstMetThroughContactID = nil
		}
		target.StayInTouchTriggerDate = calculateStayInTouchTriggerDate(target.LastTalkedTo, target.StayInTouchFrequencyDays)
		if err := tx.Omit(clause.Associations).Save(&target).Error; err != nil {
			return err
		}
		// Redirect inbound references before soft deletion invokes Contact.BeforeDelete.
		if err := tx.Model(&models.Contact{}).Where("first_met_through_contact_id IN ? AND id NOT IN ?", req.SourceContactIDs, ids).Update("first_met_through_contact_id", target.ID).Error; err != nil {
			return err
		}
		var states []models.ContactSubscriptionState
		if err := tx.Where("contact_id IN ?", req.SourceContactIDs).Find(&states).Error; err != nil {
			return err
		}
		for _, state := range states {
			remoteTargets = append(remoteTargets, newContactRemoteDeletionTarget(state))
		}
		if err := tx.Where("contact_id IN ?", req.SourceContactIDs).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
			return err
		}
		for _, sourceID := range req.SourceContactIDs {
			source := byID[sourceID]
			if err := tx.Delete(&source).Error; err != nil {
				return err
			}
		}
		if err := NewFeedRecorder(tx).Record(target.ID, userID, ActionContactUpdated, fmt.Sprintf("Merged %d contacts; original profiles are saved in notes", len(req.SourceContactIDs)), nil, nil); err != nil {
			return err
		}
		formatter, err := newContactNameFormatter(tx, userID)
		if err != nil {
			return err
		}
		if err := reloadContactWithSameVaultFirstMetThrough(tx, &target, vaultID); err != nil {
			return err
		}
		var favorites int64
		if err := tx.Model(&models.ContactVaultUser{}).Where("contact_id = ? AND user_id = ? AND is_favorite = ?", target.ID, userID, true).Count(&favorites).Error; err != nil {
			return err
		}
		result, err = toContactResponse(&target, favorites > 0, formatter)
		return err
	})
	if err != nil {
		return nil, err
	}
	// Only committed data may reach external indexes or CardDAV servers.
	if s.searchService != nil {
		for _, sourceID := range req.SourceContactIDs {
			if err := s.searchService.DeleteContact(sourceID); err != nil {
				log.Printf("[contact-merge] search removal: %v", err)
			}
		}
		if err := s.searchService.IndexContact(&target); err != nil {
			log.Printf("[contact-merge] search contact: %v", err)
		}
		var notes []models.Note
		if err := s.db.Where("contact_id = ?", target.ID).Find(&notes).Error; err != nil {
			log.Printf("[contact-merge] load search notes: %v", err)
		} else {
			for i := range notes {
				if err := s.searchService.IndexNote(&notes[i]); err != nil {
					log.Printf("[contact-merge] search note: %v", err)
				}
			}
		}
	}
	if s.davPushService != nil {
		workerOwnsLocks = true
		go func() {
			defer releaseDAV()
			s.davPushService.pushCapturedContactDelete(remoteTargets)
			s.davPushService.pushContactChange(target.ID, vaultID)
		}()
	}
	return &result, nil
}

func fillMergedContactProfile(target, source *models.Contact) {
	fillMergeString(&target.FirstName, source.FirstName)
	fillMergeString(&target.MiddleName, source.MiddleName)
	fillMergeString(&target.LastName, source.LastName)
	fillMergeString(&target.Nickname, source.Nickname)
	fillMergeString(&target.MaidenName, source.MaidenName)
	fillMergeString(&target.Prefix, source.Prefix)
	fillMergeString(&target.Suffix, source.Suffix)
	fillMergeString(&target.Description, source.Description)
	fillMergeString(&target.FoodPreferences, source.FoodPreferences)
	fillMergePointer(&target.GenderID, source.GenderID)
	fillMergePointer(&target.PronounID, source.PronounID)
	fillMergePointer(&target.ReligionID, source.ReligionID)
	fillMergePointer(&target.FileID, source.FileID)
	// Company/position and first-met date parts are units; filling each component
	// independently could manufacture a job or date that nobody entered.
	if target.CompanyID == nil && ptrToStr(target.JobPosition) == "" {
		target.CompanyID, target.JobPosition = source.CompanyID, source.JobPosition
	}
	if target.FirstMetAt == nil && target.FirstMetYear == nil && target.FirstMetMonth == nil && target.FirstMetDay == nil {
		target.FirstMetAt, target.FirstMetDatePrecision = source.FirstMetAt, source.FirstMetDatePrecision
		target.FirstMetYear, target.FirstMetMonth, target.FirstMetDay = source.FirstMetYear, source.FirstMetMonth, source.FirstMetDay
	}
	fillMergePointer(&target.FirstMetThroughContactID, source.FirstMetThroughContactID)
	fillMergePointer(&target.StayInTouchFrequencyDays, source.StayInTouchFrequencyDays)
	if source.LastTalkedTo != nil && (target.LastTalkedTo == nil || source.LastTalkedTo.After(*target.LastTalkedTo)) {
		target.LastTalkedTo = source.LastTalkedTo
	}
	target.NeedsVerification = target.NeedsVerification || source.NeedsVerification
	target.ShowQuickFacts = target.ShowQuickFacts || source.ShowQuickFacts
	// Target visibility, template and remote identity remain authoritative.
}

func fillMergePointer[T any](target **T, source *T) {
	if *target == nil {
		*target = source
	}
}
func fillMergeString(target **string, source *string) {
	if *target == nil || strings.TrimSpace(**target) == "" {
		*target = source
	}
}

func recordMergedContactProfile(tx *gorm.DB, source models.Contact, targetID, userID string) error {
	// Query scalar columns only: serializing the Contact model would also include
	// empty association objects and obscure the original profile.
	profile := map[string]any{}
	if err := tx.Model(&models.Contact{}).Where("id = ?", source.ID).Take(&profile).Error; err != nil {
		return err
	}
	keys := make([]string, 0, len(profile))
	for key, value := range profile {
		if value != nil && value != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var body strings.Builder
	fmt.Fprintf(&body, "Original profile: %s %s\n\n", ptrToStr(source.FirstName), ptrToStr(source.LastName))
	for _, key := range keys {
		value, err := json.Marshal(profile[key])
		if err != nil {
			return err
		}
		fmt.Fprintf(&body, "%s: %s\n", strings.ReplaceAll(key, "_", " "), value)
	}
	// Group roles can conflict when both contacts were in the same group.
	var groups []models.ContactGroup
	if err := tx.Where("contact_id = ?", source.ID).Find(&groups).Error; err != nil {
		return err
	}
	for _, group := range groups {
		if group.GroupTypeRoleID != nil {
			fmt.Fprintf(&body, "Group %d, role %d\n", group.GroupID, *group.GroupTypeRoleID)
		}
	}
	var relationships []models.Relationship
	if err := tx.Where("contact_id = ? OR (related_contact_id = ? AND contact_id = ?)", source.ID, source.ID, targetID).Find(&relationships).Error; err != nil {
		return err
	}
	for _, relationship := range relationships {
		fmt.Fprintf(&body, "Relationship from %s to %s, type %d\n", relationship.ContactID, relationship.RelatedContactID, relationship.RelationshipTypeID)
	}
	title := "Merged contact: " + strings.TrimSpace(ptrToStr(source.FirstName)+" "+ptrToStr(source.LastName))
	kind := "contact_merge"
	return tx.Create(&models.Note{ContactID: targetID, VaultID: source.VaultID, AuthorID: &userID, Title: &title, Body: body.String(), BodyFormat: "plain", SourceType: &kind, SourceUUID: &source.ID}).Error
}
