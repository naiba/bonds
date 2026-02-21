package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupNotificationTest(t *testing.T) (*NotificationService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "notifications-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewNotificationService(db), resp.User.ID
}

func TestNotificationCreate(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	ch, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:          "email",
		Label:         "Work Email",
		Content:       "work@example.com",
		PreferredTime: "09:00",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if ch.Type != "email" {
		t.Errorf("Expected type 'email', got '%s'", ch.Type)
	}
	if ch.Label != "Work Email" {
		t.Errorf("Expected label 'Work Email', got '%s'", ch.Label)
	}
	if ch.Content != "work@example.com" {
		t.Errorf("Expected content 'work@example.com', got '%s'", ch.Content)
	}
	if ch.PreferredTime != "09:00" {
		t.Errorf("Expected preferred_time '09:00', got '%s'", ch.PreferredTime)
	}
	if ch.ID == 0 {
		t.Error("Expected ID to be non-zero")
	}
}

func TestNotificationList(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	_, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "email",
		Content: "one@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "telegram",
		Content: "@testbot",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	channels, err := svc.List(userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(channels) != 3 {
		t.Errorf("Expected 3 channels (1 seeded + 2 created), got %d", len(channels))
	}
}

func TestNotificationToggle(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "email",
		Content: "toggle@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Active {
		t.Error("Expected new channel to be inactive")
	}

	toggled, err := svc.Toggle(created.ID, userID)
	if err != nil {
		t.Fatalf("Toggle failed: %v", err)
	}
	if !toggled.Active {
		t.Error("Expected channel to be active after toggle")
	}

	toggledBack, err := svc.Toggle(created.ID, userID)
	if err != nil {
		t.Fatalf("Toggle back failed: %v", err)
	}
	if toggledBack.Active {
		t.Error("Expected channel to be inactive after second toggle")
	}
}

func TestNotificationDelete(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "email",
		Content: "delete@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, userID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	channels, err := svc.List(userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("Expected 1 channel after delete (seeded channel remains), got %d", len(channels))
	}
}

func TestNotificationUpdate(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:          "email",
		Label:         "Old Label",
		Content:       "old@example.com",
		PreferredTime: "09:00",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, userID, dto.UpdateNotificationChannelRequest{
		Label:         "New Label",
		Content:       "old@example.com",
		PreferredTime: "10:00",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "New Label" {
		t.Errorf("Expected label 'New Label', got '%s'", updated.Label)
	}
	if updated.PreferredTime != "10:00" {
		t.Errorf("Expected preferred_time '10:00', got '%s'", updated.PreferredTime)
	}
	if updated.Content != "old@example.com" {
		t.Errorf("Expected content unchanged, got '%s'", updated.Content)
	}
}

func TestNotificationUpdateContentResetsVerification(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "notif-update-verify@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	userID := resp.User.ID

	_, err = vaultSvc.CreateVault(resp.User.AccountID, userID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewNotificationService(db)

	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "email",
		Label:   "Email Channel",
		Content: "original@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var dbCh models.UserNotificationChannel
	db.First(&dbCh, created.ID)
	token := *dbCh.VerificationToken
	if err := svc.Verify(created.ID, userID, token); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	verified, err := svc.List(userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	var found bool
	for _, ch := range verified {
		if ch.ID == created.ID {
			found = true
			if ch.VerifiedAt == nil {
				t.Fatal("Expected channel to be verified before update")
			}
		}
	}
	if !found {
		t.Fatal("Created channel not found in list")
	}

	updated, err := svc.Update(created.ID, userID, dto.UpdateNotificationChannelRequest{
		Label:   "Email Channel",
		Content: "changed@example.com",
	})
	if err != nil {
		t.Fatalf("Update with changed content failed: %v", err)
	}
	if updated.VerifiedAt != nil {
		t.Error("Expected VerifiedAt to be nil after content change")
	}
	if updated.Active {
		t.Error("Expected channel to be inactive after content change")
	}
	if updated.Content != "changed@example.com" {
		t.Errorf("Expected content 'changed@example.com', got '%s'", updated.Content)
	}
}

func TestNotificationUpdateShoutrrrKeepsVerified(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "shoutrrr",
		Label:   "Telegram Bot",
		Content: "telegram://token@telegram?channels=123",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !created.Active {
		t.Error("Expected shoutrrr channel to be auto-active")
	}

	updated, err := svc.Update(created.ID, userID, dto.UpdateNotificationChannelRequest{
		Label:   "Telegram Bot Updated",
		Content: "telegram://token@telegram?channels=456",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.VerifiedAt == nil {
		t.Error("Expected shoutrrr channel to remain verified after content change")
	}
	if updated.Label != "Telegram Bot Updated" {
		t.Errorf("Expected label 'Telegram Bot Updated', got '%s'", updated.Label)
	}
}

func TestNotificationUpdateNotFound(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	_, err := svc.Update(9999, userID, dto.UpdateNotificationChannelRequest{
		Label:   "Whatever",
		Content: "test@example.com",
	})
	if err != ErrNotificationChannelNotFound {
		t.Errorf("Expected ErrNotificationChannelNotFound, got %v", err)
	}
}

func TestNotificationNotFound(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	err := svc.Delete(9999, userID)
	if err != ErrNotificationChannelNotFound {
		t.Errorf("Delete: expected ErrNotificationChannelNotFound, got %v", err)
	}

	_, err = svc.Toggle(9999, userID)
	if err != ErrNotificationChannelNotFound {
		t.Errorf("Toggle: expected ErrNotificationChannelNotFound, got %v", err)
	}
}

func TestNotificationCreateGeneratesVerificationToken(t *testing.T) {
	svc, userID := setupNotificationTest(t)

	ch, err := svc.Create(userID, dto.CreateNotificationChannelRequest{
		Type:    "email",
		Content: "verify@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var dbCh models.UserNotificationChannel
	if err := svc.db.First(&dbCh, ch.ID).Error; err != nil {
		t.Fatalf("Failed to load channel: %v", err)
	}
	if dbCh.VerificationToken == nil || *dbCh.VerificationToken == "" {
		t.Error("Expected VerificationToken to be set")
	}
}

func TestNotificationVerifySchedulesReminders(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "verify-schedule@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	reminderSvc := NewReminderService(db)
	day, month := 15, 6
	_, err = reminderSvc.Create(contact.ID, vault.ID, dto.CreateReminderRequest{
		Label: "Birthday", Day: &day, Month: &month, Type: "one_time",
	})
	if err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	notifSvc := NewNotificationService(db)
	ch, err := notifSvc.Create(resp.User.ID, dto.CreateNotificationChannelRequest{
		Type: "email", Content: "verify-sched@example.com", PreferredTime: "09:00",
	})
	if err != nil {
		t.Fatalf("Create notification channel failed: %v", err)
	}

	var dbCh models.UserNotificationChannel
	db.First(&dbCh, ch.ID)
	token := *dbCh.VerificationToken

	if err := notifSvc.Verify(ch.ID, resp.User.ID, token); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	var scheduledCount int64
	db.Model(&models.ContactReminderScheduled{}).
		Where("user_notification_channel_id = ?", ch.ID).
		Count(&scheduledCount)
	if scheduledCount == 0 {
		t.Error("Expected at least 1 scheduled reminder after Verify")
	}
}
