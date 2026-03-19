package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrVaultNotFound    = errors.New("vault not found")
	ErrVaultForbidden   = errors.New("vault access forbidden")
	ErrInsufficientPerm = errors.New("insufficient permissions")
)

type VaultService struct {
	db *gorm.DB
}

func NewVaultService(db *gorm.DB) *VaultService {
	return &VaultService{db: db}
}

func (s *VaultService) ListVaults(userID string) ([]dto.VaultResponse, error) {
	var userVaults []models.UserVault
	if err := s.db.Where("user_id = ?", userID).Find(&userVaults).Error; err != nil {
		return nil, err
	}

	vaultIDs := make([]string, len(userVaults))
	contactIDByVault := make(map[string]string, len(userVaults))
	for i, uv := range userVaults {
		vaultIDs[i] = uv.VaultID
		contactIDByVault[uv.VaultID] = uv.ContactID
	}

	if len(vaultIDs) == 0 {
		return []dto.VaultResponse{}, nil
	}

	var vaults []models.Vault
	if err := s.db.Where("id IN ?", vaultIDs).Find(&vaults).Error; err != nil {
		return nil, err
	}

	result := make([]dto.VaultResponse, len(vaults))
	for i, v := range vaults {
		result[i] = toVaultResponse(&v, contactIDByVault[v.ID])
	}
	return result, nil
}

func (s *VaultService) CreateVault(accountID, userID string, req dto.CreateVaultRequest, locale string) (*dto.VaultResponse, error) {
	desc := req.Description
	vault := models.Vault{
		AccountID:   accountID,
		Name:        req.Name,
		Description: &desc,
		Type:        "personal",
	}

	var userContactID string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&vault).Error; err != nil {
			return err
		}
		userVault := models.UserVault{
			UserID:     userID,
			VaultID:    vault.ID,
			Permission: models.PermissionManager,
		}
		if err := tx.Create(&userVault).Error; err != nil {
			return err
		}

		// Monica v5 pattern: auto-create a "self" Contact for the vault creator,
		// linked via UserVault.ContactID. This shadow contact (Listed=false, CanBeDeleted=false)
		// is used for mood tracking, life events, etc.
		contactID, err := createUserSelfContact(tx, userID, vault.ID)
		if err != nil {
			return err
		}
		userContactID = contactID
		if err := tx.Model(&userVault).Update("contact_id", contactID).Error; err != nil {
			return err
		}

		return models.SeedVaultDefaults(tx, vault.ID, locale)
	})
	if err != nil {
		return nil, err
	}

	resp := toVaultResponse(&vault, userContactID)
	return &resp, nil
}

func (s *VaultService) GetVault(vaultID, userID string) (*dto.VaultResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}
	userContactID := s.getUserContactID(userID, vaultID)
	resp := toVaultResponse(&vault, userContactID)
	return &resp, nil
}

func (s *VaultService) UpdateVault(vaultID, userID string, req dto.UpdateVaultRequest) (*dto.VaultResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}

	vault.Name = req.Name
	desc := req.Description
	vault.Description = &desc

	if err := s.db.Save(&vault).Error; err != nil {
		return nil, err
	}

	userContactID := s.getUserContactID(userID, vaultID)
	resp := toVaultResponse(&vault, userContactID)
	return &resp, nil
}

func (s *VaultService) DeleteVault(vaultID string) error {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVaultNotFound
		}
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		return deleteVaultCascade(tx, vaultID)
	})
}

// deleteVaultCascade deletes ALL data associated with a vault in the correct order
// to respect foreign key constraints. Discussion #68.
//
// Deletion order rationale:
//  1. Collect all contact IDs in the vault (including soft-deleted contacts)
//  2. Delete deepest children first: ContactReminderScheduled, Streaks, LifeEvents,
//     PostMetrics, PostSections, PostTags, etc.
//  3. Delete contact-level children: reminders, goals, notes, tasks, pivots, etc.
//  4. Delete vault-level children: journals, timeline events, labels, etc.
//  5. Delete contacts themselves
//  6. Delete the vault
func deleteVaultCascade(tx *gorm.DB, vaultID string) error {
	// Step 1: Collect ALL contact IDs in this vault (including soft-deleted ones,
	// since child tables still reference them).
	var contactIDs []string
	if err := tx.Model(&models.Contact{}).Unscoped().
		Where("vault_id = ?", vaultID).
		Pluck("id", &contactIDs).Error; err != nil {
		return err
	}

	// Step 2: Delete contact-level children (deepest grandchildren first).
	if len(contactIDs) > 0 {
		// --- Grandchildren (depend on contact children) ---

		// ContactReminderScheduled → depends on ContactReminder
		if err := tx.Where("contact_reminder_id IN (?)",
			tx.Model(&models.ContactReminder{}).Select("id").Where("contact_id IN ?", contactIDs),
		).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
			return err
		}

		// Streak → depends on Goal
		if err := tx.Where("goal_id IN (?)",
			tx.Model(&models.Goal{}).Select("id").Where("contact_id IN ?", contactIDs),
		).Delete(&models.Streak{}).Error; err != nil {
			return err
		}

		// --- Contact-level children (direct FK to contact_id) ---

		contactChildModels := []interface{}{
			&models.ContactInformation{},
			&models.ContactImportantDate{},
			&models.ContactReminder{},
			&models.ContactTask{},
			&models.ContactFeedItem{},
			&models.Call{},
			&models.Pet{},
			&models.Goal{},
			&models.Gift{},
			&models.Relationship{},
			&models.MoodTrackingEvent{},
			&models.QuickFact{},
			&models.Note{}, // Note has both contact_id and vault_id; delete by contact_id here
		}
		for _, m := range contactChildModels {
			if err := tx.Where("contact_id IN ?", contactIDs).Delete(m).Error; err != nil {
				return err
			}
		}

		// Also delete Relationships where this vault's contacts are the "related" side
		if err := tx.Where("related_contact_id IN ?", contactIDs).Delete(&models.Relationship{}).Error; err != nil {
			return err
		}

		// --- Pivot tables (contact_id FK) ---

		contactPivotModels := []interface{}{
			&models.ContactLabel{},
			&models.ContactGroup{},
			&models.ContactAddress{},
			&models.ContactPost{},
			&models.ContactCompany{},
			&models.ContactLifeMetric{},
			&models.LifeEventParticipant{},
			&models.TimelineEventParticipant{},
			&models.ContactSubscriptionState{},
		}
		for _, m := range contactPivotModels {
			if err := tx.Where("contact_id IN ?", contactIDs).Delete(m).Error; err != nil {
				return err
			}
		}

		// ContactLoan pivot uses loaner_id / loanee_id instead of contact_id
		if err := tx.Where("loaner_id IN ? OR loanee_id IN ?", contactIDs, contactIDs).
			Delete(&models.ContactLoan{}).Error; err != nil {
			return err
		}

		// ContactGift pivot uses loaner_id / loanee_id (same pattern as ContactLoan)
		if err := tx.Where("loaner_id IN ? OR loanee_id IN ?", contactIDs, contactIDs).
			Delete(&models.ContactGift{}).Error; err != nil {
			return err
		}

		// DavSyncLog has nullable contact_id
		if err := tx.Where("contact_id IN ?", contactIDs).Delete(&models.DavSyncLog{}).Error; err != nil {
			return err
		}
	}

	// Step 3: Delete vault-level children (grandchildren of vault-scoped tables first).

	// Journal cascade: PostMetric → PostSection → PostTag → ContactPost → Post → SliceOfLife → JournalMetric → Journal
	var journalIDs []uint
	tx.Model(&models.Journal{}).Where("vault_id = ?", vaultID).Pluck("id", &journalIDs)
	if len(journalIDs) > 0 {
		var postIDs []uint
		tx.Model(&models.Post{}).Where("journal_id IN ?", journalIDs).Pluck("id", &postIDs)
		if len(postIDs) > 0 {
			tx.Where("post_id IN ?", postIDs).Delete(&models.PostMetric{})
			tx.Where("post_id IN ?", postIDs).Delete(&models.PostSection{})
			tx.Where("post_id IN ?", postIDs).Delete(&models.PostTag{})
			tx.Where("post_id IN ?", postIDs).Delete(&models.ContactPost{})
			tx.Where("id IN ?", postIDs).Delete(&models.Post{})
		}

		var journalMetricIDs []uint
		tx.Model(&models.JournalMetric{}).Where("journal_id IN ?", journalIDs).Pluck("id", &journalMetricIDs)
		if len(journalMetricIDs) > 0 {
			// PostMetric references JournalMetricID — already deleted above via postIDs
			tx.Where("id IN ?", journalMetricIDs).Delete(&models.JournalMetric{})
		}

		tx.Where("journal_id IN ?", journalIDs).Delete(&models.SliceOfLife{})
		tx.Where("id IN ?", journalIDs).Delete(&models.Journal{})
	}

	// LifeEventCategory cascade: LifeEvent → LifeEventType → LifeEventCategory
	var categoryIDs []uint
	tx.Model(&models.LifeEventCategory{}).Where("vault_id = ?", vaultID).Pluck("id", &categoryIDs)
	if len(categoryIDs) > 0 {
		var typeIDs []uint
		tx.Model(&models.LifeEventType{}).Where("life_event_category_id IN ?", categoryIDs).Pluck("id", &typeIDs)
		if len(typeIDs) > 0 {
			tx.Where("life_event_type_id IN ?", typeIDs).Delete(&models.LifeEvent{})
			tx.Where("id IN ?", typeIDs).Delete(&models.LifeEventType{})
		}
		tx.Where("id IN ?", categoryIDs).Delete(&models.LifeEventCategory{})
	}

	// TimelineEvent cascade: LifeEvent (by timeline_event_id) → TimelineEventParticipant → TimelineEvent
	var timelineIDs []uint
	tx.Model(&models.TimelineEvent{}).Where("vault_id = ?", vaultID).Pluck("id", &timelineIDs)
	if len(timelineIDs) > 0 {
		tx.Where("timeline_event_id IN ?", timelineIDs).Delete(&models.LifeEvent{})
		tx.Where("timeline_event_id IN ?", timelineIDs).Delete(&models.TimelineEventParticipant{})
		tx.Where("id IN ?", timelineIDs).Delete(&models.TimelineEvent{})
	}

	// AddressBookSubscription cascade: DavSyncLog + ContactSubscriptionState → AddressBookSubscription
	var subIDs []string
	tx.Model(&models.AddressBookSubscription{}).Where("vault_id = ?", vaultID).Pluck("id", &subIDs)
	if len(subIDs) > 0 {
		tx.Where("address_book_subscription_id IN ?", subIDs).Delete(&models.DavSyncLog{})
		tx.Where("address_book_subscription_id IN ?", subIDs).Delete(&models.ContactSubscriptionState{})
		tx.Where("id IN ?", subIDs).Delete(&models.AddressBookSubscription{})
	}

	// --- Simple vault-level tables (no children of their own, or children already deleted) ---

	vaultChildModels := []interface{}{
		&models.ContactImportantDateType{},
		&models.MoodTrackingParameter{},
		&models.VaultQuickFactsTemplate{},
		&models.Label{},
		&models.Company{},
		&models.Group{},
		&models.Tag{},
		&models.Loan{},
		&models.File{},
		&models.Address{},
		&models.LifeMetric{},
		&models.Note{}, // Notes with vault_id but no contact (edge case safety)
		&models.ContactVaultUser{},
		&models.UserVault{},
	}
	for _, m := range vaultChildModels {
		if err := tx.Where("vault_id = ?", vaultID).Delete(m).Error; err != nil {
			return err
		}
	}

	// Step 4: Delete contacts (including soft-deleted ones via Unscoped).
	if err := tx.Unscoped().Where("vault_id = ?", vaultID).Delete(&models.Contact{}).Error; err != nil {
		return err
	}

	// Step 5: Delete the vault itself.
	return tx.Where("id = ?", vaultID).Delete(&models.Vault{}).Error
}

func (s *VaultService) CheckUserVaultAccess(userID, vaultID string, requiredPerm int) error {
	var uv models.UserVault
	if err := s.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVaultForbidden
		}
		return err
	}
	if uv.Permission > requiredPerm {
		return ErrInsufficientPerm
	}
	return nil
}

// createUserSelfContact creates a shadow Contact in the vault for a user (Monica v5 pattern).
// GORM zero-value bool trick: CanBeDeleted defaults to true, Listed defaults to true,
// so we must Create first then Update both to false.
func createUserSelfContact(tx *gorm.DB, userID, vaultID string) (string, error) {
	var user models.User
	if err := tx.First(&user, "id = ?", userID).Error; err != nil {
		return "", err
	}

	contact := models.Contact{
		VaultID:   vaultID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}
	if err := tx.Create(&contact).Error; err != nil {
		return "", err
	}
	// GORM skips false for bool fields with default:true — must update after create
	if err := tx.Model(&contact).Updates(map[string]interface{}{
		"can_be_deleted": false,
		"listed":         false,
	}).Error; err != nil {
		return "", err
	}
	return contact.ID, nil
}

func (s *VaultService) getUserContactID(userID, vaultID string) string {
	var uv models.UserVault
	if err := s.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err != nil {
		return ""
	}
	return uv.ContactID
}

func toVaultResponse(v *models.Vault, userContactID string) dto.VaultResponse {
	desc := ""
	if v.Description != nil {
		desc = *v.Description
	}
	return dto.VaultResponse{
		ID:                 v.ID,
		AccountID:          v.AccountID,
		Name:               v.Name,
		Description:        desc,
		DefaultActivityTab: v.DefaultActivityTab,
		UserContactID:      userContactID,
		CreatedAt:          v.CreatedAt,
		UpdatedAt:          v.UpdatedAt,
	}
}
