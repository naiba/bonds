package models

import "gorm.io/gorm"

// importantDateTypeBackfillKeys maps the original English labels emitted by
// seedContactImportantDateTypes (before LabelTranslationKey landed) back to
// their i18n keys. Used to populate the column for rows created by older
// versions of the seeder so SyncAllTranslations can pick them up.
//
// Stays small and explicit on purpose — adding speculative variants risks
// rewriting custom rows a user happened to label with the same English text.
// If a future seed string is renamed, add the new mapping here AND keep the
// old one so legacy vaults still get covered.
var importantDateTypeBackfillKeys = map[string]string{
	"Birthdate":     "seed.important_date_types.birthdate",
	"Deceased date": "seed.important_date_types.deceased_date",
	"Anniversary":   "seed.important_date_types.anniversary",
	"Wedding":       "seed.important_date_types.wedding",
	"One-time date": "seed.important_date_types.one_time",
}

// BackfillContactImportantDateTranslationKeys assigns LabelTranslationKey on
// rows that pre-date the column. Rows with a non-null key are left untouched;
// rows whose label matches a known English seed default get the corresponding
// key so SyncAllTranslations can re-translate them on the next locale change.
//
// Idempotent: runs on every server boot. Custom user-created types (label not
// in the seed map) are intentionally skipped — we cannot guess their key.
func BackfillContactImportantDateTranslationKeys(db *gorm.DB) error {
	for label, key := range importantDateTypeBackfillKeys {
		k := key
		if err := db.Model(&ContactImportantDateType{}).
			Where("label = ? AND (label_translation_key IS NULL OR label_translation_key = '')", label).
			Update("label_translation_key", &k).Error; err != nil {
			return err
		}
	}
	return nil
}
