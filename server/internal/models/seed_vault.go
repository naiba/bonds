package models

import "gorm.io/gorm"

func SeedVaultDefaults(tx *gorm.DB, vaultID string) error {
	seeders := []func(*gorm.DB, string) error{
		seedContactImportantDateTypes,
		seedMoodTrackingParameters,
		seedLifeEventCategoriesAndTypes,
		seedVaultQuickFactsTemplates,
	}
	for _, fn := range seeders {
		if err := fn(tx, vaultID); err != nil {
			return err
		}
	}
	return nil
}

func seedContactImportantDateTypes(tx *gorm.DB, vaultID string) error {
	items := []ContactImportantDateType{
		{VaultID: vaultID, Label: "Birthdate", InternalType: strPtr("birthdate")},
		{VaultID: vaultID, Label: "Deceased date", InternalType: strPtr("deceased_date")},
	}
	if err := tx.Create(&items).Error; err != nil {
		return err
	}
	ids := make([]uint, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return tx.Model(&ContactImportantDateType{}).
		Where("id IN ?", ids).
		Update("can_be_deleted", false).Error
}

func seedMoodTrackingParameters(tx *gorm.DB, vaultID string) error {
	items := []MoodTrackingParameter{
		{VaultID: vaultID, Label: strPtr("🥳 Awesome"), HexColor: "bg-lime-500", Position: intPtr(1)},
		{VaultID: vaultID, Label: strPtr("😀 Good"), HexColor: "bg-lime-300", Position: intPtr(2)},
		{VaultID: vaultID, Label: strPtr("😐 Meh"), HexColor: "bg-cyan-600", Position: intPtr(3)},
		{VaultID: vaultID, Label: strPtr("😔 Bad"), HexColor: "bg-orange-300", Position: intPtr(4)},
		{VaultID: vaultID, Label: strPtr("😩 Awful"), HexColor: "bg-red-700", Position: intPtr(5)},
	}
	return tx.Create(&items).Error
}

func seedLifeEventCategoriesAndTypes(tx *gorm.DB, vaultID string) error {
	type categoryDef struct {
		label    string
		position int
		types    []string
	}

	categories := []categoryDef{
		{"Transportation", 1, []string{"Rode a bike", "Drove", "Walked", "Took the bus", "Took the metro"}},
		{"Social", 2, []string{"Ate", "Drank", "Went to a bar", "Watched a movie", "Watched TV", "Watched a tv show"}},
		{"Sport", 3, []string{"Ran", "Played soccer", "Played basketball", "Played golf", "Played tennis"}},
		{"Work", 4, []string{"Took a new job", "Quit job", "Got fired", "Had a promotion"}},
	}

	for _, cat := range categories {
		pos := cat.position
		category := LifeEventCategory{
			VaultID:  vaultID,
			Label:    strPtr(cat.label),
			Position: &pos,
		}
		if err := tx.Create(&category).Error; err != nil {
			return err
		}
		for i, typeName := range cat.types {
			typePos := i + 1
			lifeEventType := LifeEventType{
				LifeEventCategoryID: category.ID,
				Label:               strPtr(typeName),
				Position:            &typePos,
			}
			if err := tx.Create(&lifeEventType).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedVaultQuickFactsTemplates(tx *gorm.DB, vaultID string) error {
	items := []VaultQuickFactsTemplate{
		{VaultID: vaultID, Label: strPtr("Hobbies"), Position: 1},
		{VaultID: vaultID, Label: strPtr("Food preferences"), Position: 2},
	}
	return tx.Create(&items).Error
}
