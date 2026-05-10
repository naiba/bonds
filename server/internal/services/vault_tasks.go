package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// VaultTaskService manages tasks at the vault level. Tasks can be either
// attached to a contact (ContactID set) or standalone vault-level tasks
// (ContactID NULL). Both kinds are surfaced in the same kanban board.
type VaultTaskService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewVaultTaskService(db *gorm.DB) *VaultTaskService {
	return &VaultTaskService{db: db}
}

func (s *VaultTaskService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

// VaultTaskFilters narrows the kanban list. All fields are optional.
type VaultTaskFilters struct {
	ContactID *string // nil = no filter; "" or pointer to "" treated as "standalone only"
	Status    *string // nil = no filter
}

// List returns all tasks in a vault (contact-attached and standalone),
// ordered by status column, then by Position within each column, then by
// CreatedAt as a stable tiebreaker. Optional filters narrow the result.
func (s *VaultTaskService) List(vaultID string, filters VaultTaskFilters) ([]dto.VaultTaskResponse, error) {
	q := s.db.Where("vault_id = ?", vaultID)
	if filters.ContactID != nil {
		if *filters.ContactID == "" {
			q = q.Where("contact_id IS NULL")
		} else {
			q = q.Where("contact_id = ?", *filters.ContactID)
		}
	}
	if filters.Status != nil && *filters.Status != "" {
		q = q.Where("status = ?", *filters.Status)
	}

	var tasks []models.ContactTask
	if err := q.Order("status ASC, position ASC, created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return []dto.VaultTaskResponse{}, nil
	}

	contactNames, err := s.collectContactNames(tasks)
	if err != nil {
		return nil, err
	}

	result := make([]dto.VaultTaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = toVaultTaskResponse(&t, contactNames)
	}
	return result, nil
}

// Create makes a vault-level task. ContactID is optional; when empty, the
// task is standalone. When set, the contact must belong to the same vault.
func (s *VaultTaskService) Create(vaultID, authorID string, req dto.CreateVaultTaskRequest) (*dto.VaultTaskResponse, error) {
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	var contactPtr *string
	if req.ContactID != "" {
		if err := validateContactBelongsToVault(s.db, req.ContactID, vaultID); err != nil {
			return nil, err
		}
		c := req.ContactID
		contactPtr = &c
	}

	task := models.ContactTask{
		VaultID:     vaultID,
		ContactID:   contactPtr,
		AuthorID:    strPtrOrNil(authorID),
		AuthorName:  "User",
		Label:       req.Label,
		Description: strPtrOrNil(req.Description),
		Status:      resolveTaskStatusOrDefault(s.db, req.Status, vaultID),
		DueAt:       req.DueAt,
	}
	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil && contactPtr != nil {
		entityType := "ContactTask"
		s.feedRecorder.Record(*contactPtr, authorID, ActionTaskCreated, "Created task: "+req.Label, &task.ID, &entityType)
	}

	contactNames, err := s.collectContactNames([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	resp := toVaultTaskResponse(&task, contactNames)
	return &resp, nil
}

// Update replaces the editable fields of a vault task in one call. Used by
// the click-to-edit modal. ContactID may be cleared (empty string) or
// changed to a different contact in the same vault. Status is also kept in
// sync with Completed so the list view stays consistent.
func (s *VaultTaskService) Update(id uint, vaultID string, req dto.UpdateVaultTaskRequest) (*dto.VaultTaskResponse, error) {
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}

	var task models.ContactTask
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	// Resolve contact pointer: empty = standalone, non-empty = must belong to vault
	var contactPtr *string
	if req.ContactID != "" {
		if err := validateContactBelongsToVault(s.db, req.ContactID, vaultID); err != nil {
			return nil, err
		}
		c := req.ContactID
		contactPtr = &c
	}

	updates := map[string]interface{}{
		"label":       req.Label,
		"description": strPtrOrNil(req.Description),
		"due_at":      req.DueAt,
		"contact_id":  contactPtr,
	}
	if req.Status != "" {
		updates["status"] = req.Status
		// Mirror the Completed/Status sync logic from UpdateStatus so the
		// modal and the kanban-drag flow agree.
		if req.Status == models.TaskStatusDone && !task.Completed {
			now := time.Now()
			updates["completed"] = true
			updates["completed_at"] = &now
		} else if req.Status != models.TaskStatusDone && task.Completed {
			updates["completed"] = false
			updates["completed_at"] = nil
		}
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	// Reload to pick up canonical values (timestamps, etc.)
	if err := s.db.Where("id = ?", task.ID).First(&task).Error; err != nil {
		return nil, err
	}

	contactNames, err := s.collectContactNames([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	resp := toVaultTaskResponse(&task, contactNames)
	return &resp, nil
}

// UpdateStatus moves a task to a different kanban column. Used by drag-drop
// across columns. Validates that the target status is recognized.
func (s *VaultTaskService) UpdateStatus(id uint, vaultID string, req dto.UpdateTaskStatusRequest) (*dto.VaultTaskResponse, error) {
	if !models.IsValidTaskStatus(req.Status) {
		return nil, ErrInvalidTaskStatus
	}
	var task models.ContactTask
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{"status": req.Status}
	// Keep Completed in sync with the Done column so existing contact-level
	// list views (which filter on Completed) continue to behave correctly.
	if req.Status == models.TaskStatusDone && !task.Completed {
		now := time.Now()
		updates["completed"] = true
		updates["completed_at"] = &now
		task.Completed = true
		task.CompletedAt = &now
	} else if req.Status != models.TaskStatusDone && task.Completed {
		updates["completed"] = false
		updates["completed_at"] = nil
		task.Completed = false
		task.CompletedAt = nil
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	task.Status = req.Status

	contactNames, err := s.collectContactNames([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	resp := toVaultTaskResponse(&task, contactNames)
	return &resp, nil
}

// UpdatePosition reorders a task within (or across) columns. The Status field
// is optional in the request — when present, the task is also moved to that
// column (drag across columns + reorder in one call). Position is the new
// 0-based index within the destination column.
func (s *VaultTaskService) UpdatePosition(id uint, vaultID string, req dto.UpdateTaskPositionRequest) (*dto.VaultTaskResponse, error) {
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	var task models.ContactTask
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{"position": req.Position}
	if req.Status != "" {
		updates["status"] = req.Status
		task.Status = req.Status
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	task.Position = req.Position

	contactNames, err := s.collectContactNames([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	resp := toVaultTaskResponse(&task, contactNames)
	return &resp, nil
}

// collectContactNames batches a single SELECT for all referenced contacts so
// the list endpoint stays at 2 queries (tasks + contacts) regardless of size.
func (s *VaultTaskService) collectContactNames(tasks []models.ContactTask) (map[string]string, error) {
	idSet := make(map[string]struct{})
	for _, t := range tasks {
		if t.ContactID != nil {
			idSet[*t.ContactID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return map[string]string{}, nil
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var contacts []models.Contact
	if err := s.db.Where("id IN ?", ids).Select("id", "first_name", "last_name").Find(&contacts).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(contacts))
	for _, c := range contacts {
		first := ptrToStr(c.FirstName)
		last := ptrToStr(c.LastName)
		out[c.ID] = formatPersonName(first, last)
	}
	return out, nil
}

func formatPersonName(first, last string) string {
	if first == "" && last == "" {
		return ""
	}
	if last == "" {
		return first
	}
	if first == "" {
		return last
	}
	return fmt.Sprintf("%s %s", first, last)
}

func toVaultTaskResponse(t *models.ContactTask, names map[string]string) dto.VaultTaskResponse {
	contactID := ptrToStr(t.ContactID)
	contactName := ""
	if contactID != "" {
		contactName = names[contactID]
	}
	desc := ""
	if t.Description != nil {
		desc = *t.Description
	}
	return dto.VaultTaskResponse{
		ID:          t.ID,
		ContactID:   contactID,
		VaultID:     t.VaultID,
		ContactName: contactName,
		AuthorName:  t.AuthorName,
		Label:       t.Label,
		Description: desc,
		Status:      t.Status,
		Position:    t.Position,
		Completed:   t.Completed,
		CompletedAt: t.CompletedAt,
		DueAt:       t.DueAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
