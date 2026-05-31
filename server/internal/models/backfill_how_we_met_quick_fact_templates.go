package models

import (
	"strings"

	"github.com/naiba/bonds/internal/i18n"
	"gorm.io/gorm"
)

const howWeMetQuickFactTranslationKey = "seed.quick_facts.how_we_met"

var howWeMetQuickFactLabels = []string{
	strings.ToLower(i18n.T("en", howWeMetQuickFactTranslationKey)),
	strings.ToLower(i18n.T("zh", howWeMetQuickFactTranslationKey)),
	strings.ToLower(i18n.T("es", howWeMetQuickFactTranslationKey)),
	strings.ToLower(i18n.T("fr", howWeMetQuickFactTranslationKey)),
}

func BackfillHowWeMetQuickFactTemplates(db *gorm.DB) error {
	var vaultIDs []string
	if err := db.Model(&Vault{}).Pluck("id", &vaultIDs).Error; err != nil {
		return err
	}
	for _, vaultID := range vaultIDs {
		var existingCount int64
		if err := db.Model(&VaultQuickFactsTemplate{}).
			Where(
				"vault_id = ? AND (label_translation_key = ? OR LOWER(COALESCE(label, '')) IN ?)",
				vaultID,
				howWeMetQuickFactTranslationKey,
				howWeMetQuickFactLabels,
			).
			Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}

		var maxPosition int
		if err := db.Model(&VaultQuickFactsTemplate{}).
			Where("vault_id = ?", vaultID).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPosition).Error; err != nil {
			return err
		}
		if err := db.Create(&VaultQuickFactsTemplate{
			VaultID:             vaultID,
			Label:               strPtr(i18n.T("en", howWeMetQuickFactTranslationKey)),
			LabelTranslationKey: strPtr(howWeMetQuickFactTranslationKey),
			Position:            maxPosition + 1,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
