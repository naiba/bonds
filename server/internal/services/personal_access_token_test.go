package services

import (
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPATTest(t *testing.T) (*PersonalAccessTokenService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "PAT",
		LastName:  "Tester",
		Email:     "pat-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	svc := NewPersonalAccessTokenService(db)
	return svc, resp.User.ID, resp.User.AccountID
}

func TestPersonalAccessTokenService_Create(t *testing.T) {
	svc, userID, accountID := setupPATTest(t)

	resp, err := svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "Test Token",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !strings.HasPrefix(resp.Token, "bonds_") {
		t.Errorf("Expected token to start with 'bonds_', got '%s'", resp.Token)
	}

	if len(resp.Token) != 86 {
		t.Errorf("Expected token length 86, got %d", len(resp.Token))
	}

	expectedHint := "..." + resp.Token[len(resp.Token)-6:]
	if resp.TokenHint != expectedHint {
		t.Errorf("Expected token hint '%s', got '%s'", expectedHint, resp.TokenHint)
	}

	if resp.Name != "Test Token" {
		t.Errorf("Expected name 'Test Token', got '%s'", resp.Name)
	}

	if resp.ID == 0 {
		t.Error("Expected non-zero ID")
	}

	var stored models.PersonalAccessToken
	if err := svc.db.First(&stored, resp.ID).Error; err != nil {
		t.Fatalf("Failed to load token from DB: %v", err)
	}
	if stored.TokenHash == resp.Token {
		t.Error("Token stored in plaintext, expected a hash")
	}
	if stored.TokenHash == "" {
		t.Error("TokenHash should not be empty")
	}
}

func TestPersonalAccessTokenService_CreateDuplicateName(t *testing.T) {
	svc, userID, accountID := setupPATTest(t)

	_, err := svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "Duplicate",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "Duplicate",
	})
	if err != ErrTokenNameDuplicate {
		t.Errorf("Expected ErrTokenNameDuplicate, got: %v", err)
	}
}

func TestPersonalAccessTokenService_List(t *testing.T) {
	svc, userID, accountID := setupPATTest(t)

	_, err := svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "Token A",
	})
	if err != nil {
		t.Fatalf("Create token A failed: %v", err)
	}

	_, err = svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "Token B",
	})
	if err != nil {
		t.Fatalf("Create token B failed: %v", err)
	}

	list, err := svc.List(userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(list))
	}
}

func TestPersonalAccessTokenService_Delete(t *testing.T) {
	svc, userID, accountID := setupPATTest(t)

	resp, err := svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "To Delete",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(resp.ID, userID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, err := svc.List(userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected 0 tokens after delete, got %d", len(list))
	}
}

func TestPersonalAccessTokenService_DeleteNotFound(t *testing.T) {
	svc, userID, _ := setupPATTest(t)

	err := svc.Delete(99999, userID)
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got: %v", err)
	}
}

func TestPersonalAccessTokenService_ValidateToken(t *testing.T) {
	svc, userID, accountID := setupPATTest(t)

	resp, err := svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name: "Validate Me",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	pat, err := svc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if pat.ID != resp.ID {
		t.Errorf("Expected token ID %d, got %d", resp.ID, pat.ID)
	}
	if pat.UserID != userID {
		t.Errorf("Expected user ID '%s', got '%s'", userID, pat.UserID)
	}
	if pat.Name != "Validate Me" {
		t.Errorf("Expected name 'Validate Me', got '%s'", pat.Name)
	}
	if pat.LastUsedAt == nil {
		t.Error("Expected LastUsedAt to be set after validation")
	}
}

func TestPersonalAccessTokenService_ValidateExpiredToken(t *testing.T) {
	svc, userID, accountID := setupPATTest(t)

	pastTime := time.Now().Add(-24 * time.Hour)
	resp, err := svc.Create(userID, accountID, dto.CreatePersonalAccessTokenRequest{
		Name:      "Expired Token",
		ExpiresAt: &pastTime,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = svc.ValidateToken(resp.Token)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got: %v", err)
	}
}

func TestPersonalAccessTokenService_ValidateInvalidToken(t *testing.T) {
	svc, _, _ := setupPATTest(t)

	_, err := svc.ValidateToken("bonds_totally_invalid_random_string")
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got: %v", err)
	}
}
