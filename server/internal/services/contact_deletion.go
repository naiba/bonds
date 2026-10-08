package services

import (
	"sort"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contactDeletionWork struct {
	contactID           string
	noteIDs             []uint
	remoteDeleteTargets []contactRemoteDeletionTarget
}

func (s *ContactService) DeleteContact(contactID, vaultID string) error {
	_, err := s.deleteContacts([]string{contactID}, vaultID)
	return err
}

func (s *ContactService) DeleteContacts(contactIDs []string, vaultID string) (*dto.BulkDeleteContactsResponse, error) {
	uniqueContactIDs, err := validateAndDedupeContactIDs(contactIDs)
	if err != nil {
		return nil, err
	}
	if len(uniqueContactIDs) == 0 {
		return nil, ErrContactDeleteEmpty
	}
	deletedCount, err := s.deleteContacts(uniqueContactIDs, vaultID)
	if err != nil {
		return nil, err
	}
	return &dto.BulkDeleteContactsResponse{DeletedCount: deletedCount}, nil
}

func (s *ContactService) deleteContacts(contactIDs []string, vaultID string) (int, error) {
	uniqueContactIDs := dedupeContactIDs(contactIDs)
	if len(uniqueContactIDs) == 0 {
		return 0, ErrContactDeleteEmpty
	}

	var releaseDAVOperations func()
	if s.davPushService != nil {
		releaseDAVOperations = lockMovedContactDAVOperations(&s.davPushService.operationLocks, uniqueContactIDs)
	}
	releaseDAVOperationsAfterCommit := false
	defer func() {
		if releaseDAVOperations != nil && !releaseDAVOperationsAfterCommit {
			releaseDAVOperations()
		}
	}()

	works := make([]contactDeletionWork, 0, len(uniqueContactIDs))
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		contacts, err := loadDeletableContacts(tx, uniqueContactIDs, vaultID)
		if err != nil {
			return err
		}
		for index := range contacts {
			work, err := s.deleteContactInTransaction(tx, &contacts[index])
			if err != nil {
				return err
			}
			works = append(works, work)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	// Search and DAV cannot move into the transaction: neither side effect can roll back with the database.
	for _, work := range works {
		if s.searchService != nil {
			_ = s.searchService.DeleteContact(work.contactID)
			for _, noteID := range work.noteIDs {
				_ = s.searchService.DeleteNote(noteID)
			}
		}
	}
	if s.davPushService != nil {
		remoteDeleteTargets := make([]contactRemoteDeletionTarget, 0)
		for _, work := range works {
			remoteDeleteTargets = append(remoteDeleteTargets, work.remoteDeleteTargets...)
		}
		if len(remoteDeleteTargets) > 0 {
			releaseDAVOperationsAfterCommit = true
			// The worker owns every contact lock after commit so remote deletion cannot race a later push.
			go func() {
				defer releaseDAVOperations()
				s.davPushService.pushCapturedContactDelete(remoteDeleteTargets)
			}()
		}
	}
	return len(works), nil
}

func loadDeletableContacts(tx *gorm.DB, contactIDs []string, vaultID string) ([]models.Contact, error) {
	contacts, err := LockContactDeletionOwners(tx, contactIDs, vaultID)
	if err != nil {
		return nil, err
	}
	for index := range contacts {
		if !contacts[index].CanBeDeleted {
			return nil, ErrContactCannotBeDeleted
		}
	}
	return contacts, nil
}

// LockContactDeletionOwners fixes the live selection and its incoming introducer
// owners in global ID order. Call inside the deletion transaction before child
// writes. HTTP and DAV callers keep their existing authorization and cleanup
// semantics; neither may let the model hook acquire an unreviewed late parent.
func LockContactDeletionOwners(tx *gorm.DB, contactIDs []string, vaultID string) ([]models.Contact, error) {
	lockedContactIDs := append([]string(nil), contactIDs...)
	sort.Strings(lockedContactIDs)

	// The model's delete hook clears incoming introducer references. Discover
	// those parents first and lock the whole set in merge's global ID order;
	// locking just the selection before its referrers can deadlock with merge.
	var referrerIDs []string
	if err := tx.Model(&models.Contact{}).Where("first_met_through_contact_id IN ?", lockedContactIDs).Pluck("id", &referrerIDs).Error; err != nil {
		return nil, err
	}
	ownerIDs := append(append([]string(nil), lockedContactIDs...), referrerIDs...)
	var owners []models.Contact
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id IN ?", ownerIDs).Order("id ASC").Find(&owners).Error; err != nil {
		return nil, err
	}
	selected := make(map[string]bool, len(lockedContactIDs))
	for _, id := range lockedContactIDs {
		selected[id] = true
	}
	locked := make(map[string]bool, len(owners))
	contacts := []models.Contact{}
	for _, owner := range owners {
		locked[owner.ID] = true
		if selected[owner.ID] && owner.VaultID == vaultID {
			contacts = append(contacts, owner)
		}
	}
	if len(contacts) != len(lockedContactIDs) {
		return nil, ErrContactNotFound
	}
	// A referrer can commit between discovery and taking the source lock.
	// Reject without writes instead of taking a newly discovered lower lock.
	var currentReferrers []string
	if err := tx.Model(&models.Contact{}).Where("first_met_through_contact_id IN ?", lockedContactIDs).Pluck("id", &currentReferrers).Error; err != nil {
		return nil, err
	}
	for _, id := range currentReferrers {
		if !locked[id] {
			return nil, ErrContactDeleteChanged
		}
	}
	return contacts, nil
}

func (s *ContactService) deleteContactInTransaction(tx *gorm.DB, contact *models.Contact) (contactDeletionWork, error) {
	work := contactDeletionWork{contactID: contact.ID}
	if err := tx.Model(&models.Note{}).Where("contact_id = ?", contact.ID).Pluck("id", &work.noteIDs).Error; err != nil {
		return work, err
	}
	var states []models.ContactSubscriptionState
	if err := tx.Where("contact_id = ?", contact.ID).Find(&states).Error; err != nil {
		return work, err
	}
	work.remoteDeleteTargets = make([]contactRemoteDeletionTarget, len(states))
	for index := range states {
		work.remoteDeleteTargets[index] = newContactRemoteDeletionTarget(states[index])
	}
	if err := tx.Where("contact_id = ?", contact.ID).Delete(&models.ContactVaultUser{}).Error; err != nil {
		return work, err
	}
	reminderIDs := tx.Model(&models.ContactReminder{}).Select("id").Where("contact_id = ?", contact.ID)
	if err := tx.Where("contact_reminder_id IN (?) AND triggered_at IS NULL", reminderIDs).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
		return work, err
	}
	if err := tx.Where("contact_id = ?", contact.ID).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
		return work, err
	}
	deleteResult := tx.Delete(contact)
	if deleteResult.Error != nil {
		return work, deleteResult.Error
	}
	if deleteResult.RowsAffected != 1 {
		return work, ErrContactNotFound
	}
	if s.feedRecorder != nil {
		if err := NewFeedRecorder(tx).Record(contact.ID, "", ActionContactDeleted, "Deleted contact", nil, nil); err != nil {
			return work, err
		}
	}
	return work, nil
}
