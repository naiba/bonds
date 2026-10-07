package database

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Labels are user text, not stable date types. Preserve the birthday previously
// exported by the label fallback once, then rely solely on the explicit type.
// Running this backfill on every startup would promote an ordinary/merged date
// after the primary birthday was removed. The journal stores only this key.
func migrateLegacyBirthdateTypes(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.DataMigration{Name: "explicit_birthdate_types"})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		var candidates []struct {
			ID        uint
			ContactID string
			TypeID    uint
		}
		if err := tx.Table("contact_important_dates AS dates").
			Select("dates.id, dates.contact_id, birthdate_types.id AS type_id").
			Joins("JOIN contacts ON contacts.id = dates.contact_id").
			Joins("JOIN contact_important_date_types AS birthdate_types ON birthdate_types.vault_id = contacts.vault_id AND birthdate_types.internal_type = ?", "birthdate").
			Joins("LEFT JOIN contact_important_date_types AS current_type ON current_type.id = dates.contact_important_date_type_id").
			Where("dates.deleted_at IS NULL AND (current_type.internal_type IS NULL) AND LOWER(dates.label) IN ?", []string{"birthday", "birthdate"}).
			Where(`NOT EXISTS (SELECT 1 FROM contact_important_dates AS primary_dates JOIN contact_important_date_types AS primary_type ON primary_type.id = primary_dates.contact_important_date_type_id WHERE primary_dates.contact_id = dates.contact_id AND primary_dates.deleted_at IS NULL AND primary_type.internal_type = ?)`, "birthdate").
			Order("dates.id ASC, birthdate_types.id ASC").Scan(&candidates).Error; err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, date := range candidates {
			if seen[date.ContactID] {
				continue
			}
			seen[date.ContactID] = true
			// Preserve label, values, reminders, timestamps and row identity; change only
			// the existing classification reference, never copy the user's date record.
			if err := tx.Model(&models.ContactImportantDate{}).Where("id = ?", date.ID).UpdateColumn("contact_important_date_type_id", date.TypeID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
