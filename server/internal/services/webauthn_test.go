package services

import (
	"testing"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestNewWebAuthnServiceEnabled(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.WebAuthnConfig{
		RPID:          "localhost",
		RPDisplayName: "Bonds Test",
		RPOrigins:     []string{"http://localhost:8080"},
	}
	svc, err := NewWebAuthnService(db, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if svc == nil {
		t.Error("Expected non-nil service when RPID is set")
	}
}

func setupWebAuthnTest(t *testing.T) (*WebAuthnService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	jwtCfg := testutil.TestJWTConfig()

	authSvc := NewAuthService(db, jwtCfg)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "WebAuthn",
		LastName:  "Test",
		Email:     "webauthn@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	cfg := &config.WebAuthnConfig{
		RPID:          "localhost",
		RPDisplayName: "Bonds Test",
		RPOrigins:     []string{"http://localhost:8080"},
	}
	svc, err := NewWebAuthnService(db, cfg)
	if err != nil {
		t.Fatalf("NewWebAuthnService failed: %v", err)
	}
	return svc, resp.User.ID
}

func TestListCredentialsEmpty(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	creds, err := svc.ListCredentials(userID)
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("Expected 0 credentials, got %d", len(creds))
	}
}

func TestListCredentialsWithData(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	cred := models.WebAuthnCredential{
		UserID:       userID,
		CredentialID: []byte("test-cred-id"),
		PublicKey:    []byte("test-public-key"),
		Name:         "My Key",
	}
	if err := svc.db.Create(&cred).Error; err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}

	creds, err := svc.ListCredentials(userID)
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("Expected 1 credential, got %d", len(creds))
	}
	if creds[0].Name != "My Key" {
		t.Errorf("Expected name 'My Key', got '%s'", creds[0].Name)
	}
}

func TestDeleteCredential(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	cred := models.WebAuthnCredential{
		UserID:       userID,
		CredentialID: []byte("delete-me"),
		PublicKey:    []byte("test-public-key"),
		Name:         "Delete Me",
	}
	if err := svc.db.Create(&cred).Error; err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}

	err := svc.DeleteCredential(cred.ID, userID)
	if err != nil {
		t.Fatalf("DeleteCredential failed: %v", err)
	}

	creds, err := svc.ListCredentials(userID)
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("Expected 0 credentials after delete, got %d", len(creds))
	}
}

func TestDeleteCredentialNotFound(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	err := svc.DeleteCredential(99999, userID)
	if err != ErrWebAuthnCredentialNotFound {
		t.Errorf("Expected ErrWebAuthnCredentialNotFound, got %v", err)
	}
}

func TestDeleteCredentialWrongUser(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	cred := models.WebAuthnCredential{
		UserID:       userID,
		CredentialID: []byte("wrong-user"),
		PublicKey:    []byte("test-public-key"),
		Name:         "Wrong User",
	}
	if err := svc.db.Create(&cred).Error; err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}

	err := svc.DeleteCredential(cred.ID, "different-user-id")
	if err != ErrWebAuthnCredentialNotFound {
		t.Errorf("Expected ErrWebAuthnCredentialNotFound for wrong user, got %v", err)
	}

	creds, err := svc.ListCredentials(userID)
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	if len(creds) != 1 {
		t.Errorf("Credential should still exist, expected 1, got %d", len(creds))
	}
}

func TestBeginRegistration(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	options, err := svc.BeginRegistration(userID)
	if err != nil {
		t.Fatalf("BeginRegistration failed: %v", err)
	}
	if options == nil {
		t.Error("Expected non-nil options")
	}
}

func TestBeginLoginNoCredentials(t *testing.T) {
	svc, _ := setupWebAuthnTest(t)

	_, err := svc.BeginLogin("webauthn@example.com")
	if err != ErrWebAuthnNoCredentials {
		t.Errorf("Expected ErrWebAuthnNoCredentials, got %v", err)
	}
}

func TestBeginLoginUserNotFound(t *testing.T) {
	svc, _ := setupWebAuthnTest(t)

	_, err := svc.BeginLogin("nonexistent@example.com")
	if err != ErrWebAuthnUserNotFound {
		t.Errorf("Expected ErrWebAuthnUserNotFound, got %v", err)
	}
}
