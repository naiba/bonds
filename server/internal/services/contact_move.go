package services

import (
	"errors"
	"fmt"
	"log"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrTargetVaultNotFound = errors.New("target vault not found")
	ErrContactMoveEmpty    = errors.New("contact move list is empty")
)

type ContactMoveService struct {
	db             *gorm.DB
	searchService  *SearchService
	davPushService *DavPushService
	fileService    *VaultFileService
}

func NewContactMoveService(db *gorm.DB) *ContactMoveService {
	return &ContactMoveService{db: db}
}

func (s *ContactMoveService) SetSearchService(ss *SearchService) {
	s.searchService = ss
}

func (s *ContactMoveService) SetDavPushService(ps *DavPushService) {
	s.davPushService = ps
}

func (s *ContactMoveService) SetFileService(fs *VaultFileService) {
	s.fileService = fs
}

func (s *ContactMoveService) Move(contactID, currentVaultID, targetVaultID, userID string) (*dto.ContactResponse, error) {
	resp, err := s.MoveMany([]string{contactID}, currentVaultID, targetVaultID, userID)
	if err != nil {
		return nil, err
	}
	if len(resp.Contacts) == 0 {
		return nil, ErrContactNotFound
	}
	return &resp.Contacts[0], nil
}

func (s *ContactMoveService) MoveMany(contactIDs []string, currentVaultID, targetVaultID, userID string) (*dto.BulkMoveContactsResponse, error) {
	uniqueContactIDs := dedupeContactIDs(contactIDs)
	if len(uniqueContactIDs) == 0 {
		return nil, ErrContactMoveEmpty
	}
	if s.davPushService != nil && currentVaultID != targetVaultID {
		releaseDAVOperations := lockMovedContactDAVOperations(&s.davPushService.operationLocks, uniqueContactIDs)
		defer releaseDAVOperations()
	}

	var movedContacts []models.Contact
	var quickFactFilesToDelete []models.File
	var sourceDAVDeleteTargets []contactRemoteDeletionTarget
	var responses []dto.ContactResponse
	var reindexedContacts []models.Contact
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := validateMoveVaultsAndTargetAccess(tx, currentVaultID, targetVaultID, userID); err != nil {
			return err
		}
		contacts, err := loadMovableContacts(tx, uniqueContactIDs, currentVaultID)
		if err != nil {
			return err
		}
		if currentVaultID != targetVaultID {
			sourceTargets, err := captureMovedContactSourceDAVDeleteTargets(tx, uniqueContactIDs, currentVaultID)
			if err != nil {
				return err
			}
			sourceDAVDeleteTargets = sourceTargets
			if err := updateBatchFirstMetThrough(tx, uniqueContactIDs, currentVaultID); err != nil {
				return err
			}
			if err := moveAllContactRows(tx, uniqueContactIDs, currentVaultID, targetVaultID); err != nil {
				return err
			}
			if err := moveContactAddresses(tx, uniqueContactIDs, currentVaultID, targetVaultID); err != nil {
				return err
			}
			if err := remapMovedImportantDateTypes(tx, uniqueContactIDs, currentVaultID, targetVaultID); err != nil {
				return err
			}
			filesToDelete, err := remapMovedQuickFacts(tx, uniqueContactIDs, currentVaultID, targetVaultID, s.fileService != nil)
			if err != nil {
				return err
			}
			quickFactFilesToDelete = filesToDelete
			if err := rescheduleMovedContactReminders(tx, uniqueContactIDs); err != nil {
				return err
			}
			if err := moveFullyOwnedTasks(tx, uniqueContactIDs, targetVaultID); err != nil {
				return err
			}
			if err := moveFullyOwnedLoans(tx, uniqueContactIDs, targetVaultID); err != nil {
				return err
			}
			if err := cleanSourceScopedMovePivots(tx, uniqueContactIDs, currentVaultID); err != nil {
				return err
			}
			if err := cleanMovedContactsFromActivities(tx, uniqueContactIDs, currentVaultID); err != nil {
				return err
			}
		}
		movedContacts = contacts

		// Response hydration is transactional so a hydration failure cannot commit a partial move.
		formatter, err := newContactNameFormatter(tx, userID)
		if err != nil {
			return err
		}
		responses = make([]dto.ContactResponse, 0, len(movedContacts))
		reindexedContacts = make([]models.Contact, 0, len(movedContacts))
		for _, movedContact := range movedContacts {
			var contact models.Contact
			if err := tx.Preload("FirstMetThrough", "vault_id = ?", targetVaultID).
				First(&contact, "id = ? AND vault_id = ?", movedContact.ID, targetVaultID).Error; err != nil {
				return err
			}
			resp, err := toContactResponse(&contact, false, formatter)
			if err != nil {
				return err
			}
			responses = append(responses, resp)
			reindexedContacts = append(reindexedContacts, contact)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	for i := range quickFactFilesToDelete {
		if err := s.fileService.deleteFileRecord(&quickFactFilesToDelete[i]); err != nil {
			// Post-commit maintenance cannot change the result of an already committed move.
			log.Printf("[contact-move] failed to delete moved quick fact file %d: %v", quickFactFilesToDelete[i].ID, err)
		}
	}

	if currentVaultID != targetVaultID {
		s.runMovedContactDAVLifecycle(uniqueContactIDs, sourceDAVDeleteTargets, targetVaultID)
		if err := s.reindexMovedSearchDocuments(uniqueContactIDs, reindexedContacts, targetVaultID); err != nil {
			log.Printf("[contact-move] failed to reindex moved search documents for target vault %s contacts %v: %v", targetVaultID, uniqueContactIDs, err)
		}
	}
	return &dto.BulkMoveContactsResponse{MovedCount: len(responses), Contacts: responses}, nil
}

func (s *ContactMoveService) reindexMovedSearchDocuments(contactIDs []string, contacts []models.Contact, targetVaultID string) error {
	if s.searchService == nil {
		return nil
	}
	errs := make([]error, 0)
	for i := range contacts {
		contacts[i].VaultID = targetVaultID
		if err := s.searchService.IndexContact(&contacts[i]); err != nil {
			deleteErr := s.searchService.DeleteContact(contacts[i].ID)
			errs = append(errs, fmt.Errorf("failed to reindex moved contact %s: %w", contacts[i].ID, errors.Join(err, deleteErr)))
		}
	}
	var notes []models.Note
	if err := s.db.Where("contact_id IN ? AND vault_id = ?", contactIDs, targetVaultID).Find(&notes).Error; err != nil {
		errs = append(errs, fmt.Errorf("failed to load moved notes for search reindex: %w", err))
		return errors.Join(errs...)
	}
	for i := range notes {
		if err := s.searchService.IndexNote(&notes[i]); err != nil {
			deleteErr := s.searchService.DeleteNote(notes[i].ID)
			errs = append(errs, fmt.Errorf("failed to reindex moved note %d: %w", notes[i].ID, errors.Join(err, deleteErr)))
		}
	}
	return errors.Join(errs...)
}

func validateMoveVaultsAndTargetAccess(tx *gorm.DB, currentVaultID, targetVaultID, userID string) error {
	var sourceVault models.Vault
	if err := tx.Where("id = ?", currentVaultID).First(&sourceVault).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrContactNotFound
		}
		return err
	}
	var targetVault models.Vault
	if err := tx.Where("id = ?", targetVaultID).First(&targetVault).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTargetVaultNotFound
		}
		return err
	}
	if targetVault.AccountID != sourceVault.AccountID {
		return ErrVaultForbidden
	}
	return NewVaultService(tx).CheckUserVaultAccess(userID, targetVaultID, models.PermissionEditor)
}

func updateBatchFirstMetThrough(tx *gorm.DB, contactIDs []string, currentVaultID string) error {
	return tx.Model(&models.Contact{}).
		Where("vault_id = ? AND id IN ? AND first_met_through_contact_id IS NOT NULL AND first_met_through_contact_id NOT IN ?", currentVaultID, contactIDs, contactIDs).
		Update("first_met_through_contact_id", nil).Error
}

func moveAllContactRows(tx *gorm.DB, contactIDs []string, currentVaultID, targetVaultID string) error {
	if err := tx.Model(&models.Contact{}).Where("id IN ?", contactIDs).Updates(map[string]interface{}{
		"vault_id":     targetVaultID,
		"distant_uuid": nil,
		"distant_etag": nil,
		"distant_uri":  nil,
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Note{}).Where("contact_id IN ? AND vault_id = ?", contactIDs, currentVaultID).Update("vault_id", targetVaultID).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.ContactVaultUser{}).Where("contact_id IN ? AND vault_id = ?", contactIDs, currentVaultID).Update("vault_id", targetVaultID).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.File{}).Where("ufileable_id IN ? AND vault_id = ?", contactIDs, currentVaultID).Update("vault_id", targetVaultID).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Contact{}).
		Where("id IN ? AND company_id IN (?)", contactIDs, tx.Model(&models.Company{}).Select("id").Where("vault_id = ?", currentVaultID)).
		Update("company_id", nil).Error; err != nil {
		return err
	}
	return nil
}

func moveContactAddresses(tx *gorm.DB, contactIDs []string, currentVaultID, targetVaultID string) error {
	var addressIDs []uint
	if err := tx.Model(&models.ContactAddress{}).
		Joins("JOIN addresses ON addresses.id = contact_address.address_id").
		Where("contact_address.contact_id IN ? AND addresses.vault_id = ?", contactIDs, currentVaultID).
		Pluck("DISTINCT contact_address.address_id", &addressIDs).Error; err != nil {
		return err
	}
	for _, addressID := range addressIDs {
		var outsideCount int64
		if err := tx.Model(&models.ContactAddress{}).
			Where("address_id = ? AND contact_id NOT IN ?", addressID, contactIDs).
			Count(&outsideCount).Error; err != nil {
			return err
		}
		if outsideCount == 0 {
			if err := tx.Model(&models.Address{}).Where("id = ?", addressID).Update("vault_id", targetVaultID).Error; err != nil {
				return err
			}
			continue
		}

		var sourceAddress models.Address
		if err := tx.First(&sourceAddress, addressID).Error; err != nil {
			return err
		}
		copiedAddress := models.Address{
			VaultID:       targetVaultID,
			AddressTypeID: sourceAddress.AddressTypeID,
			Line1:         cloneStringPtr(sourceAddress.Line1),
			Line2:         cloneStringPtr(sourceAddress.Line2),
			City:          cloneStringPtr(sourceAddress.City),
			Province:      cloneStringPtr(sourceAddress.Province),
			PostalCode:    cloneStringPtr(sourceAddress.PostalCode),
			Country:       cloneStringPtr(sourceAddress.Country),
			Latitude:      cloneFloat64Ptr(sourceAddress.Latitude),
			Longitude:     cloneFloat64Ptr(sourceAddress.Longitude),
		}
		if err := tx.Create(&copiedAddress).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.ContactAddress{}).
			Where("address_id = ? AND contact_id IN ?", addressID, contactIDs).
			Update("address_id", copiedAddress.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func remapMovedImportantDateTypes(tx *gorm.DB, contactIDs []string, currentVaultID, targetVaultID string) error {
	type dateTypeRow struct {
		DateID       uint
		InternalType *string
		Label        string
	}
	var rows []dateTypeRow
	if err := tx.Table("contact_important_dates").
		Select("contact_important_dates.id AS date_id, contact_important_date_types.internal_type, contact_important_date_types.label").
		Joins("JOIN contact_important_date_types ON contact_important_date_types.id = contact_important_dates.contact_important_date_type_id").
		Where("contact_important_dates.contact_id IN ? AND contact_important_date_types.vault_id = ?", contactIDs, currentVaultID).
		Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		targetTypeID, err := findMatchingImportantDateType(tx, targetVaultID, row.InternalType, row.Label)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"contact_important_date_type_id": nil,
			"distant_uuid":                   nil,
			"distant_etag":                   nil,
			"distant_uri":                    nil,
		}
		if targetTypeID != nil {
			updates["contact_important_date_type_id"] = *targetTypeID
		}
		if err := tx.Model(&models.ContactImportantDate{}).Where("id = ?", row.DateID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return tx.Model(&models.ContactImportantDate{}).Where("contact_id IN ?", contactIDs).Updates(map[string]interface{}{
		"distant_uuid": nil,
		"distant_etag": nil,
		"distant_uri":  nil,
	}).Error
}

func findMatchingImportantDateType(tx *gorm.DB, targetVaultID string, internalType *string, label string) (*uint, error) {
	query := tx.Model(&models.ContactImportantDateType{}).Where("vault_id = ?", targetVaultID)
	if internalType != nil && *internalType != "" {
		query = query.Where("internal_type = ?", *internalType)
	} else {
		query = query.Where("label = ?", label)
	}
	var id uint
	if err := query.Select("id").Limit(1).Scan(&id).Error; err != nil {
		return nil, err
	}
	if id == 0 {
		return nil, nil
	}
	return &id, nil
}

func remapMovedQuickFacts(tx *gorm.DB, contactIDs []string, currentVaultID, targetVaultID string, canDeleteFiles bool) ([]models.File, error) {
	type quickFactRow struct {
		FactID              uint
		FileID              *uint
		Label               *string
		LabelTranslationKey *string
		FieldType           string
	}
	var rows []quickFactRow
	if err := tx.Table("quick_facts").
		Select("quick_facts.id AS fact_id, quick_facts.file_id, vault_quick_facts_templates.label, vault_quick_facts_templates.label_translation_key, vault_quick_facts_templates.field_type").
		Joins("JOIN vault_quick_facts_templates ON vault_quick_facts_templates.id = quick_facts.vault_quick_facts_template_id").
		Where("quick_facts.contact_id IN ? AND vault_quick_facts_templates.vault_id = ?", contactIDs, currentVaultID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	deleteFactIDs := make([]uint, 0)
	deleteFileIDs := make(map[uint]struct{})
	for _, row := range rows {
		targetTemplateID, err := findMatchingQuickFactTemplate(tx, targetVaultID, row.LabelTranslationKey, row.Label, row.FieldType)
		if err != nil {
			return nil, err
		}
		if targetTemplateID == nil {
			deleteFactIDs = append(deleteFactIDs, row.FactID)
			if row.FileID != nil {
				deleteFileIDs[*row.FileID] = struct{}{}
			}
			continue
		}
		if err := tx.Model(&models.QuickFact{}).Where("id = ?", row.FactID).Update("vault_quick_facts_template_id", *targetTemplateID).Error; err != nil {
			return nil, err
		}
	}
	filesToDelete := make([]models.File, 0, len(deleteFileIDs))
	if len(deleteFileIDs) > 0 {
		if !canDeleteFiles {
			return nil, errors.New("contact move requires file service to delete missing quick fact files")
		}
		fileIDs := make([]uint, 0, len(deleteFileIDs))
		for id := range deleteFileIDs {
			fileIDs = append(fileIDs, id)
		}
		if err := tx.Where("id IN ?", fileIDs).Find(&filesToDelete).Error; err != nil {
			return nil, err
		}
	}
	if len(deleteFactIDs) > 0 {
		if err := tx.Where("id IN ?", deleteFactIDs).Delete(&models.QuickFact{}).Error; err != nil {
			return nil, err
		}
	}
	return filesToDelete, nil
}

func findMatchingQuickFactTemplate(tx *gorm.DB, targetVaultID string, translationKey *string, label *string, fieldType string) (*uint, error) {
	query := tx.Model(&models.VaultQuickFactsTemplate{}).Where("vault_id = ? AND field_type = ?", targetVaultID, fieldType)
	if translationKey != nil && *translationKey != "" {
		query = query.Where("label_translation_key = ?", *translationKey)
	} else if label != nil && *label != "" {
		query = query.Where("label = ?", *label)
	} else {
		return nil, nil
	}
	var id uint
	if err := query.Select("id").Limit(1).Scan(&id).Error; err != nil {
		return nil, err
	}
	if id == 0 {
		return nil, nil
	}
	return &id, nil
}

func rescheduleMovedContactReminders(tx *gorm.DB, contactIDs []string) error {
	var reminders []models.ContactReminder
	if err := tx.Where("contact_id IN ?", contactIDs).Find(&reminders).Error; err != nil {
		return err
	}
	if len(reminders) == 0 {
		return nil
	}
	reminderIDs := make([]uint, len(reminders))
	for i := range reminders {
		reminderIDs[i] = reminders[i].ID
	}
	if err := tx.Where("contact_reminder_id IN ? AND triggered_at IS NULL", reminderIDs).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
		return err
	}
	for i := range reminders {
		if err := scheduleReminderForVaultUsers(tx, &reminders[i]); err != nil {
			return err
		}
	}
	return nil
}

func moveFullyOwnedTasks(tx *gorm.DB, contactIDs []string, targetVaultID string) error {
	var taskIDs []uint
	if err := tx.Model(&models.TaskContact{}).Where("contact_id IN ?", contactIDs).Pluck("DISTINCT contact_task_id", &taskIDs).Error; err != nil {
		return err
	}
	for _, taskID := range taskIDs {
		var outsideCount int64
		if err := tx.Model(&models.TaskContact{}).Where("contact_task_id = ? AND contact_id NOT IN ?", taskID, contactIDs).Count(&outsideCount).Error; err != nil {
			return err
		}
		if outsideCount == 0 {
			if err := tx.Model(&models.ContactTask{}).Where("id = ?", taskID).Update("vault_id", targetVaultID).Error; err != nil {
				return err
			}
		} else if err := tx.Where("contact_task_id = ? AND contact_id IN ?", taskID, contactIDs).Delete(&models.TaskContact{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func moveFullyOwnedLoans(tx *gorm.DB, contactIDs []string, targetVaultID string) error {
	var loanIDs []uint
	if err := tx.Model(&models.ContactLoan{}).Where("loaner_id IN ? OR loanee_id IN ?", contactIDs, contactIDs).Pluck("DISTINCT loan_id", &loanIDs).Error; err != nil {
		return err
	}
	for _, loanID := range loanIDs {
		var outsideCount int64
		if err := tx.Model(&models.ContactLoan{}).Where("loan_id = ? AND (loaner_id NOT IN ? OR loanee_id NOT IN ?)", loanID, contactIDs, contactIDs).Count(&outsideCount).Error; err != nil {
			return err
		}
		if outsideCount == 0 {
			if err := tx.Model(&models.Loan{}).Where("id = ?", loanID).Update("vault_id", targetVaultID).Error; err != nil {
				return err
			}
		} else if err := tx.Where("loan_id = ? AND (loaner_id IN ? OR loanee_id IN ?)", loanID, contactIDs, contactIDs).Delete(&models.ContactLoan{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func cleanSourceScopedMovePivots(tx *gorm.DB, contactIDs []string, currentVaultID string) error {
	if err := tx.Where("contact_id IN ? AND label_id IN (?)", contactIDs, tx.Model(&models.Label{}).Select("id").Where("vault_id = ?", currentVaultID)).Delete(&models.ContactLabel{}).Error; err != nil {
		return err
	}
	if err := tx.Where("contact_id IN ? AND group_id IN (?)", contactIDs, tx.Model(&models.Group{}).Select("id").Where("vault_id = ?", currentVaultID)).Delete(&models.ContactGroup{}).Error; err != nil {
		return err
	}
	if err := tx.Where("contact_id IN ? AND company_id IN (?)", contactIDs, tx.Model(&models.Company{}).Select("id").Where("vault_id = ?", currentVaultID)).Delete(&models.ContactCompany{}).Error; err != nil {
		return err
	}
	if err := tx.Where("contact_id IN ? AND post_id IN (?)", contactIDs, tx.Model(&models.Post{}).Select("posts.id").Joins("JOIN journals ON journals.id = posts.journal_id").Where("journals.vault_id = ?", currentVaultID)).Delete(&models.ContactPost{}).Error; err != nil {
		return err
	}
	if err := tx.Where("contact_id IN ? AND life_metric_id IN (?)", contactIDs, tx.Model(&models.LifeMetric{}).Select("id").Where("vault_id = ?", currentVaultID)).Delete(&models.ContactLifeMetric{}).Error; err != nil {
		return err
	}
	if err := cleanMovedContactDavState(tx, contactIDs, currentVaultID); err != nil {
		return err
	}
	return nil
}

func cleanMovedContactDavState(tx *gorm.DB, contactIDs []string, currentVaultID string) error {
	sourceSubscriptionIDs := tx.Model(&models.AddressBookSubscription{}).Select("id").Where("vault_id = ?", currentVaultID)
	if err := tx.Where("contact_id IN ? AND address_book_subscription_id IN (?)", contactIDs, sourceSubscriptionIDs).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
		return err
	}
	if err := tx.Where("contact_id IN ? AND address_book_subscription_id IN (?)", contactIDs, sourceSubscriptionIDs).Delete(&models.DavSyncLog{}).Error; err != nil {
		return err
	}
	return nil
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cleanMovedContactsFromActivities(tx *gorm.DB, contactIDs []string, currentVaultID string) error {
	var affectedActivityIDs []uint
	if err := tx.Model(&models.ActivityParticipant{}).
		Where("contact_id IN ?", contactIDs).
		Distinct().
		Pluck("activity_id", &affectedActivityIDs).Error; err != nil {
		return err
	}
	if len(affectedActivityIDs) == 0 {
		return nil
	}
	if err := tx.Where("contact_id IN ?", contactIDs).Delete(&models.ActivityParticipant{}).Error; err != nil {
		return err
	}
	var orphanActivityIDs []uint
	if err := tx.Model(&models.Activity{}).
		Where("vault_id = ? AND id IN ? AND subject_user_id IS NULL", currentVaultID, affectedActivityIDs).
		Where("NOT EXISTS (?)", tx.Model(&models.ActivityParticipant{}).
			Select("1").
			Where("activity_participants.activity_id = activities.id")).
		Pluck("id", &orphanActivityIDs).Error; err != nil {
		return err
	}
	if len(orphanActivityIDs) == 0 {
		return nil
	}
	if err := tx.Model(&models.Activity{}).
		Where("parent_id IN ?", orphanActivityIDs).
		Update("parent_id", nil).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", orphanActivityIDs).Delete(&models.Activity{}).Error
}
