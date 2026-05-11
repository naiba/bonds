package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// taskAssignees loads the contact assignees for a set of task IDs in two
// queries (pivot + contacts), returning a map from task ID to a slice of
// TaskContactRef sorted by first name then last name for stable output.
func taskAssignees(db *gorm.DB, taskIDs []uint) (map[uint][]dto.TaskContactRef, error) {
	if len(taskIDs) == 0 {
		return map[uint][]dto.TaskContactRef{}, nil
	}

	var pivots []models.TaskContact
	if err := db.Where("contact_task_id IN ?", taskIDs).Find(&pivots).Error; err != nil {
		return nil, err
	}
	if len(pivots) == 0 {
		return map[uint][]dto.TaskContactRef{}, nil
	}

	contactIDSet := make(map[string]struct{})
	for _, p := range pivots {
		contactIDSet[p.ContactID] = struct{}{}
	}
	contactIDs := make([]string, 0, len(contactIDSet))
	for id := range contactIDSet {
		contactIDs = append(contactIDs, id)
	}
	var contacts []models.Contact
	if err := db.Where("id IN ?", contactIDs).
		Select("id", "first_name", "last_name").Find(&contacts).Error; err != nil {
		return nil, err
	}
	nameByID := make(map[string]string, len(contacts))
	for _, c := range contacts {
		nameByID[c.ID] = formatPersonName(ptrToStr(c.FirstName), ptrToStr(c.LastName))
	}

	result := make(map[uint][]dto.TaskContactRef, len(taskIDs))
	for _, p := range pivots {
		result[p.ContactTaskID] = append(result[p.ContactTaskID], dto.TaskContactRef{
			ID:   p.ContactID,
			Name: nameByID[p.ContactID],
		})
	}
	return result, nil
}

// replaceTaskAssignees swaps the assignee set for one task. Caller is
// responsible for validating that every ID belongs to the task's vault.
func replaceTaskAssignees(tx *gorm.DB, taskID uint, contactIDs []string) error {
	if err := tx.Where("contact_task_id = ?", taskID).Delete(&models.TaskContact{}).Error; err != nil {
		return err
	}
	if len(contactIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(contactIDs))
	rows := make([]models.TaskContact, 0, len(contactIDs))
	for _, cid := range contactIDs {
		if cid == "" {
			continue
		}
		if _, dup := seen[cid]; dup {
			continue
		}
		seen[cid] = struct{}{}
		rows = append(rows, models.TaskContact{ContactTaskID: taskID, ContactID: cid})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// validateContactsBelongToVault ensures every contact in the set is in vault.
// Returns ErrContactNotFound if any is missing.
func validateContactsBelongToVault(db *gorm.DB, contactIDs []string, vaultID string) error {
	if len(contactIDs) == 0 {
		return nil
	}
	var count int64
	if err := db.Model(&models.Contact{}).
		Where("id IN ? AND vault_id = ?", contactIDs, vaultID).
		Count(&count).Error; err != nil {
		return err
	}
	if int(count) != distinctCount(contactIDs) {
		return ErrContactNotFound
	}
	return nil
}

func distinctCount(ids []string) int {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
	}
	return len(seen)
}
