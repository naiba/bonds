package services

import (
	"testing"

	"github.com/markbates/goth"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/naiba/bonds/pkg/secret"
)

func TestOAuthProvider_ClientSecretEncryptedAtRest(t *testing.T) {
	defer goth.ClearProviders()
	db := testutil.SetupTestDB(t)
	svc := NewOAuthProviderServiceWithCipher(db, "boot-key")

	if _, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type:         "github",
		Name:         "gh",
		ClientID:     "id",
		ClientSecret: "raw-secret",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var row models.OAuthProvider
	if err := db.Where("name = ?", "gh").First(&row).Error; err != nil {
		t.Fatalf("read row: %v", err)
	}
	if !secret.IsCiphertext(row.ClientSecret) {
		t.Fatalf("client_secret must be encrypted, got %q", row.ClientSecret)
	}
	if row.ClientSecret == "raw-secret" {
		t.Fatal("plaintext leaked into client_secret column")
	}
}

func TestOAuthProvider_NoEncryptionWhenKeyDisabled(t *testing.T) {
	defer goth.ClearProviders()
	db := testutil.SetupTestDB(t)
	svc := NewOAuthProviderServiceWithCipher(db, "")

	if _, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type:         "github",
		Name:         "gh",
		ClientID:     "id",
		ClientSecret: "raw-secret",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var row models.OAuthProvider
	if err := db.Where("name = ?", "gh").First(&row).Error; err != nil {
		t.Fatalf("read row: %v", err)
	}
	if row.ClientSecret != "raw-secret" {
		t.Fatalf("expected plaintext when key disabled, got %q", row.ClientSecret)
	}
}

func TestOAuthProvider_UpdateReencryptsSecret(t *testing.T) {
	defer goth.ClearProviders()
	db := testutil.SetupTestDB(t)
	svc := NewOAuthProviderServiceWithCipher(db, "boot-key")

	created, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "gh", ClientID: "id", ClientSecret: "old",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newSecret := "new"
	if _, err := svc.Update(created.ID, dto.UpdateOAuthProviderRequest{
		ClientSecret: &newSecret,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var row models.OAuthProvider
	_ = db.First(&row, created.ID).Error
	if !secret.IsCiphertext(row.ClientSecret) {
		t.Fatalf("updated secret must remain encrypted, got %q", row.ClientSecret)
	}

	pt, err := svc.decryptSecret(row.ClientSecret)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if pt != "new" {
		t.Fatalf("Update did not re-encrypt with new value: got %q", pt)
	}
}

func TestOAuthProvider_MigratePlaintextSecretsIdempotent(t *testing.T) {
	defer goth.ClearProviders()
	db := testutil.SetupTestDB(t)

	plain := NewOAuthProviderServiceWithCipher(db, "")
	if _, err := plain.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "gh", ClientID: "id", ClientSecret: "legacy",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	enc := NewOAuthProviderServiceWithCipher(db, "boot-key")
	migrated, err := enc.MigratePlaintextSecrets()
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if migrated != 1 {
		t.Errorf("expected 1 migrated, got %d", migrated)
	}

	var row models.OAuthProvider
	_ = db.Where("name = ?", "gh").First(&row).Error
	if !secret.IsCiphertext(row.ClientSecret) {
		t.Fatalf("expected ciphertext after migration, got %q", row.ClientSecret)
	}

	migrated2, err := enc.MigratePlaintextSecrets()
	if err != nil {
		t.Fatalf("Migrate#2: %v", err)
	}
	if migrated2 != 0 {
		t.Errorf("second run should be no-op, got %d", migrated2)
	}

	pt, err := enc.decryptSecret(row.ClientSecret)
	if err != nil {
		t.Fatalf("decrypt after migration: %v", err)
	}
	if pt != "legacy" {
		t.Fatalf("decryption mismatch: got %q", pt)
	}
}

func TestOAuthProvider_ReloadDecryptsBeforeUsingGoth(t *testing.T) {
	defer goth.ClearProviders()
	db := testutil.SetupTestDB(t)
	settings := NewSystemSettingServiceWithCipher(db, "boot-key")
	_ = settings.Set("app.url", "http://localhost:8080")

	svc := NewOAuthProviderServiceWithCipher(db, "boot-key")
	svc.SetSystemSettings(settings)

	if _, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "gh", ClientID: "id", ClientSecret: "raw",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	svc.ReloadProviders()

	if _, err := goth.GetProvider("github"); err != nil {
		t.Fatalf("github provider must be registered after reload: %v", err)
	}
}
