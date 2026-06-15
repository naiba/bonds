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
	}, "en")
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

func setupWebAuthnTestWithEmptyEnvConfig(t *testing.T) (*WebAuthnService, *SystemSettingService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	jwtCfg := testutil.TestJWTConfig()

	authSvc := NewAuthService(db, jwtCfg)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "WebAuthn",
		LastName:  "Test",
		Email:     "webauthn@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	svc, err := NewWebAuthnService(db, &config.WebAuthnConfig{})
	if err != nil {
		t.Fatalf("NewWebAuthnService failed: %v", err)
	}

	settings := NewSystemSettingService(db)
	if err := settings.Set("webauthn.rp_id", "localhost"); err != nil {
		t.Fatalf("Failed to seed rp_id: %v", err)
	}
	if err := settings.Set("webauthn.rp_display_name", "Bonds Test"); err != nil {
		t.Fatalf("Failed to seed rp_display_name: %v", err)
	}
	if err := settings.Set("webauthn.rp_origins", "http://localhost:8080"); err != nil {
		t.Fatalf("Failed to seed rp_origins: %v", err)
	}

	return svc, settings, resp.User.ID
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

func TestBeginRegistrationReloadsPersistedDBConfigAfterAttachingSystemSettings(t *testing.T) {
	svc, settings, userID := setupWebAuthnTestWithEmptyEnvConfig(t)
	svc.SetSystemSettings(settings)

	if !svc.IsEnabled() {
		t.Fatal("Expected WebAuthn to be enabled from persisted DB settings")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("BeginRegistration panicked after attaching persisted DB settings: %v", r)
		}
	}()

	options, err := svc.BeginRegistration(userID)
	if err != nil {
		t.Fatalf("BeginRegistration failed: %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil registration options after loading persisted DB settings")
	}
}

func TestReloadConfigTrimsDBRpOrigins(t *testing.T) {
	svc, settings, _ := setupWebAuthnTestWithEmptyEnvConfig(t)
	if err := settings.Set("webauthn.rp_origins", "https://bonds.example.com, https://www.bonds.example.com"); err != nil {
		t.Fatalf("Failed to update rp_origins: %v", err)
	}

	svc.SetSystemSettings(settings)
	if err := svc.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}

	if svc.webauthn == nil {
		t.Fatal("Expected WebAuthn config to be built from persisted DB settings")
	}

	expected := []string{"https://bonds.example.com", "https://www.bonds.example.com"}
	if len(svc.webauthn.Config.RPOrigins) != len(expected) {
		t.Fatalf("Expected %d RP origins, got %d: %#v", len(expected), len(svc.webauthn.Config.RPOrigins), svc.webauthn.Config.RPOrigins)
	}
	for i := range expected {
		if svc.webauthn.Config.RPOrigins[i] != expected[i] {
			t.Fatalf("Expected RP origin %d to be %q, got %q", i, expected[i], svc.webauthn.Config.RPOrigins[i])
		}
	}
}

func TestInvalidPersistedDBWebAuthnConfigIsDisabled(t *testing.T) {
	svc, settings, userID := setupWebAuthnTestWithEmptyEnvConfig(t)
	if err := settings.Set("webauthn.rp_origins", "   "); err != nil {
		t.Fatalf("Failed to clear rp_origins: %v", err)
	}

	svc.SetSystemSettings(settings)

	if svc.IsEnabled() {
		t.Fatal("Expected WebAuthn to stay disabled when persisted DB config is invalid")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("BeginRegistration panicked with invalid persisted DB settings: %v", r)
		}
	}()

	_, err := svc.BeginRegistration(userID)
	if err != ErrWebAuthnNotConfigured {
		t.Fatalf("Expected ErrWebAuthnNotConfigured, got %v", err)
	}
}

func TestReloadConfigDisablesPreviouslyEnabledWebAuthnWhenPersistedConfigIsInvalid(t *testing.T) {
	svc, settings, userID := setupWebAuthnTestWithEmptyEnvConfig(t)
	svc.SetSystemSettings(settings)

	if !svc.IsEnabled() {
		t.Fatal("Expected WebAuthn to be enabled from initial persisted DB settings")
	}

	invalidSettings := []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty rp id", key: "webauthn.rp_id", value: ""},
		{name: "blank rp origins", key: "webauthn.rp_origins", value: "   "},
	}

	for _, tt := range invalidSettings {
		t.Run(tt.name, func(t *testing.T) {
			svc.SetSystemSettings(settings)
			if !svc.IsEnabled() {
				t.Fatal("Expected WebAuthn to be enabled before invalid persisted setting is applied")
			}

			if err := settings.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Failed to apply invalid setting: %v", err)
			}
			_ = svc.ReloadConfig()

			if svc.IsEnabled() {
				t.Fatal("Expected WebAuthn to be disabled after invalid persisted DB settings")
			}

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("BeginRegistration panicked after invalid persisted DB settings: %v", r)
				}
			}()

			_, err := svc.BeginRegistration(userID)
			if err != ErrWebAuthnNotConfigured {
				t.Fatalf("Expected ErrWebAuthnNotConfigured, got %v", err)
			}

			if err := settings.Set("webauthn.rp_id", "localhost"); err != nil {
				t.Fatalf("Failed to restore rp_id: %v", err)
			}
			if err := settings.Set("webauthn.rp_origins", "http://localhost:8080"); err != nil {
				t.Fatalf("Failed to restore rp_origins: %v", err)
			}
		})
	}
}

func TestWebAuthnCredentialsRehydratesPersistedFlags(t *testing.T) {
	// Regression for #174: go-webauthn rejects login when the stored
	// credential's BackupEligible flag disagrees with the assertion, so dropped
	// flags break every synced-passkey login.
	svc, userID := setupWebAuthnTest(t)

	cred := models.WebAuthnCredential{
		UserID:         userID,
		CredentialID:   []byte("synced-passkey"),
		PublicKey:      []byte("test-public-key"),
		Name:           "iCloud Keychain",
		UserPresent:    true,
		UserVerified:   true,
		BackupEligible: true,
		BackupState:    true,
	}
	if err := svc.db.Create(&cred).Error; err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}

	var loaded []models.WebAuthnCredential
	if err := svc.db.Where("user_id = ?", userID).Find(&loaded).Error; err != nil {
		t.Fatalf("Failed to load credentials: %v", err)
	}

	user := &models.User{ID: userID, Email: "webauthn@example.com"}
	wCreds := newWebAuthnUser(user, loaded).WebAuthnCredentials()
	if len(wCreds) != 1 {
		t.Fatalf("Expected 1 rehydrated credential, got %d", len(wCreds))
	}

	flags := wCreds[0].Flags
	if !flags.BackupEligible {
		t.Error("Expected rehydrated BackupEligible=true; got false (synced passkey login would fail)")
	}
	if !flags.BackupState {
		t.Error("Expected rehydrated BackupState=true; got false")
	}
	if !flags.UserPresent {
		t.Error("Expected rehydrated UserPresent=true; got false")
	}
	if !flags.UserVerified {
		t.Error("Expected rehydrated UserVerified=true; got false")
	}
}

func TestWebAuthnCredentialPersistsFlagsRoundTrip(t *testing.T) {
	svc, userID := setupWebAuthnTest(t)

	cred := models.WebAuthnCredential{
		UserID:         userID,
		CredentialID:   []byte("roundtrip"),
		PublicKey:      []byte("test-public-key"),
		Name:           "Round Trip",
		UserPresent:    true,
		UserVerified:   false,
		BackupEligible: true,
		BackupState:    false,
	}
	if err := svc.db.Create(&cred).Error; err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}

	var loaded models.WebAuthnCredential
	if err := svc.db.First(&loaded, "credential_id = ?", []byte("roundtrip")).Error; err != nil {
		t.Fatalf("Failed to reload credential: %v", err)
	}

	if !loaded.UserPresent || loaded.UserVerified || !loaded.BackupEligible || loaded.BackupState {
		t.Errorf("Flags not persisted faithfully: UP=%v UV=%v BE=%v BS=%v",
			loaded.UserPresent, loaded.UserVerified, loaded.BackupEligible, loaded.BackupState)
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
