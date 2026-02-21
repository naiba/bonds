package models

import (
	"github.com/naiba/bonds/internal/i18n"
	"gorm.io/gorm"
)

func SeedVaultDefaults(tx *gorm.DB, vaultID, locale string) error {
	seeders := []func(*gorm.DB, string, string) error{
		seedContactImportantDateTypes,
		seedMoodTrackingParameters,
		seedLifeEventCategoriesAndTypes,
		seedVaultQuickFactsTemplates,
	}
	for _, fn := range seeders {
		if err := fn(tx, vaultID, locale); err != nil {
			return err
		}
	}
	return nil
}

func seedContactImportantDateTypes(tx *gorm.DB, vaultID, locale string) error {
	type dateDef struct {
		key          string
		internalType string
	}
	defs := []dateDef{
		{"seed.important_date_types.birthdate", "birthdate"},
		{"seed.important_date_types.deceased_date", "deceased_date"},
		{"seed.important_date_types.anniversary", ""},
		{"seed.important_date_types.wedding", ""},
		{"seed.important_date_types.one_time", ""},
	}
	items := make([]ContactImportantDateType, len(defs))
	for idx, d := range defs {
		items[idx] = ContactImportantDateType{
			VaultID:      vaultID,
			Label:        i18n.T(locale, d.key),
			InternalType: strPtr(d.internalType),
		}
	}
	if err := tx.Create(&items).Error; err != nil {
		return err
	}
	var undeletableIDs []uint
	for i, d := range defs {
		if d.internalType != "" {
			undeletableIDs = append(undeletableIDs, items[i].ID)
		}
	}
	if len(undeletableIDs) > 0 {
		return tx.Model(&ContactImportantDateType{}).
			Where("id IN ?", undeletableIDs).
			Update("can_be_deleted", false).Error
	}
	return nil
}

func seedMoodTrackingParameters(tx *gorm.DB, vaultID, locale string) error {
	type moodDef struct {
		key      string
		hexColor string
		position int
	}
	defs := []moodDef{
		{"seed.mood_tracking.awesome", "bg-lime-500", 1},
		{"seed.mood_tracking.good", "bg-lime-300", 2},
		{"seed.mood_tracking.meh", "bg-cyan-600", 3},
		{"seed.mood_tracking.bad", "bg-orange-300", 4},
		{"seed.mood_tracking.awful", "bg-red-700", 5},
	}
	items := make([]MoodTrackingParameter, len(defs))
	for idx, d := range defs {
		items[idx] = MoodTrackingParameter{
			VaultID:             vaultID,
			Label:               strPtr(i18n.T(locale, d.key)),
			LabelTranslationKey: strPtr(d.key),
			HexColor:            d.hexColor,
			Position:            intPtr(d.position),
		}
	}
	return tx.Create(&items).Error
}

func seedLifeEventCategoriesAndTypes(tx *gorm.DB, vaultID, locale string) error {
	type categoryDef struct {
		key      string
		position int
		types    []string
	}

	categories := []categoryDef{
		{"seed.life_event_categories.transportation", 1, []string{
			"seed.life_event_types.rode_a_bike",
			"seed.life_event_types.drove",
			"seed.life_event_types.walked",
			"seed.life_event_types.took_the_bus",
			"seed.life_event_types.took_the_metro",
		}},
		{"seed.life_event_categories.social", 2, []string{
			"seed.life_event_types.ate",
			"seed.life_event_types.drank",
			"seed.life_event_types.went_to_a_bar",
			"seed.life_event_types.watched_a_movie",
			"seed.life_event_types.watched_tv",
			"seed.life_event_types.watched_a_tv_show",
		}},
		{"seed.life_event_categories.sport", 3, []string{
			"seed.life_event_types.ran",
			"seed.life_event_types.played_soccer",
			"seed.life_event_types.played_basketball",
			"seed.life_event_types.played_golf",
			"seed.life_event_types.played_tennis",
		}},
		{"seed.life_event_categories.work", 4, []string{
			"seed.life_event_types.took_a_new_job",
			"seed.life_event_types.quit_job",
			"seed.life_event_types.got_fired",
			"seed.life_event_types.had_a_promotion",
		}},
	}

	for _, cat := range categories {
		pos := cat.position
		category := LifeEventCategory{
			VaultID:             vaultID,
			Label:               strPtr(i18n.T(locale, cat.key)),
			LabelTranslationKey: strPtr(cat.key),
			Position:            &pos,
		}
		if err := tx.Create(&category).Error; err != nil {
			return err
		}
		for idx, typeKey := range cat.types {
			typePos := idx + 1
			lifeEventType := LifeEventType{
				LifeEventCategoryID: category.ID,
				Label:               strPtr(i18n.T(locale, typeKey)),
				LabelTranslationKey: strPtr(typeKey),
				Position:            &typePos,
			}
			if err := tx.Create(&lifeEventType).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedVaultQuickFactsTemplates(tx *gorm.DB, vaultID, locale string) error {
	type qfDef struct {
		key      string
		position int
	}
	defs := []qfDef{
		{"seed.quick_facts.hobbies", 1},
		{"seed.quick_facts.food_preferences", 2},
	}
	items := make([]VaultQuickFactsTemplate, len(defs))
	for idx, d := range defs {
		items[idx] = VaultQuickFactsTemplate{
			VaultID:             vaultID,
			Label:               strPtr(i18n.T(locale, d.key)),
			LabelTranslationKey: strPtr(d.key),
			Position:            d.position,
		}
	}
	return tx.Create(&items).Error
}
