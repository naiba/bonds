package models

import "gorm.io/gorm"

// BackfillLifeEventTaskPostCalendarTypes stamps `calendar_type='gregorian'` on
// LifeEvent / ContactTask / Post rows that pre-date the column. GORM's column
// default only fires for new inserts; legacy rows inserted before the
// AutoMigrate ALTER end up with an empty string, and applyCalendarFields
// treats empty as "fall through to gregorian" which is the right behaviour —
// but unifying the on-disk value keeps reminders, exports and admin queries
// from having to special-case the empty string.
//
// Idempotent: rows already carrying any non-empty value are skipped, including
// freshly-inserted rows defaulted by GORM. Safe to invoke on every boot.
func BackfillLifeEventTaskPostCalendarTypes(db *gorm.DB) error {
	updates := []struct {
		table string
		model any
	}{
		{"life_events", &LifeEvent{}},
		{"contact_tasks", &ContactTask{}},
		{"posts", &Post{}},
	}
	for _, u := range updates {
		if !db.Migrator().HasTable(u.table) {
			continue
		}
		if err := db.Model(u.model).
			Where("calendar_type IS NULL OR calendar_type = ''").
			Update("calendar_type", "gregorian").Error; err != nil {
			return err
		}
	}
	return nil
}
