package services

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

var (
	ErrCannotDisableSelf = errors.New("cannot disable yourself")
	ErrCannotDemoteSelf  = errors.New("cannot demote yourself")
	ErrUserDisabled      = errors.New("user account is disabled")
	ErrAdminUserNotFound = errors.New("user not found")
)

type AdminService struct {
	db        *gorm.DB
	uploadDir string
}

func NewAdminService(db *gorm.DB, uploadDir string) *AdminService {
	return &AdminService{db: db, uploadDir: uploadDir}
}

func (s *AdminService) ListUsers(page, perPage int) ([]dto.AdminUserResponse, response.Meta, error) {
	var total int64
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var users []models.User
	if err := s.db.Order("created_at ASC").Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return nil, response.Meta{}, err
	}

	result := make([]dto.AdminUserResponse, len(users))
	for i, u := range users {
		result[i] = s.toAdminUserResponse(u)
	}

	meta := response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}
	return result, meta, nil
}

func adminContactCountSQL() string {
	return `
		SELECT COUNT(DISTINCT c.id)
		FROM contacts c
		INNER JOIN vaults v ON c.vault_id = v.id
		WHERE v.account_id = ?`
}

func (s *AdminService) toAdminUserResponse(u models.User) dto.AdminUserResponse {
	var contactCount int64
	s.db.Raw(adminContactCountSQL(), u.AccountID).Scan(&contactCount)

	var vaultCount int64
	s.db.Model(&models.Vault{}).Where("account_id = ?", u.AccountID).Count(&vaultCount)

	storageUsed := s.calculateStorageUsed(u.AccountID)

	var account models.Account
	s.db.First(&account, "id = ?", u.AccountID)

	return dto.AdminUserResponse{
		ID:                      u.ID,
		AccountID:               u.AccountID,
		FirstName:               ptrToStr(u.FirstName),
		LastName:                ptrToStr(u.LastName),
		Email:                   u.Email,
		IsAccountAdministrator:  u.IsAccountAdministrator,
		IsInstanceAdministrator: u.IsInstanceAdministrator,
		Disabled:                u.Disabled,
		ContactCount:            contactCount,
		StorageUsed:             storageUsed,
		StorageLimitInMB:        account.StorageLimitInMB,
		VaultCount:              vaultCount,
		CreatedAt:               u.CreatedAt,
	}
}

func (s *AdminService) calculateStorageUsed(accountID string) int64 {
	var files []models.File
	s.db.Joins("INNER JOIN vaults ON files.vault_id = vaults.id").
		Where("vaults.account_id = ?", accountID).
		Select("files.uuid, files.size").
		Find(&files)

	var totalSize int64
	for _, f := range files {
		totalSize += int64(f.Size)
	}
	return totalSize
}

func (s *AdminService) ToggleUser(actorID, targetID string, disabled bool) error {
	if actorID == targetID {
		return ErrCannotDisableSelf
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}

	return s.db.Model(&user).Update("disabled", disabled).Error
}

func (s *AdminService) SetAdmin(actorID, targetID string, isAdmin bool) error {
	if actorID == targetID && !isAdmin {
		return ErrCannotDemoteSelf
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}

	return s.db.Model(&user).Update("is_instance_administrator", isAdmin).Error
}

func (s *AdminService) SetStorageLimit(targetID string, limitMB int) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	return s.db.Model(&models.Account{}).Where("id = ?", user.AccountID).Update("storage_limit_in_mb", limitMB).Error
}

func (s *AdminService) DeleteUser(actorID, targetID string) error {
	if actorID == targetID {
		return ErrCannotDeleteSelf
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}

	// Count how many users share this account (invitation system allows multiple users per account).
	var accountUserCount int64
	if err := s.db.Model(&models.User{}).Where("account_id = ?", user.AccountID).Count(&accountUserCount).Error; err != nil {
		return err
	}

	if accountUserCount <= 1 {
		// This is the only user on the account — delete the entire account and all its data.
		return s.deleteEntireAccount(user)
	}

	// Other users still share this account — only remove this user's personal data.
	return s.deleteUserOnly(user)
}

// deleteEntireAccount removes the user, the account, and all associated data (vaults, contacts, files, etc.).
func (s *AdminService) deleteEntireAccount(user models.User) error {
	var fileUUIDs []string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Collect physical file paths while the rows still exist; the files
		// themselves are removed only after the transaction commits.
		if err := tx.Model(&models.File{}).
			Joins("INNER JOIN vaults ON files.vault_id = vaults.id").
			Where("vaults.account_id = ?", user.AccountID).
			Pluck("uuid", &fileUUIDs).Error; err != nil {
			return fmt.Errorf("collect user files: %w", err)
		}

		var vaults []models.Vault
		if err := tx.Where("account_id = ?", user.AccountID).Find(&vaults).Error; err != nil {
			return err
		}

		for _, v := range vaults {
			if err := s.deleteVaultData(tx, v.ID); err != nil {
				return fmt.Errorf("delete vault %s data: %w", v.ID, err)
			}
		}

		groupTypeSubquery := tx.Model(&models.GroupType{}).Select("id").Where("account_id = ?", user.AccountID)
		if err := tx.Where("group_type_id IN (?)", groupTypeSubquery).Delete(&models.GroupTypeRole{}).Error; err != nil {
			return fmt.Errorf("delete group type roles: %w", err)
		}

		relGroupTypeSubquery := tx.Model(&models.RelationshipGroupType{}).Select("id").Where("account_id = ?", user.AccountID)
		if err := tx.Where("relationship_group_type_id IN (?)", relGroupTypeSubquery).Delete(&models.RelationshipType{}).Error; err != nil {
			return fmt.Errorf("delete relationship types: %w", err)
		}

		callReasonTypeSubquery := tx.Model(&models.CallReasonType{}).Select("id").Where("account_id = ?", user.AccountID)
		if err := tx.Where("call_reason_type_id IN (?)", callReasonTypeSubquery).Delete(&models.CallReason{}).Error; err != nil {
			return fmt.Errorf("delete call reasons: %w", err)
		}

		postTemplateSubquery := tx.Model(&models.PostTemplate{}).Select("id").Where("account_id = ?", user.AccountID)
		if err := tx.Where("post_template_id IN (?)", postTemplateSubquery).Delete(&models.PostTemplateSection{}).Error; err != nil {
			return fmt.Errorf("delete post template sections: %w", err)
		}

		accountTables := []interface{}{
			&models.Invitation{},
			&models.AccountCurrency{},
			&models.TaskStatus{},
			&models.Gender{},
			&models.Pronoun{},
			&models.AddressType{},
			&models.PetCategory{},
			&models.ContactInformationType{},
			&models.RelationshipGroupType{},
			&models.CallReasonType{},
			&models.Religion{},
			&models.Emotion{},
			&models.GiftOccasion{},
			&models.GiftState{},
			&models.PostTemplate{},
			&models.GroupType{},
			&models.SyncToken{},
		}

		for _, model := range accountTables {
			if err := tx.Where("account_id = ?", user.AccountID).Delete(model).Error; err != nil {
				return fmt.Errorf("delete account data for %T: %w", model, err)
			}
		}

		if err := tx.Where("account_id = ?", user.AccountID).Delete(&models.Vault{}).Error; err != nil {
			return err
		}

		if err := s.deleteUserDirectData(tx, user.ID); err != nil {
			return err
		}

		if err := tx.Delete(&user).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ?", user.AccountID).Delete(&models.Account{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Delete physical files only after the transaction commits: a rollback must
	// not leave the account without the files it still references.
	return s.removeFiles(fileUUIDs)
}

// deleteUserOnly removes only the user record and their personal data (notification channels,
// WebAuthn credentials, etc.) without touching the shared account, vaults, or contacts.
func (s *AdminService) deleteUserOnly(user models.User) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.deleteUserDirectData(tx, user.ID); err != nil {
			return err
		}
		return tx.Delete(&user).Error
	})
}

func (s *AdminService) deleteUserDirectData(tx *gorm.DB, userID string) error {
	// UserNotificationSent has no user_id; it links through UserNotificationChannel
	channelSubquery := tx.Model(&models.UserNotificationChannel{}).Select("id").Where("user_id = ?", userID)
	if err := tx.Where("user_notification_channel_id IN (?)", channelSubquery).Delete(&models.UserNotificationSent{}).Error; err != nil {
		return fmt.Errorf("delete user notification sent: %w", err)
	}

	userTables := []interface{}{
		&models.MoodTrackingEvent{},
		&models.UserNotificationChannel{},
		&models.UserToken{},
		&models.WebAuthnCredential{},
		&models.UserVault{},
	}
	for _, model := range userTables {
		if err := tx.Where("user_id = ?", userID).Delete(model).Error; err != nil {
			return fmt.Errorf("delete user data for %T: %w", model, err)
		}
	}
	return nil
}

func (s *AdminService) deleteVaultData(tx *gorm.DB, vaultID string) error {
	contactTemplateSubquery := tx.Model(&models.VaultContactTemplate{}).Select("id").Where("vault_id = ?", vaultID)
	contactPageSubquery := tx.Model(&models.VaultContactTemplatePage{}).Select("id").Where("template_id IN (?)", contactTemplateSubquery)
	if err := tx.Where("template_page_id IN (?)", contactPageSubquery).Delete(&models.VaultContactTemplateModule{}).Error; err != nil {
		return fmt.Errorf("delete vault contact template modules: %w", err)
	}
	if err := tx.Where("template_id IN (?)", contactTemplateSubquery).Delete(&models.VaultContactTemplatePage{}).Error; err != nil {
		return fmt.Errorf("delete vault contact template pages: %w", err)
	}
	if err := tx.Where("vault_id = ?", vaultID).Delete(&models.VaultContactTemplate{}).Error; err != nil {
		return fmt.Errorf("delete vault contact templates: %w", err)
	}
	var contacts []models.Contact
	if err := tx.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		return err
	}

	for _, c := range contacts {
		if err := s.deleteContactData(tx, c.ID); err != nil {
			return fmt.Errorf("delete contact %s data: %w", c.ID, err)
		}
	}

	if err := tx.Where("vault_id = ?", vaultID).Delete(&models.Contact{}).Error; err != nil {
		return err
	}

	journalSubquery := tx.Model(&models.Journal{}).Select("id").Where("vault_id = ?", vaultID)
	postSubquery := tx.Model(&models.Post{}).Select("id").Where("journal_id IN (?)", journalSubquery)

	for _, model := range []interface{}{&models.PostSection{}, &models.PostMetric{}, &models.PostTag{}} {
		if err := tx.Where("post_id IN (?)", postSubquery).Delete(model).Error; err != nil {
			return fmt.Errorf("delete post sub-data for %T: %w", model, err)
		}
	}
	if err := tx.Where("journal_id IN (?)", journalSubquery).Delete(&models.Post{}).Error; err != nil {
		return fmt.Errorf("delete posts: %w", err)
	}
	if err := tx.Where("journal_id IN (?)", journalSubquery).Delete(&models.SliceOfLife{}).Error; err != nil {
		return fmt.Errorf("delete slices of life: %w", err)
	}
	if err := tx.Where("journal_id IN (?)", journalSubquery).Delete(&models.JournalMetric{}).Error; err != nil {
		return fmt.Errorf("delete journal metrics: %w", err)
	}

	activitySubquery := tx.Model(&models.Activity{}).Select("id").Where("vault_id = ?", vaultID)
	if err := tx.Where("activity_id IN (?)", activitySubquery).Delete(&models.ActivityParticipant{}).Error; err != nil {
		return fmt.Errorf("delete activity participants: %w", err)
	}
	if err := tx.Where("vault_id = ?", vaultID).Delete(&models.Activity{}).Error; err != nil {
		return fmt.Errorf("delete activities: %w", err)
	}

	lifeCategorySubquery := tx.Model(&models.ActivityCategory{}).Select("id").Where("vault_id = ?", vaultID)
	if err := tx.Where("activity_category_id IN (?)", lifeCategorySubquery).Delete(&models.ActivityType{}).Error; err != nil {
		return fmt.Errorf("delete activity types: %w", err)
	}

	abSubquery := tx.Model(&models.AddressBookSubscription{}).Select("id").Where("vault_id = ?", vaultID)
	if err := tx.Where("address_book_subscription_id IN (?)", abSubquery).Delete(&models.DavSyncLog{}).Error; err != nil {
		return fmt.Errorf("delete dav sync logs: %w", err)
	}
	if err := tx.Where("address_book_subscription_id IN (?)", abSubquery).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
		return fmt.Errorf("delete contact subscription states: %w", err)
	}

	vaultTables := []interface{}{
		&models.MoodTrackingEvent{},
		&models.UserVault{},
		&models.ContactVaultUser{},
		&models.File{},
		&models.Label{},
		&models.Tag{},
		&models.ContactImportantDateType{},
		&models.MoodTrackingParameter{},
		&models.ActivityCategory{},
		&models.VaultQuickFactsTemplate{},
		&models.Group{},
		&models.Journal{},
		&models.Company{},
		&models.AddressBookSubscription{},
		&models.Address{},
		&models.Loan{},
		&models.ContactTask{},
		&models.LifeMetric{},
	}

	for _, model := range vaultTables {
		if err := tx.Where("vault_id = ?", vaultID).Delete(model).Error; err != nil {
			return fmt.Errorf("delete vault data for %T: %w", model, err)
		}
	}

	return nil
}

func (s *AdminService) deleteContactData(tx *gorm.DB, contactID string) error {
	// Reminder schedules and selected recipients depend on ContactReminder.
	if err := tx.Where("contact_reminder_id IN (?)",
		tx.Model(&models.ContactReminder{}).Select("id").Where("contact_id = ?", contactID),
	).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
		return fmt.Errorf("delete scheduled reminders: %w", err)
	}
	if err := tx.Where("contact_reminder_id IN (?)",
		tx.Model(&models.ContactReminder{}).Select("id").Where("contact_id = ?", contactID),
	).Delete(&models.ContactReminderSelectedUser{}).Error; err != nil {
		return fmt.Errorf("delete selected reminder recipients: %w", err)
	}
	goalSubquery := tx.Model(&models.Goal{}).Select("id").Where("contact_id = ?", contactID)
	if err := tx.Where("goal_id IN (?)", goalSubquery).Delete(&models.Streak{}).Error; err != nil {
		return fmt.Errorf("delete streaks: %w", err)
	}

	contactTables := []interface{}{
		&models.Note{},
		&models.ContactReminder{},
		&models.ContactFeedItem{},
		&models.ContactImportantDate{},
		&models.Call{},
		&models.ContactAddress{},
		&models.ContactInformation{},
		&models.Gift{},
		&models.Pet{},
		&models.Relationship{},
		&models.Goal{},
		&models.ContactGroup{},
		&models.ContactLabel{},
		&models.QuickFact{},
		&models.ContactPost{},
		&models.ContactLifeMetric{},
	}

	for _, model := range contactTables {
		if err := tx.Where("contact_id = ?", contactID).Delete(model).Error; err != nil {
			return fmt.Errorf("delete contact data for %T: %w", model, err)
		}
	}

	// Detach the contact from any tasks via the m2m pivot. The tasks
	// themselves are vault-scoped and are deleted in the vault cascade.
	if err := tx.Where("contact_id = ?", contactID).Delete(&models.TaskContact{}).Error; err != nil {
		return fmt.Errorf("delete contact data for *models.TaskContact: %w", err)
	}

	if err := tx.Where("loaner_id = ? OR loanee_id = ?", contactID, contactID).Delete(&models.ContactLoan{}).Error; err != nil {
		return fmt.Errorf("delete contact loans: %w", err)
	}

	if err := tx.Where("loaner_id = ? OR loanee_id = ?", contactID, contactID).Delete(&models.ContactGift{}).Error; err != nil {
		return fmt.Errorf("delete contact gifts: %w", err)
	}

	return nil
}

func (s *AdminService) removeFiles(fileUUIDs []string) error {
	for _, uuid := range fileUUIDs {
		filePath := filepath.Join(s.uploadDir, uuid)
		os.Remove(filePath)
	}
	return nil
}
