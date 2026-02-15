package services

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestTelegramWebhookHandleUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewTelegramWebhookService(db)

	userID := "test-user-id"
	token := "test-verification-token"
	ch := models.UserNotificationChannel{
		UserID:            &userID,
		Type:              "telegram",
		Content:           "",
		VerificationToken: &token,
	}
	if err := db.Create(&ch).Error; err != nil {
		t.Fatalf("Create channel failed: %v", err)
	}

	update := TelegramUpdate{
		Message: &TelegramMessage{
			Chat: TelegramChat{ID: 123456789},
			Text: "/start " + token,
		},
	}

	if err := svc.HandleUpdate(update); err != nil {
		t.Fatalf("HandleUpdate failed: %v", err)
	}

	var updated models.UserNotificationChannel
	if err := db.First(&updated, ch.ID).Error; err != nil {
		t.Fatalf("Find channel failed: %v", err)
	}
	if updated.Content != "123456789" {
		t.Errorf("Expected content '123456789', got '%s'", updated.Content)
	}
	if updated.VerifiedAt == nil {
		t.Error("Expected VerifiedAt to be set")
	}
	if !updated.Active {
		t.Error("Expected channel to be active")
	}
}

func TestTelegramWebhookNilMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewTelegramWebhookService(db)

	err := svc.HandleUpdate(TelegramUpdate{Message: nil})
	if err != ErrInvalidTelegramUpdate {
		t.Errorf("Expected ErrInvalidTelegramUpdate, got %v", err)
	}
}

func TestTelegramWebhookNonStartMessage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewTelegramWebhookService(db)

	err := svc.HandleUpdate(TelegramUpdate{
		Message: &TelegramMessage{
			Chat: TelegramChat{ID: 123},
			Text: "hello world",
		},
	})
	if err != nil {
		t.Errorf("Expected nil error for non-start message, got %v", err)
	}
}

func TestTelegramWebhookInvalidToken(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewTelegramWebhookService(db)

	err := svc.HandleUpdate(TelegramUpdate{
		Message: &TelegramMessage{
			Chat: TelegramChat{ID: 123},
			Text: "/start invalid-token",
		},
	})
	if err != ErrTelegramChannelNotFound {
		t.Errorf("Expected ErrTelegramChannelNotFound, got %v", err)
	}
}

func TestTelegramWebhookCreateChannel(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewTelegramWebhookService(db)

	ch, token, err := svc.CreateChannel("user-123")
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}
	if ch.ID == 0 {
		t.Error("Expected non-zero channel ID")
	}
	if token == "" {
		t.Error("Expected non-empty token")
	}
	if ch.Type != "telegram" {
		t.Errorf("Expected type 'telegram', got '%s'", ch.Type)
	}
}
