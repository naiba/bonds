package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
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
	})
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

	toggled, err := svc.Toggle(created.ID)
	if err != nil {
		t.Fatalf("Toggle failed: %v", err)
	}
	if !toggled.Active {
		t.Error("Expected channel to be active after toggle")
	}

	toggledBack, err := svc.Toggle(created.ID)
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

	if err := svc.Delete(created.ID); err != nil {
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

func TestNotificationDeleteNotFound(t *testing.T) {
	svc, _ := setupNotificationTest(t)

	err := svc.Delete(9999)
	if err != ErrNotificationChannelNotFound {
		t.Errorf("Expected ErrNotificationChannelNotFound, got %v", err)
	}
}

func TestNotificationToggleNotFound(t *testing.T) {
	svc, _ := setupNotificationTest(t)

	_, err := svc.Toggle(9999)
	if err != ErrNotificationChannelNotFound {
		t.Errorf("Expected ErrNotificationChannelNotFound, got %v", err)
	}
}
