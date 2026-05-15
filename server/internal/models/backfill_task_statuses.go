package models

import "gorm.io/gorm"

// BackfillTaskStatuses ensures every account has the seeded TaskStatus rows.
// Runs at server startup so accounts that pre-date the TaskStatus model
// (i.e. were created before this feature shipped) end up with the same
// default columns the kanban expects. Idempotent — accounts that already
// have rows are skipped.
//
// Locale is "en" because the seed translation keys are committed and the
// account's preferred locale isn't readily available outside the request
// path; users can re-translate via the existing personalize sync flow.
func BackfillTaskStatuses(db *gorm.DB) error {
	var accountIDs []string
	if err := db.Table("accounts").Pluck("id", &accountIDs).Error; err != nil {
		return err
	}
	for _, accountID := range accountIDs {
		var count int64
		if err := db.Model(&TaskStatus{}).Where("account_id = ?", accountID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			return seedTaskStatuses(tx, accountID, "en")
		}); err != nil {
			return err
		}
	}
	return nil
}
