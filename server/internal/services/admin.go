package services

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	userTimezone "github.com/naiba/bonds/internal/timezone"
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
	db          *gorm.DB
	uploadDir   string
	credentials *CredentialActionService
}

func NewAdminService(db *gorm.DB, uploadDir string) *AdminService {
	return &AdminService{db: db, uploadDir: uploadDir}
}

func (s *AdminService) SetCredentialActions(service *CredentialActionService) {
	s.credentials = service
}

// CreateUser provisions a separate, private home account. The operator never
// receives a password or gains access to the user's vaults. The new user
// chooses their own password using a one-time email link.
func (s *AdminService) CreateUser(req dto.AdminCreateUserRequest) (*dto.AdminUserResponse, error) {
	if s.credentials == nil {
		return nil, ErrMailerNotConfigured
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	var existing int64
	if err := s.db.Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing != 0 {
		return nil, ErrEmailExists
	}
	var user models.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		account := models.Account{}
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		tz := userTimezone.Default
		user = models.User{AccountID: account.ID, Email: email, FirstName: strPtrOrNil(req.FirstName),
			LastName: strPtrOrNil(req.LastName), Timezone: &tz, Locale: "en", IsAccountAdministrator: true}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.AccountMembership{AccountID: account.ID, UserID: user.ID, IsAdmin: true}).Error; err != nil {
			return err
		}
		if err := models.SeedAccountDefaults(tx, account.ID, user.ID, email, "en"); err != nil {
			return err
		}
		return s.credentials.Send(tx, &user, "password_setup")
	})
	if err != nil {
		return nil, err
	}
	result := s.toAdminUserResponse(user)
	return &result, nil
}

func (s *AdminService) SendPasswordReset(targetID string) error {
	if s.credentials == nil {
		return ErrMailerNotConfigured
	}
	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.credentials.Send(tx, &user, "password_reset")
	})
}

func (s *AdminService) UpdateIdentity(targetID string, req dto.AdminUpdateIdentityRequest) (*dto.AdminUserResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminUserNotFound
		}
		return nil, err
	}
	if err := s.db.Model(&user).Updates(map[string]interface{}{"first_name": strings.TrimSpace(req.FirstName), "last_name": strings.TrimSpace(req.LastName)}).Error; err != nil {
		return nil, err
	}
	result := s.toAdminUserResponse(user)
	result.FirstName = strings.TrimSpace(req.FirstName)
	result.LastName = strings.TrimSpace(req.LastName)
	return &result, nil
}

func (s *AdminService) RequestEmailChange(targetID, email string) error {
	if s.credentials == nil {
		return ErrMailerNotConfigured
	}
	var user models.User
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.credentials.RequestEmailChange(tx, &user, email)
	})
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

func (s *AdminService) ListAudit(page, perPage int) ([]dto.AuditEventResponse, response.Meta, error) {
	return s.listAudit(nil, page, perPage)
}

func (s *AdminService) ListVaultAudit(vaultID string, page, perPage int) ([]dto.AuditEventResponse, response.Meta, error) {
	return s.listAudit(&vaultID, page, perPage)
}

func (s *AdminService) listAudit(vaultID *string, page, perPage int) ([]dto.AuditEventResponse, response.Meta, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	var total int64
	query := s.db.Model(&models.AuditEvent{})
	if vaultID == nil {
		query = query.Where("vault_id IS NULL")
	} else {
		query = query.Where("vault_id = ?", *vaultID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}
	var rows []models.AuditEvent
	if err := query.Order("id DESC").Limit(perPage).Offset((page - 1) * perPage).Find(&rows).Error; err != nil {
		return nil, response.Meta{}, err
	}
	result := make([]dto.AuditEventResponse, len(rows))
	for i, row := range rows {
		result[i] = dto.AuditEventResponse{ID: row.ID, ActorUserID: row.ActorUserID,
			AccountID: row.AccountID, VaultID: row.VaultID, Method: row.Method, Route: row.Route,
			Status: row.Status, RequestID: row.RequestID, CreatedAt: row.CreatedAt}
	}
	return result, response.Meta{Page: page, PerPage: perPage, Total: total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage)))}, nil
}

func (s *AdminService) toAdminUserResponse(u models.User) dto.AdminUserResponse {
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
		StorageLimitInMB:        account.StorageLimitInMB,
		CreatedAt:               u.CreatedAt,
	}
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

	if !disabled {
		return s.db.Model(&user).Update("disabled", false).Error
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := ensureUserIsNotSoleVaultManager(tx, user.ID); err != nil {
			return err
		}
		return tx.Model(&user).Update("disabled", true).Error
	})
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

	// Memberships, not home-account IDs, are authoritative. Other users may
	// have joined this account without changing their original home account.
	var accountUserCount int64
	if err := s.db.Model(&models.AccountMembership{}).Where("account_id = ?", user.AccountID).Count(&accountUserCount).Error; err != nil {
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
	err := companyEmploymentTransaction(s.db, func(tx *gorm.DB) error {
		fileUUIDs = nil // A retry must only publish file removals from its committed attempt.
		// Collect physical file paths while the rows still exist; the files
		// themselves are removed only after the transaction commits.
		if err := tx.Model(&models.File{}).
			Joins("INNER JOIN vaults ON files.vault_id = vaults.id").
			Where("vaults.account_id = ?", user.AccountID).
			Pluck("uuid", &fileUUIDs).Error; err != nil {
			return fmt.Errorf("collect user files: %w", err)
		}

		var vaults []models.Vault
		if err := tx.Where("account_id = ?", user.AccountID).Order("id ASC").Find(&vaults).Error; err != nil {
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
			&models.AccountMembership{},
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
		if err := ensureUserIsNotSoleVaultManager(tx, user.ID); err != nil {
			return err
		}
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
		&models.AccountMembership{},
		&models.AuthActionToken{},
		&models.PersonalAccessToken{},
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

// Use the same company/contact lock order and complete cascade as vault deletion
// and account cancellation; a separate child list previously missed jobs.
func (s *AdminService) deleteVaultData(tx *gorm.DB, vaultID string) error {
	return deleteVaultCascade(tx, vaultID)
}

func (s *AdminService) deleteContactData(tx *gorm.DB, contactID string) error {
	var noteIDs []uint
	if err := tx.Model(&models.Note{}).Where("contact_id = ?", contactID).Pluck("id", &noteIDs).Error; err != nil {
		return fmt.Errorf("collect note IDs: %w", err)
	}
	if len(noteIDs) > 0 {
		if err := tx.Where("owner_type = ? AND owner_id IN ?", models.ContentOwnerNote, noteIDs).
			Delete(&models.ContentFileReference{}).Error; err != nil {
			return fmt.Errorf("delete note file references: %w", err)
		}
	}
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
