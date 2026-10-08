package services

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func mergeContactAssociations(tx *gorm.DB, sourceID, targetID string) error {
	if err := mergeContactImportantDates(tx, sourceID, targetID); err != nil {
		return err
	}
	if err := mergeContactUserHistory(tx, sourceID, targetID); err != nil {
		return err
	}
	var sourceGroups []models.ContactGroup
	if err := tx.Where("contact_id = ?", sourceID).Find(&sourceGroups).Error; err != nil {
		return err
	}
	for _, group := range sourceGroups {
		if group.GroupTypeRoleID != nil {
			if err := tx.Model(&models.ContactGroup{}).Where("contact_id = ? AND group_id = ? AND group_type_role_id IS NULL", targetID, group.GroupID).Update("group_type_role_id", group.GroupTypeRoleID).Error; err != nil {
				return err
			}
		}
	}
	// These are sets. Remove only identical memberships before moving their rows,
	// including on SQLite where an UPDATE would otherwise violate unique indexes.
	for _, pivot := range []struct{ table, key string }{
		{"task_contacts", "contact_task_id"}, {"activity_participants", "activity_id"},
		{"contact_post", "post_id"}, {"contact_group", "group_id"}, {"contact_label", "label_id"},
	} {
		existing := tx.Table(pivot.table).Select(pivot.key).Where("contact_id = ?", targetID)
		if err := tx.Table(pivot.table).Where("contact_id = ? AND "+pivot.key+" IN (?)", sourceID, existing).Delete(nil).Error; err != nil {
			return err
		}
		if err := tx.Table(pivot.table).Where("contact_id = ?", sourceID).Update("contact_id", targetID).Error; err != nil {
			return err
		}
	}
	// Keep individual records (even similar ones): notes, typed facts, address
	// history, jobs, events and scheduled reminders carry independent information.
	for _, model := range []any{
		&models.ContactInformation{}, &models.Note{}, &models.ContactReminder{}, &models.Call{},
		&models.Pet{}, &models.Goal{}, &models.Gift{}, &models.QuickFact{}, &models.ContactCompany{},
		&models.ContactAddress{}, &models.ContactLifeMetric{}, &models.ContactFeedItem{},
	} {
		if err := tx.Model(model).Where("contact_id = ?", sourceID).Update("contact_id", targetID).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&models.File{}).Where("ufileable_id = ?", sourceID).Update("ufileable_id", targetID).Error; err != nil {
		return err
	}
	for _, model := range []any{&models.ContactLoan{}, &models.ContactGift{}} {
		for _, column := range []string{"loaner_id", "loanee_id"} {
			if err := tx.Model(model).Where(column+" = ?", sourceID).Update(column, targetID).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func mergeContactImportantDates(tx *gorm.DB, sourceID, targetID string) error {
	var dates []models.ContactImportantDate
	if err := tx.Preload("ContactImportantDateType").Where("contact_id IN ?", []string{targetID, sourceID}).Order("id ASC").Find(&dates).Error; err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, date := range dates {
		if date.ContactID == targetID && date.ContactImportantDateType != nil && date.ContactImportantDateType.InternalType != nil {
			seen[*date.ContactImportantDateType.InternalType] = true
		}
	}
	for _, date := range dates {
		if date.ContactID != sourceID {
			continue
		}
		if date.ContactImportantDateType != nil && date.ContactImportantDateType.InternalType != nil {
			internalType := *date.ContactImportantDateType.InternalType
			if _, singleton := singletonImportantDateInternalTypes[internalType]; singleton {
				if seen[internalType] {
					// Keep conflicting birthdays/death dates as ordinary dates with the same
					// label and ID, so reminders and calendar references survive the merge.
					// Use an empty model: saving the preloaded type association would restore the cleared foreign key.
					if err := tx.Model(&models.ContactImportantDate{}).Where("id = ?", date.ID).Update("contact_important_date_type_id", nil).Error; err != nil {
						return err
					}
				}
				seen[internalType] = true
			}
		}
	}
	return tx.Unscoped().Model(&models.ContactImportantDate{}).Where("contact_id = ?", sourceID).Update("contact_id", targetID).Error
}

func mergeContactUserHistory(tx *gorm.DB, sourceID, targetID string) error {
	var histories []models.ContactVaultUser
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("contact_id IN ?", []string{targetID, sourceID}).Order("id ASC").Find(&histories).Error; err != nil {
		return err
	}
	byUser := map[string]models.ContactVaultUser{}
	for _, history := range histories {
		previous, exists := byUser[history.UserID]
		if !exists {
			history.ContactID = targetID
			byUser[history.UserID] = history
			continue
		}
		previous.NumberOfViews += history.NumberOfViews
		previous.IsFavorite = previous.IsFavorite || history.IsFavorite
		if history.LastConsultedAt != nil && (previous.LastConsultedAt == nil || history.LastConsultedAt.After(*previous.LastConsultedAt)) {
			previous.LastConsultedAt = history.LastConsultedAt
		}
		byUser[history.UserID] = previous
		if err := tx.Delete(&history).Error; err != nil {
			return err
		}
	}
	for _, history := range byUser {
		if err := tx.Save(&history).Error; err != nil {
			return err
		}
	}
	return nil
}
