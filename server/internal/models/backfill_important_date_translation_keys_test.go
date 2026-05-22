package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupContactImportantDateTypeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ContactImportantDateType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestBackfillContactImportantDateTranslationKeys covers vault databases that
// pre-date the LabelTranslationKey column: rows live there with the English
// label baked in and a NULL key. Without this backfill, SyncAllTranslations
// would skip them (its WHERE filter rejects null/empty keys), so the user
// would never see those types re-translate even after the fix shipped.
func TestBackfillContactImportantDateTranslationKeys(t *testing.T) {
	db := setupContactImportantDateTypeDB(t)

	englishToKey := map[string]string{
		"Birthdate":     "seed.important_date_types.birthdate",
		"Deceased date": "seed.important_date_types.deceased_date",
		"Anniversary":   "seed.important_date_types.anniversary",
		"Wedding":       "seed.important_date_types.wedding",
		"One-time date": "seed.important_date_types.one_time",
	}

	rows := make([]ContactImportantDateType, 0, len(englishToKey)+1)
	for label := range englishToKey {
		rows = append(rows, ContactImportantDateType{
			VaultID: "vault-legacy",
			Label:   label,
		})
	}
	rows = append(rows, ContactImportantDateType{
		VaultID: "vault-legacy",
		Label:   "Some Custom Type The User Made",
	})
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed legacy rows: %v", err)
	}

	if err := BackfillContactImportantDateTranslationKeys(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var all []ContactImportantDateType
	if err := db.Where("vault_id = ?", "vault-legacy").Find(&all).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, ty := range all {
		if want, isSeedDefault := englishToKey[ty.Label]; isSeedDefault {
			if ty.LabelTranslationKey == nil || *ty.LabelTranslationKey != want {
				t.Errorf("%q: want LabelTranslationKey=%q, got %v", ty.Label, want, ty.LabelTranslationKey)
			}
			continue
		}
		// Custom user rows must be left alone — they have no equivalent
		// translation key, so a wrong guess would freeze the wrong label
		// next time Sync runs.
		if ty.LabelTranslationKey != nil {
			t.Errorf("custom row %q should not have been assigned a key, got %v", ty.Label, ty.LabelTranslationKey)
		}
	}

	// Second run must be a no-op — the backfill is invoked on every boot.
	if err := BackfillContactImportantDateTranslationKeys(db); err != nil {
		t.Fatalf("backfill (second run): %v", err)
	}
}

// TestBackfillSkipsRowsAlreadyHavingKey ensures rows from newly-seeded vaults
// (which have the key set) are not touched.
func TestBackfillSkipsRowsAlreadyHavingKey(t *testing.T) {
	db := setupContactImportantDateTypeDB(t)
	existingKey := "seed.important_date_types.birthdate"
	row := ContactImportantDateType{
		VaultID:             "vault-new",
		Label:               "生日",
		LabelTranslationKey: &existingKey,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := BackfillContactImportantDateTranslationKeys(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var reloaded ContactImportantDateType
	if err := db.First(&reloaded, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Label != "生日" {
		t.Errorf("label mutated: got %q want %q", reloaded.Label, "生日")
	}
	if reloaded.LabelTranslationKey == nil || *reloaded.LabelTranslationKey != existingKey {
		t.Errorf("key mutated: got %v want %q", reloaded.LabelTranslationKey, existingKey)
	}
}
