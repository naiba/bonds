package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupOAuthTest(t *testing.T) *OAuthService {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	return NewOAuthService(db, cfg, "http://localhost:8080")
}

func TestFindOrCreateUserNew(t *testing.T) {
	svc := setupOAuthTest(t)

	resp, err := svc.FindOrCreateUser("github", "gh-12345", "newuser@example.com", "John Doe", "en")
	if err != nil {
		t.Fatalf("FindOrCreateUser failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected token to be non-empty")
	}
	if resp.User.Email != "newuser@example.com" {
		t.Errorf("Expected email newuser@example.com, got %s", resp.User.Email)
	}
	if resp.User.FirstName != "John" {
		t.Errorf("Expected first name John, got %s", resp.User.FirstName)
	}
	if resp.User.LastName != "Doe" {
		t.Errorf("Expected last name Doe, got %s", resp.User.LastName)
	}
	if resp.User.AccountID == "" {
		t.Error("Expected account ID to be non-empty")
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "github", "gh-12345").First(&token).Error; err != nil {
		t.Fatalf("Expected UserToken to be created: %v", err)
	}
	if token.UserID != resp.User.ID {
		t.Errorf("Expected token user ID %s, got %s", resp.User.ID, token.UserID)
	}
}

func TestFindOrCreateUserExisting(t *testing.T) {
	svc := setupOAuthTest(t)

	resp1, err := svc.FindOrCreateUser("github", "gh-12345", "existing@example.com", "Jane Doe", "en")
	if err != nil {
		t.Fatalf("First FindOrCreateUser failed: %v", err)
	}

	resp2, err := svc.FindOrCreateUser("github", "gh-12345", "existing@example.com", "Jane Doe", "en")
	if err != nil {
		t.Fatalf("Second FindOrCreateUser failed: %v", err)
	}

	if resp2.User.ID != resp1.User.ID {
		t.Errorf("Expected same user ID %s, got %s", resp1.User.ID, resp2.User.ID)
	}
	if resp2.User.AccountID != resp1.User.AccountID {
		t.Errorf("Expected same account ID %s, got %s", resp1.User.AccountID, resp2.User.AccountID)
	}
}

func TestFindOrCreateUserLinkEmail(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	regResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Link",
		LastName:  "User",
		Email:     "link@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	existingUserID := regResp.User.ID

	resp, err := svc.FindOrCreateUser("google", "goo-99999", "link@example.com", "Link User", "en")
	if err != nil {
		t.Fatalf("FindOrCreateUser failed: %v", err)
	}

	if resp.User.ID != existingUserID {
		t.Errorf("Expected to link to existing user %s, got %s", existingUserID, resp.User.ID)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "google", "goo-99999").First(&token).Error; err != nil {
		t.Fatalf("Expected UserToken to be created: %v", err)
	}
	if token.UserID != existingUserID {
		t.Errorf("Expected token linked to user %s, got %s", existingUserID, token.UserID)
	}
}

func TestSaveTokenCreate(t *testing.T) {
	svc := setupOAuthTest(t)

	resp, err := svc.FindOrCreateUser("github", "gh-save", "save@example.com", "Save User", "en")
	if err != nil {
		t.Fatalf("FindOrCreateUser failed: %v", err)
	}

	err = svc.SaveToken(resp.User.ID, "github", "gh-save", "access-token-123", "refresh-token-456", 3600)
	if err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "github", "gh-save").First(&token).Error; err != nil {
		t.Fatalf("Token not found: %v", err)
	}
	if token.Token != "access-token-123" {
		t.Errorf("Expected access token 'access-token-123', got '%s'", token.Token)
	}
}

func TestSaveTokenUpdate(t *testing.T) {
	svc := setupOAuthTest(t)

	resp, err := svc.FindOrCreateUser("github", "gh-update", "update@example.com", "Update User", "en")
	if err != nil {
		t.Fatalf("FindOrCreateUser failed: %v", err)
	}

	err = svc.SaveToken(resp.User.ID, "github", "gh-update", "token-v1", "", 0)
	if err != nil {
		t.Fatalf("First SaveToken failed: %v", err)
	}

	err = svc.SaveToken(resp.User.ID, "github", "gh-update", "token-v2", "refresh-v2", 7200)
	if err != nil {
		t.Fatalf("Second SaveToken failed: %v", err)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "github", "gh-update").First(&token).Error; err != nil {
		t.Fatalf("Token not found: %v", err)
	}
	if token.Token != "token-v2" {
		t.Errorf("Expected access token 'token-v2', got '%s'", token.Token)
	}
}
