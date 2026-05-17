package services

import (
	"errors"
	"fmt"

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
		return fmt.Errorf("collect contacts: %w", err)
	}

	// Step 2: Delete contact-level children (deepest grandchildren first).
	if len(contactIDs) > 0 {
		// --- Grandchildren (depend on contact children) ---

		// ContactReminderScheduled → depends on ContactReminder
		if err := tx.Where("contact_reminder_id IN (?)",
			tx.Model(&models.ContactReminder{}).Select("id").Where("contact_id IN ?", contactIDs),
		).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
			return fmt.Errorf("delete ContactReminderScheduled: %w", err)
		}

		// Streak → depends on Goal
		if err := tx.Where("goal_id IN (?)",
			tx.Model(&models.Goal{}).Select("id").Where("contact_id IN ?", contactIDs),
		).Delete(&models.Streak{}).Error; err != nil {
			return fmt.Errorf("delete Streak: %w", err)
		}

		// --- Contact-level children (direct FK to contact_id) ---

		contactChildModels := []interface{}{
			&models.ContactInformation{},
			&models.ContactImportantDate{},
			&models.ContactReminder{},
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
		// Hard-delete (Unscoped) is required: soft-deletable child models
		// (ContactImportantDate here; Group and ContactTask in the vault-scoped
		// path further down) carry gorm.DeletedAt, so a regular Delete leaves
		// the row in the table with its FKs to vault-scoped parents intact
		// (e.g. contact_important_dates → contact_important_date_types).
		// Postgres then rejects the parent delete in Step 3 with a foreign-key
		// violation. A vault delete is intentionally destructive, so
		// soft-delete is the wrong semantic here.
		// NB: ContactTask is intentionally NOT in this list — since #107 it
		// has no contact_id column (assignees live in the task_contacts pivot
		// instead), so deleting by contact_id would be a SQL error. The
		// dedicated ContactTask cascade further down handles tasks by id and
		// drops the task_contacts pivot rows first to satisfy the FK.
		for _, m := range contactChildModels {
			if err := tx.Unscoped().Where("contact_id IN ?", contactIDs).Delete(m).Error; err != nil {
				return fmt.Errorf("delete contact child %T: %w", m, err)
			}
		}

		// Also delete Relationships where this vault's contacts are the "related" side
		if err := tx.Unscoped().Where("related_contact_id IN ?", contactIDs).Delete(&models.Relationship{}).Error; err != nil {
			return fmt.Errorf("delete Relationship by related_contact_id: %w", err)
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
			if err := tx.Unscoped().Where("contact_id IN ?", contactIDs).Delete(m).Error; err != nil {
				return fmt.Errorf("delete contact pivot %T: %w", m, err)
			}
		}

		// ContactLoan pivot uses loaner_id / loanee_id instead of contact_id
		if err := tx.Unscoped().Where("loaner_id IN ? OR loanee_id IN ?", contactIDs, contactIDs).
			Delete(&models.ContactLoan{}).Error; err != nil {
			return fmt.Errorf("delete ContactLoan: %w", err)
		}

		// ContactGift pivot uses loaner_id / loanee_id (same pattern as ContactLoan)
		if err := tx.Unscoped().Where("loaner_id IN ? OR loanee_id IN ?", contactIDs, contactIDs).
			Delete(&models.ContactGift{}).Error; err != nil {
			return fmt.Errorf("delete ContactGift: %w", err)
		}

		// DavSyncLog has nullable contact_id
		if err := tx.Unscoped().Where("contact_id IN ?", contactIDs).Delete(&models.DavSyncLog{}).Error; err != nil {
			return fmt.Errorf("delete DavSyncLog by contact_id: %w", err)
		}
	}

	// Step 3: Delete vault-level children (grandchildren of vault-scoped tables first).

	// Journal cascade: PostMetric → PostSection → PostTag → ContactPost → Post → SliceOfLife → JournalMetric → Journal
	var journalIDs []uint
	if err := tx.Model(&models.Journal{}).Where("vault_id = ?", vaultID).Pluck("id", &journalIDs).Error; err != nil {
		return fmt.Errorf("pluck Journal ids: %w", err)
	}
	if len(journalIDs) > 0 {
		var postIDs []uint
		if err := tx.Model(&models.Post{}).Where("journal_id IN ?", journalIDs).Pluck("id", &postIDs).Error; err != nil {
			return fmt.Errorf("pluck Post ids: %w", err)
		}
		if len(postIDs) > 0 {
			if err := tx.Where("post_id IN ?", postIDs).Delete(&models.PostMetric{}).Error; err != nil {
				return fmt.Errorf("delete PostMetric: %w", err)
			}
			if err := tx.Where("post_id IN ?", postIDs).Delete(&models.PostSection{}).Error; err != nil {
				return fmt.Errorf("delete PostSection: %w", err)
			}
			if err := tx.Where("post_id IN ?", postIDs).Delete(&models.PostTag{}).Error; err != nil {
				return fmt.Errorf("delete PostTag: %w", err)
			}
			if err := tx.Where("post_id IN ?", postIDs).Delete(&models.ContactPost{}).Error; err != nil {
				return fmt.Errorf("delete ContactPost by post_id: %w", err)
			}
			if err := tx.Where("id IN ?", postIDs).Delete(&models.Post{}).Error; err != nil {
				return fmt.Errorf("delete Post: %w", err)
			}
		}

		var journalMetricIDs []uint
		if err := tx.Model(&models.JournalMetric{}).Where("journal_id IN ?", journalIDs).Pluck("id", &journalMetricIDs).Error; err != nil {
			return fmt.Errorf("pluck JournalMetric ids: %w", err)
		}
		if len(journalMetricIDs) > 0 {
			// PostMetric references JournalMetricID — already deleted above via postIDs
			if err := tx.Where("id IN ?", journalMetricIDs).Delete(&models.JournalMetric{}).Error; err != nil {
				return fmt.Errorf("delete JournalMetric: %w", err)
			}
		}

		if err := tx.Where("journal_id IN ?", journalIDs).Delete(&models.SliceOfLife{}).Error; err != nil {
			return fmt.Errorf("delete SliceOfLife: %w", err)
		}
		if err := tx.Where("id IN ?", journalIDs).Delete(&models.Journal{}).Error; err != nil {
			return fmt.Errorf("delete Journal: %w", err)
		}
	}

	// LifeEventCategory cascade: LifeEvent → LifeEventType → LifeEventCategory
	var categoryIDs []uint
	if err := tx.Model(&models.LifeEventCategory{}).Where("vault_id = ?", vaultID).Pluck("id", &categoryIDs).Error; err != nil {
		return fmt.Errorf("pluck LifeEventCategory ids: %w", err)
	}
	if len(categoryIDs) > 0 {
		var typeIDs []uint
		if err := tx.Model(&models.LifeEventType{}).Where("life_event_category_id IN ?", categoryIDs).Pluck("id", &typeIDs).Error; err != nil {
			return fmt.Errorf("pluck LifeEventType ids: %w", err)
		}
		if len(typeIDs) > 0 {
			if err := tx.Where("life_event_type_id IN ?", typeIDs).Delete(&models.LifeEvent{}).Error; err != nil {
				return fmt.Errorf("delete LifeEvent by life_event_type_id: %w", err)
			}
			if err := tx.Where("id IN ?", typeIDs).Delete(&models.LifeEventType{}).Error; err != nil {
				return fmt.Errorf("delete LifeEventType: %w", err)
			}
		}
		if err := tx.Where("id IN ?", categoryIDs).Delete(&models.LifeEventCategory{}).Error; err != nil {
			return fmt.Errorf("delete LifeEventCategory: %w", err)
		}
	}

	// TimelineEvent cascade: LifeEvent (by timeline_event_id) → TimelineEventParticipant → TimelineEvent
	var timelineIDs []uint
	if err := tx.Model(&models.TimelineEvent{}).Where("vault_id = ?", vaultID).Pluck("id", &timelineIDs).Error; err != nil {
		return fmt.Errorf("pluck TimelineEvent ids: %w", err)
	}
	if len(timelineIDs) > 0 {
		if err := tx.Where("timeline_event_id IN ?", timelineIDs).Delete(&models.LifeEvent{}).Error; err != nil {
			return fmt.Errorf("delete LifeEvent by timeline_event_id: %w", err)
		}
		if err := tx.Where("timeline_event_id IN ?", timelineIDs).Delete(&models.TimelineEventParticipant{}).Error; err != nil {
			return fmt.Errorf("delete TimelineEventParticipant: %w", err)
		}
		if err := tx.Where("id IN ?", timelineIDs).Delete(&models.TimelineEvent{}).Error; err != nil {
			return fmt.Errorf("delete TimelineEvent: %w", err)
		}
	}

	// AddressBookSubscription cascade: DavSyncLog + ContactSubscriptionState → AddressBookSubscription
	var subIDs []string
	if err := tx.Model(&models.AddressBookSubscription{}).Where("vault_id = ?", vaultID).Pluck("id", &subIDs).Error; err != nil {
		return fmt.Errorf("pluck AddressBookSubscription ids: %w", err)
	}
	if len(subIDs) > 0 {
		if err := tx.Where("address_book_subscription_id IN ?", subIDs).Delete(&models.DavSyncLog{}).Error; err != nil {
			return fmt.Errorf("delete DavSyncLog by address_book_subscription_id: %w", err)
		}
		if err := tx.Where("address_book_subscription_id IN ?", subIDs).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
			return fmt.Errorf("delete ContactSubscriptionState by address_book_subscription_id: %w", err)
		}
		if err := tx.Where("id IN ?", subIDs).Delete(&models.AddressBookSubscription{}).Error; err != nil {
			return fmt.Errorf("delete AddressBookSubscription: %w", err)
		}
	}

	// --- Cross-vault FK cleanup ---
	//
	// Step 2 deletes child rows by contact_id IN (this vault's contacts), but a
	// contact in ANOTHER vault may still hold a child row whose catalog FK
	// points at one of THIS vault's catalog rows (e.g. a contact in another
	// vault has a QuickFact filed under this vault's template). Those rows
	// keep the catalog FK constraint alive and would block Step 3's catalog
	// deletes with a Postgres FK violation.
	//
	// For nullable FKs, NULL them — preserves the child row in the other vault.
	// For NOT NULL FKs, hard-delete the cross-vault child row.

	type nullCleanup struct {
		instance, catalog interface{}
		instanceFKCol     string
	}
	for _, n := range []nullCleanup{
		// ContactImportantDate.contact_important_date_type_id → ContactImportantDateType (nullable)
		{&models.ContactImportantDate{}, &models.ContactImportantDateType{}, "contact_important_date_type_id"},
		// Contact.company_id → Company (nullable on Contact)
		{&models.Contact{}, &models.Company{}, "company_id"},
		// Contact.file_id → File (nullable on Contact)
		{&models.Contact{}, &models.File{}, "file_id"},
	} {
		if err := tx.Unscoped().Model(n.instance).
			Where(n.instanceFKCol+" IN (?)",
				tx.Unscoped().Model(n.catalog).Select("id").Where("vault_id = ?", vaultID),
			).
			Update(n.instanceFKCol, nil).Error; err != nil {
			return fmt.Errorf("null cross-vault %T.%s: %w", n.instance, n.instanceFKCol, err)
		}
	}

	type deleteCleanup struct {
		instance, catalog interface{}
		instanceFKCol     string
	}
	for _, d := range []deleteCleanup{
		// QuickFact.vault_quick_facts_template_id → VaultQuickFactsTemplate (NOT NULL)
		{&models.QuickFact{}, &models.VaultQuickFactsTemplate{}, "vault_quick_facts_template_id"},
		// MoodTrackingEvent.mood_tracking_parameter_id → MoodTrackingParameter (NOT NULL)
		{&models.MoodTrackingEvent{}, &models.MoodTrackingParameter{}, "mood_tracking_parameter_id"},
		// ContactLabel.label_id → Label (NOT NULL pivot)
		{&models.ContactLabel{}, &models.Label{}, "label_id"},
		// ContactGroup.group_id → Group (NOT NULL pivot)
		{&models.ContactGroup{}, &models.Group{}, "group_id"},
		// ContactCompany.company_id → Company (NOT NULL pivot)
		{&models.ContactCompany{}, &models.Company{}, "company_id"},
		// ContactAddress.address_id → Address (NOT NULL pivot)
		{&models.ContactAddress{}, &models.Address{}, "address_id"},
		// ContactLifeMetric.life_metric_id → LifeMetric (NOT NULL pivot)
		{&models.ContactLifeMetric{}, &models.LifeMetric{}, "life_metric_id"},
		// ContactLoan.loan_id → Loan (NOT NULL pivot)
		{&models.ContactLoan{}, &models.Loan{}, "loan_id"},
	} {
		if err := tx.Unscoped().
			Where(d.instanceFKCol+" IN (?)",
				tx.Unscoped().Model(d.catalog).Select("id").Where("vault_id = ?", vaultID),
			).
			Delete(d.instance).Error; err != nil {
			return fmt.Errorf("delete cross-vault %T by %s: %w", d.instance, d.instanceFKCol, err)
		}
	}

	// ContactTask cascade: TaskContact pivot rows must go first since they
	// reference the task by FK, then the tasks themselves. Pluck is Unscoped
	// so soft-deleted ContactTask rows are also caught — otherwise their
	// lingering TaskContact rows would block the Unscoped delete that the
	// vaultChildModels safety net runs below.
	var taskIDs []uint
	if err := tx.Model(&models.ContactTask{}).Unscoped().Where("vault_id = ?", vaultID).Pluck("id", &taskIDs).Error; err != nil {
		return fmt.Errorf("pluck ContactTask ids: %w", err)
	}
	if len(taskIDs) > 0 {
		if err := tx.Where("contact_task_id IN ?", taskIDs).Delete(&models.TaskContact{}).Error; err != nil {
			return fmt.Errorf("delete TaskContact: %w", err)
		}
		if err := tx.Unscoped().Where("id IN ?", taskIDs).Delete(&models.ContactTask{}).Error; err != nil {
			return fmt.Errorf("delete ContactTask: %w", err)
		}
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
		&models.Note{},        // Notes with vault_id but no contact (edge case safety)
		&models.ContactTask{}, // Standalone vault tasks have vault_id but no contact_id.
		&models.ContactVaultUser{},
		&models.UserVault{},
	}
	for _, m := range vaultChildModels {
		if err := tx.Unscoped().Where("vault_id = ?", vaultID).Delete(m).Error; err != nil {
			return fmt.Errorf("delete vault child %T: %w", m, err)
		}
	}

	// Step 4: Delete contacts (including soft-deleted ones via Unscoped).
	if err := tx.Unscoped().Where("vault_id = ?", vaultID).Delete(&models.Contact{}).Error; err != nil {
		return fmt.Errorf("delete Contact: %w", err)
	}

	// Step 5: Delete the vault itself.
	if err := tx.Where("id = ?", vaultID).Delete(&models.Vault{}).Error; err != nil {
		return fmt.Errorf("delete Vault: %w", err)
	}
	return nil
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
