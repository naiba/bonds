package services

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrContactMergeSelection = errors.New("select a target and between 1 and 49 distinct source contacts")

// MergeContacts applies a reviewed field selection without storing profile copies.
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
	// Drain queued local DAV writers before inspecting subscription mappings.
	// Pausing a subscription does not cancel a PUT that is already in flight.
	if s.davPushService != nil {
		release := lockMovedContactDAVOperations(&s.davPushService.operationLocks, ids)
		defer release()
	}
	var target models.Contact
	var result dto.ContactResponse
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
		review, err := buildContactMergePreview(tx, vaultID, userID, req, contacts)
		if err != nil {
			return err
		}
		if len(review.Blockers) != 0 {
			return ErrContactMergeBlocked
		}
		// Recheck inside the transaction: a preview is not authority to overwrite
		// values or references changed since the user reviewed them.
		if req.ReviewToken == "" || req.ReviewToken != review.ReviewToken {
			return ErrContactMergeReviewChanged
		}
		target = byID[req.TargetContactID]
		if err := applyContactMergeChoices(&target, byID, review.Fields, req.FieldChoices); err != nil {
			return err
		}
		for _, sourceID := range req.SourceContactIDs {
			source := byID[sourceID]
			fillMergedContactProfile(&target, &source)
			if err := mergeContactAssociations(tx, source.ID, target.ID); err != nil {
				return err
			}
		}
		for _, activity := range review.payerChanges {
			result := tx.Model(&models.Activity{}).Where("id = ? AND vault_id = ? AND paid_by_contact_id = ?", activity.ID, activity.VaultID, activity.PaidByContactID).Update("paid_by_contact_id", target.ID)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrContactMergeReviewChanged
			}
		}
		for _, change := range review.relationshipChanges {
			if change.remove {
				if err := tx.Delete(&change.original).Error; err != nil {
					return err
				}
			} else if err := tx.Model(&models.Relationship{}).Where("id = ?", change.original.ID).Updates(map[string]any{"contact_id": change.contactID, "related_contact_id": change.relatedID}).Error; err != nil {
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
		for _, sourceID := range req.SourceContactIDs {
			source := byID[sourceID]
			if err := tx.Delete(&source).Error; err != nil {
				return err
			}
		}
		if err := NewFeedRecorder(tx).Record(target.ID, userID, ActionContactUpdated, fmt.Sprintf("Merged %d contacts after field review", len(req.SourceContactIDs)), nil, nil); err != nil {
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
	// Only committed data may reach external indexes. DAV-linked merges are blocked.
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

	return &result, nil
}

func fillMergedContactProfile(target, source *models.Contact) {
	if source.LastTalkedTo != nil && (target.LastTalkedTo == nil || source.LastTalkedTo.After(*target.LastTalkedTo)) {
		target.LastTalkedTo = source.LastTalkedTo
	}
	target.NeedsVerification = target.NeedsVerification || source.NeedsVerification
	target.ShowQuickFacts = target.ShowQuickFacts || source.ShowQuickFacts
	// Target visibility, template and remote identity remain authoritative.
}
