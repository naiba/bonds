package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/naiba/bonds/pkg/secret"
)

func TestSystemSetting_PlaintextWhenNoKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "")

	if err := svc.Set("smtp.password", "p@ss"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var row models.SystemSetting
	if err := db.Where("key = ?", "smtp.password").First(&row).Error; err != nil {
		t.Fatalf("read raw row: %v", err)
	}
	if row.Value != "p@ss" {
		t.Fatalf("expected raw plaintext when key disabled, got %q", row.Value)
	}

	v, err := svc.Get("smtp.password")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if v != "p@ss" {
		t.Fatalf("Get returned wrong value: %q", v)
	}
}

func TestSystemSetting_EncryptedAtRestForSecretKeys(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "boot-key")

	if err := svc.Set("smtp.password", "p@ss"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var row models.SystemSetting
	if err := db.Where("key = ?", "smtp.password").First(&row).Error; err != nil {
		t.Fatalf("read raw row: %v", err)
	}
	if !secret.IsCiphertext(row.Value) {
		t.Fatalf("expected ciphertext on disk, got %q", row.Value)
	}
	if row.Value == "p@ss" {
		t.Fatal("plaintext leaked into row.value")
	}

	v, err := svc.Get("smtp.password")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if v != "p@ss" {
		t.Fatalf("Get must return decrypted plaintext, got %q", v)
	}
}

func TestSystemSetting_NonSecretKeysNotEncrypted(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "boot-key")

	if err := svc.Set("app.name", "Bonds"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var row models.SystemSetting
	if err := db.Where("key = ?", "app.name").First(&row).Error; err != nil {
		t.Fatalf("read row: %v", err)
	}
	if secret.IsCiphertext(row.Value) {
		t.Fatalf("non-secret key should not be encrypted, got %q", row.Value)
	}
	if row.Value != "Bonds" {
		t.Fatalf("expected plaintext app.name, got %q", row.Value)
	}
}

func TestSystemSetting_GetAllRedactsSecrets(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "boot-key")

	_ = svc.Set("smtp.password", "p@ss")
	_ = svc.Set("smtp.host", "mail.example.com")
	_ = svc.Set("geocoding.api_key", "abc")

	items, err := svc.GetAllRedacted()
	if err != nil {
		t.Fatalf("GetAllRedacted: %v", err)
	}

	got := map[string]string{}
	for _, it := range items {
		got[it.Key] = it.Value
	}
	if got["smtp.password"] != RedactedSecretValue {
		t.Errorf("smtp.password should be redacted, got %q", got["smtp.password"])
	}
	if got["geocoding.api_key"] != RedactedSecretValue {
		t.Errorf("geocoding.api_key should be redacted, got %q", got["geocoding.api_key"])
	}
	if got["smtp.host"] != "mail.example.com" {
		t.Errorf("non-secret values must be visible, got %q", got["smtp.host"])
	}
}

func TestSystemSetting_GetAllRedactsEmptyAsEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "boot-key")
	_ = svc.Set("smtp.password", "")

	items, err := svc.GetAllRedacted()
	if err != nil {
		t.Fatalf("GetAllRedacted: %v", err)
	}
	for _, it := range items {
		if it.Key == "smtp.password" && it.Value != "" {
			t.Errorf("empty secret should serialise as empty, got %q", it.Value)
		}
	}
}

func TestSystemSetting_BulkSetRedactedSentinelKeepsExistingSecret(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "boot-key")

	if err := svc.Set("smtp.password", "original"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err := svc.BulkSet([]dto.SystemSettingItem{
		{Key: "smtp.password", Value: RedactedSecretValue},
		{Key: "smtp.host", Value: "mail2.example.com"},
	})
	if err != nil {
		t.Fatalf("BulkSet: %v", err)
	}

	v, err := svc.Get("smtp.password")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if v != "original" {
		t.Fatalf("redacted sentinel must keep existing secret, got %q", v)
	}

	if got := svc.GetWithDefault("smtp.host", ""); got != "mail2.example.com" {
		t.Errorf("non-secret update must apply, got %q", got)
	}
}

func TestSystemSetting_BulkSetUpdatesSecretWhenRealValueProvided(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "boot-key")

	_ = svc.Set("smtp.password", "old")
	err := svc.BulkSet([]dto.SystemSettingItem{
		{Key: "smtp.password", Value: "new"},
	})
	if err != nil {
		t.Fatalf("BulkSet: %v", err)
	}

	v, err := svc.Get("smtp.password")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if v != "new" {
		t.Fatalf("expected secret to be updated, got %q", v)
	}

	var row models.SystemSetting
	if err := db.Where("key = ?", "smtp.password").First(&row).Error; err != nil {
		t.Fatalf("row: %v", err)
	}
	if !secret.IsCiphertext(row.Value) {
		t.Fatalf("updated secret must remain encrypted on disk, got %q", row.Value)
	}
}

func TestSystemSetting_MigratePlaintextSecretsIdempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	plain := NewSystemSettingServiceWithCipher(db, "")
	_ = plain.Set("smtp.password", "legacy")
	_ = plain.Set("smtp.host", "mail.example.com")

	enc := NewSystemSettingServiceWithCipher(db, "boot-key")

	migrated, err := enc.MigratePlaintextSecrets()
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	if migrated != 1 {
		t.Errorf("expected 1 migrated row (smtp.password), got %d", migrated)
	}

	var row models.SystemSetting
	_ = db.Where("key = ?", "smtp.password").First(&row)
	if !secret.IsCiphertext(row.Value) {
		t.Fatalf("smtp.password should be encrypted after migration, got %q", row.Value)
	}

	migrated2, err := enc.MigratePlaintextSecrets()
	if err != nil {
		t.Fatalf("Migrate#2 failed: %v", err)
	}
	if migrated2 != 0 {
		t.Errorf("second migration should be no-op, got %d", migrated2)
	}

	v, err := enc.Get("smtp.password")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if v != "legacy" {
		t.Fatalf("decryption after migration mismatch: got %q", v)
	}
}

func TestSystemSetting_MigrateNoOpWhenDisabled(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "")
	_ = svc.Set("smtp.password", "legacy")

	migrated, err := svc.MigratePlaintextSecrets()
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if migrated != 0 {
		t.Errorf("disabled cipher must not migrate, got %d", migrated)
	}
}

func TestSystemSetting_LegacyPlaintextReadableAfterEnablingKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	plain := NewSystemSettingServiceWithCipher(db, "")
	_ = plain.Set("smtp.password", "legacy")

	enc := NewSystemSettingServiceWithCipher(db, "boot-key")
	v, err := enc.Get("smtp.password")
	if err != nil {
		t.Fatalf("Get failed on legacy plaintext: %v", err)
	}
	if v != "legacy" {
		t.Fatalf("legacy plaintext must remain readable, got %q", v)
	}
}

func TestSystemSetting_DecryptFailsWithWrongKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingServiceWithCipher(db, "key-A")
	_ = svc.Set("smtp.password", "secret")

	wrong := NewSystemSettingServiceWithCipher(db, "key-B")
	if _, err := wrong.Get("smtp.password"); err == nil {
		t.Fatal("expected decryption failure with wrong key")
	}
}

func TestIsSecretKey(t *testing.T) {
	for _, k := range []string{"smtp.password", "geocoding.api_key", "secret.foo"} {
		if !IsSecretKey(k) {
			t.Errorf("%q should be classified as secret", k)
		}
	}
	for _, k := range []string{"smtp.host", "app.name", "backup.cron"} {
		if IsSecretKey(k) {
			t.Errorf("%q must not be classified as secret", k)
		}
	}
}
