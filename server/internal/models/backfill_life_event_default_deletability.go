package models

import "gorm.io/gorm"

var seededLifeEventCategoryTranslationKeys = map[string]struct{}{
	"seed.life_event_categories.transportation": {},
	"seed.life_event_categories.social":         {},
	"seed.life_event_categories.sport":          {},
	"seed.life_event_categories.work":           {},
}

var seededLifeEventTypeTranslationKeys = map[string]struct{}{
	"seed.life_event_types.rode_a_bike":      {},
	"seed.life_event_types.drove":            {},
	"seed.life_event_types.walked":           {},
	"seed.life_event_types.took_the_bus":     {},
	"seed.life_event_types.took_the_metro":   {},
	"seed.life_event_types.ate":              {},
	"seed.life_event_types.drank":            {},
	"seed.life_event_types.went_to_a_bar":    {},
	"seed.life_event_types.watched_a_movie":  {},
	"seed.life_event_types.watched_tv":       {},
	"seed.life_event_types.watched_a_tv_show": {},
	"seed.life_event_types.ran":              {},
	"seed.life_event_types.played_soccer":    {},
	"seed.life_event_types.played_basketball": {},
	"seed.life_event_types.played_golf":      {},
	"seed.life_event_types.played_tennis":    {},
	"seed.life_event_types.took_a_new_job":   {},
	"seed.life_event_types.quit_job":         {},
	"seed.life_event_types.got_fired":        {},
	"seed.life_event_types.had_a_promotion":  {},
}

// BackfillLifeEventDefaultDeletability repairs vaults seeded before default life
// event categories/types were marked deletable. We scope the update to known seed
// translation keys so user-created rows keep whatever delete policy they already
// had. Idempotent: runs on every boot.
func BackfillLifeEventDefaultDeletability(db *gorm.DB) error {
	categoryKeys := make([]string, 0, len(seededLifeEventCategoryTranslationKeys))
	for key := range seededLifeEventCategoryTranslationKeys {
		categoryKeys = append(categoryKeys, key)
	}
	typeKeys := make([]string, 0, len(seededLifeEventTypeTranslationKeys))
	for key := range seededLifeEventTypeTranslationKeys {
		typeKeys = append(typeKeys, key)
	}

	if err := db.Model(&LifeEventCategory{}).
		Where("label_translation_key IN ? AND can_be_deleted = ?", categoryKeys, false).
		Update("can_be_deleted", true).Error; err != nil {
		return err
	}

	if err := db.Model(&LifeEventType{}).
		Where("label_translation_key IN ? AND can_be_deleted = ?", typeKeys, false).
		Update("can_be_deleted", true).Error; err != nil {
		return err
	}

	return nil
}
