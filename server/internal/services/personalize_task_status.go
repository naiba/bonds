package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// ErrTaskStatusUndeletable is returned by Delete when the user tries to remove
// a status flagged can_be_deleted=false (the seeded core columns).
var ErrTaskStatusUndeletable = errors.New("this task status cannot be deleted")

var taskStatusSlugRunPattern = regexp.MustCompile(`[^a-z0-9]+`)

// taskStatusSlugify produces a URL-safe lowercase identifier from a display
// name. "In Review" -> "in-review". Stricter than the existing
// monica-import slugify: collapses any non-alphanumeric run into a single
// dash, trims edges, and substitutes "status" if the result is empty.
func taskStatusSlugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = taskStatusSlugRunPattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "status"
	}
	return s
}

// createTaskStatus is invoked by PersonalizeService.Create when entity is
// "task-statuses". The standard generic insert can't satisfy our schema
// (Slug is NOT NULL, Position needs a sensible default, can_be_deleted is
// always true for user-created rows) so we run a focused custom insert.
func (s *PersonalizeService) createTaskStatus(accountID string, req dto.PersonalizeEntityRequest) (*dto.PersonalizeEntityResponse, error) {
	val := req.Label
	if val == "" {
		val = req.Name
	}
	if strings.TrimSpace(val) == "" {
		return nil, fmt.Errorf("name required")
	}

	// Auto-derive slug; ensure unique per account by appending -2, -3, …
	base := taskStatusSlugify(val)
	slug := base
	for suffix := 2; suffix < 1000; suffix++ {
		var count int64
		if err := s.db.Model(&models.TaskStatus{}).
			Where("account_id = ? AND slug = ?", accountID, slug).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, suffix)
	}

	// Position = (max existing position) + 1
	var maxPos int
	row := s.db.Model(&models.TaskStatus{}).
		Select("COALESCE(MAX(position), -1)").
		Where("account_id = ?", accountID).
		Row()
	_ = row.Scan(&maxPos)

	now := time.Now()
	status := models.TaskStatus{
		AccountID:    accountID,
		Name:         &val,
		Slug:         slug,
		Position:     maxPos + 1,
		IsDefault:    false,
		CanBeDeleted: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.db.Create(&status).Error; err != nil {
		return nil, err
	}

	resp := dto.PersonalizeEntityResponse{
		ID:        status.ID,
		Label:     val,
		Name:      val,
		CreatedAt: status.CreatedAt,
		UpdatedAt: status.UpdatedAt,
	}
	return &resp, nil
}

// deleteTaskStatus is invoked by PersonalizeService.Delete when entity is
// "task-statuses". It enforces can_be_deleted and cascades any tasks that
// reference the status by slug onto the account's default status — so the
// kanban never has orphan tasks pointing at a non-existent status.
func (s *PersonalizeService) deleteTaskStatus(accountID string, id uint) error {
	var status models.TaskStatus
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPersonalizeEntityNotFound
		}
		return err
	}
	if !status.CanBeDeleted {
		return ErrTaskStatusUndeletable
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Find the default status for this account so orphan tasks land
		// somewhere sensible. If no default is marked, fall back to the
		// lowest-position remaining status.
		var defaultStatus models.TaskStatus
		err := tx.Where("account_id = ? AND id != ? AND is_default = ?", accountID, id, true).
			First(&defaultStatus).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = tx.Where("account_id = ? AND id != ?", accountID, id).
				Order("position ASC").
				First(&defaultStatus).Error
		}
		if err != nil {
			return err
		}

		// Reassign tasks that point at the slug being deleted.
		if err := tx.Model(&models.ContactTask{}).
			Where("status = ?", status.Slug).
			Update("status", defaultStatus.Slug).Error; err != nil {
			return err
		}

		if err := tx.Delete(&status).Error; err != nil {
			return err
		}
		return nil
	})
}

// listTaskStatuses returns the account's task statuses ordered by position,
// with the extra metadata (slug, is_default, can_be_deleted) populated so
// the kanban frontend can render columns and protect undeletable rows.
func (s *PersonalizeService) listTaskStatuses(accountID string) ([]dto.PersonalizeEntityResponse, error) {
	var statuses []models.TaskStatus
	if err := s.db.Where("account_id = ?", accountID).
		Order("position ASC, id ASC").
		Find(&statuses).Error; err != nil {
		return nil, err
	}
	results := make([]dto.PersonalizeEntityResponse, len(statuses))
	for i, st := range statuses {
		name := ""
		if st.Name != nil {
			name = *st.Name
		}
		pos := st.Position
		isDefault := st.IsDefault
		canDelete := st.CanBeDeleted
		results[i] = dto.PersonalizeEntityResponse{
			ID:           st.ID,
			Label:        name,
			Name:         name,
			Position:     &pos,
			Slug:         st.Slug,
			IsDefault:    &isDefault,
			CanBeDeleted: &canDelete,
			CreatedAt:    st.CreatedAt,
			UpdatedAt:    st.UpdatedAt,
		}
	}
	return results, nil
}

// DefaultTaskStatusSlug returns the slug of the account's default status.
// Used by VaultTaskService.Create when the request leaves Status empty.
// Falls back to "todo" if the account somehow has no default seeded.
func (s *PersonalizeService) DefaultTaskStatusSlug(accountID string) string {
	var status models.TaskStatus
	if err := s.db.Where("account_id = ? AND is_default = ?", accountID, true).
		First(&status).Error; err == nil {
		return status.Slug
	}
	if err := s.db.Where("account_id = ?", accountID).
		Order("position ASC").
		First(&status).Error; err == nil {
		return status.Slug
	}
	return "todo"
}
