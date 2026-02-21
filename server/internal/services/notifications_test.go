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
