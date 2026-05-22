package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// setupVaultWithLocale registers an account, creates a vault, runs the
// vault-default seeders for the given locale, and returns (db, vaultID,
// accountID). The locale flows through the seed translation lookups so we
// can assert label persistence reflects the requested language.
func setupVaultWithLocale(t *testing.T, locale string) (*gorm.DB, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Sync",
		LastName:  "Tester",
		Email:     "sync-tester-" + locale + "@example.com",
		Password:  "password123",
	}, locale)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	var account models.Account
	if err := db.First(&account, "id = ?", resp.User.AccountID).Error; err != nil {
		t.Fatalf("load account: %v", err)
	}

	vault := models.Vault{
		Name:      "Test Vault",
		AccountID: resp.User.AccountID,
	}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("create vault: %v", err)
	}
	if err := models.SeedVaultDefaults(db, vault.ID, locale); err != nil {
		t.Fatalf("seed vault defaults: %v", err)
	}
	return db, vault.ID, resp.User.AccountID
}

// TestSyncContactImportantDateTypesAcrossLocales guards against the silent
// gap where seeded ContactImportantDateType rows would freeze in whichever
// language the vault was created in. The user reported the symptom: switch
// locale to Chinese, hit "Sync translations", but birthday/anniversary type
// labels stay English because personalize.go's vaultSyncEntities never
// included this table and the model carried no translation key.
func TestSyncContactImportantDateTypesAcrossLocales(t *testing.T) {
	db, vaultID, accountID := setupVaultWithLocale(t, "en")

	var typesAfterSeed []models.ContactImportantDateType
	if err := db.Where("vault_id = ?", vaultID).Find(&typesAfterSeed).Error; err != nil {
		t.Fatalf("query seeded types: %v", err)
	}
	if len(typesAfterSeed) == 0 {
		t.Fatal("seed did not produce any ContactImportantDateType rows")
	}
	for _, ty := range typesAfterSeed {
		if ty.LabelTranslationKey == nil || *ty.LabelTranslationKey == "" {
			t.Errorf("seeded type %q has empty LabelTranslationKey; SyncAllTranslations cannot retranslate it", ty.Label)
		}
	}

	svc := NewPersonalizeService(db)
	if err := svc.SyncAllTranslations(accountID, "zh"); err != nil {
		t.Fatalf("SyncAllTranslations(zh): %v", err)
	}

	var afterSync []models.ContactImportantDateType
	if err := db.Where("vault_id = ?", vaultID).Find(&afterSync).Error; err != nil {
		t.Fatalf("query after sync: %v", err)
	}

	englishBaselines := map[string]bool{
		"Birthdate":     true,
		"Deceased date": true,
		"Anniversary":   true,
		"Wedding":       true,
		"One-time":      true,
	}
	for _, ty := range afterSync {
		if englishBaselines[ty.Label] {
			t.Errorf("after sync to zh, type still has English label %q (translation_key=%v)", ty.Label, ty.LabelTranslationKey)
		}
	}
}

// TestSyncContactImportantDateTypesIgnoresCustomRows ensures user-created
// custom types (no LabelTranslationKey) survive a sync untouched — only the
// seeded defaults should flip languages.
func TestSyncContactImportantDateTypesIgnoresCustomRows(t *testing.T) {
	db, vaultID, accountID := setupVaultWithLocale(t, "en")

	custom := models.ContactImportantDateType{
		VaultID: vaultID,
		Label:   "My Custom Date Type",
	}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatalf("create custom type: %v", err)
	}

	svc := NewPersonalizeService(db)
	if err := svc.SyncAllTranslations(accountID, "zh"); err != nil {
		t.Fatalf("SyncAllTranslations(zh): %v", err)
	}

	var reloaded models.ContactImportantDateType
	if err := db.First(&reloaded, custom.ID).Error; err != nil {
		t.Fatalf("reload custom: %v", err)
	}
	if reloaded.Label != "My Custom Date Type" {
		t.Errorf("custom type label was mutated: got %q want %q", reloaded.Label, "My Custom Date Type")
	}
}
