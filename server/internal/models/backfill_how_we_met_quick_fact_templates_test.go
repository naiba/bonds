package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupQuickFactTemplateBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Vault{}, &VaultQuickFactsTemplate{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBackfillHowWeMetQuickFactTemplatesAppendsWithoutReorderingExistingTemplates(t *testing.T) {
	db := setupQuickFactTemplateBackfillDB(t)
	vault := Vault{ID: "vault-old", AccountID: "account-1", Type: "personal", Name: "Old Vault"}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("create vault: %v", err)
	}
	if err := db.Create(&[]VaultQuickFactsTemplate{
		{VaultID: vault.ID, Label: strPtr("Hobbies"), LabelTranslationKey: strPtr("seed.quick_facts.hobbies"), Position: 1},
		{VaultID: vault.ID, Label: strPtr("Food preferences"), LabelTranslationKey: strPtr("seed.quick_facts.food_preferences"), Position: 2},
	}).Error; err != nil {
		t.Fatalf("create templates: %v", err)
	}

	if err := BackfillHowWeMetQuickFactTemplates(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var templates []VaultQuickFactsTemplate
	if err := db.Where("vault_id = ?", vault.ID).Order("position ASC").Find(&templates).Error; err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}
	if templates[0].Label == nil || *templates[0].Label != "Hobbies" || templates[0].Position != 1 {
		t.Fatalf("expected Hobbies to remain first, got label %v position %d", templates[0].Label, templates[0].Position)
	}
	if templates[1].Label == nil || *templates[1].Label != "Food preferences" || templates[1].Position != 2 {
		t.Fatalf("expected Food preferences to remain second, got label %v position %d", templates[1].Label, templates[1].Position)
	}
	inserted := templates[2]
	if inserted.Label == nil || *inserted.Label != "How we met" {
		t.Fatalf("expected inserted label How we met, got %v", inserted.Label)
	}
	if inserted.LabelTranslationKey == nil || *inserted.LabelTranslationKey != howWeMetQuickFactTranslationKey {
		t.Fatalf("expected inserted translation key %q, got %v", howWeMetQuickFactTranslationKey, inserted.LabelTranslationKey)
	}
	if inserted.Position != 3 {
		t.Fatalf("expected inserted position 3, got %d", inserted.Position)
	}

	if err := BackfillHowWeMetQuickFactTemplates(db); err != nil {
		t.Fatalf("backfill second run: %v", err)
	}
	var count int64
	if err := db.Model(&VaultQuickFactsTemplate{}).Where("vault_id = ?", vault.ID).Count(&count).Error; err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected idempotent count 3, got %d", count)
	}
}

func TestBackfillHowWeMetQuickFactTemplatesSkipsUserTemplate(t *testing.T) {
	db := setupQuickFactTemplateBackfillDB(t)
	vault := Vault{ID: "vault-custom", AccountID: "account-1", Type: "personal", Name: "Custom Vault"}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("create vault: %v", err)
	}
	if err := db.Create(&VaultQuickFactsTemplate{VaultID: vault.ID, Label: strPtr("How we met"), Position: 1}).Error; err != nil {
		t.Fatalf("create user template: %v", err)
	}

	if err := BackfillHowWeMetQuickFactTemplates(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var count int64
	if err := db.Model(&VaultQuickFactsTemplate{}).Where("vault_id = ?", vault.ID).Count(&count).Error; err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected existing user template to prevent duplicate, got %d templates", count)
	}
}

func TestBackfillHowWeMetQuickFactTemplatesSkipsLocalizedUserTemplate(t *testing.T) {
	db := setupQuickFactTemplateBackfillDB(t)
	vault := Vault{ID: "vault-localized", AccountID: "account-1", Type: "personal", Name: "Localized Vault"}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("create vault: %v", err)
	}
	if err := db.Create(&VaultQuickFactsTemplate{VaultID: vault.ID, Label: strPtr("如何认识"), Position: 1}).Error; err != nil {
		t.Fatalf("create localized user template: %v", err)
	}

	if err := BackfillHowWeMetQuickFactTemplates(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var count int64
	if err := db.Model(&VaultQuickFactsTemplate{}).Where("vault_id = ?", vault.ID).Count(&count).Error; err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected localized user template to prevent duplicate, got %d templates", count)
	}
}
