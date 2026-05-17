package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// VaultTaskService manages tasks at the vault level. A task has zero or more
// contact assignees via the task_contacts pivot — zero = standalone.
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
	// ContactID: nil = no filter; pointer to "" = standalone only (no
	// assignees); pointer to a real ID = tasks where that contact is among
	// the assignees.
	ContactID *string
	Status    *string // nil = no filter
}

// List returns all tasks in a vault, ordered by status column then Position
// within each column, with CreatedAt as a stable tiebreaker.
func (s *VaultTaskService) List(vaultID string, filters VaultTaskFilters) ([]dto.VaultTaskResponse, error) {
	q := s.db.Model(&models.ContactTask{}).Where("contact_tasks.vault_id = ?", vaultID)
	if filters.ContactID != nil {
		if *filters.ContactID == "" {
			// Standalone: no row in the pivot.
			q = q.Where(`NOT EXISTS (
				SELECT 1 FROM task_contacts tc WHERE tc.contact_task_id = contact_tasks.id
			)`)
		} else {
			q = q.Where(`EXISTS (
				SELECT 1 FROM task_contacts tc
				WHERE tc.contact_task_id = contact_tasks.id AND tc.contact_id = ?
			)`, *filters.ContactID)
		}
	}
	if filters.Status != nil && *filters.Status != "" {
		q = q.Where("contact_tasks.status = ?", *filters.Status)
	}

	var tasks []models.ContactTask
	if err := q.Order("contact_tasks.status ASC, contact_tasks.position ASC, contact_tasks.created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return s.buildResponses(tasks)
}

func (s *VaultTaskService) Create(vaultID, authorID string, req dto.CreateVaultTaskRequest) (*dto.VaultTaskResponse, error) {
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	if err := validateContactsBelongToVault(s.db, req.ContactIDs, vaultID); err != nil {
		return nil, err
	}
	if err := validateParentTask(s.db, req.ParentTaskID, 0, vaultID); err != nil {
		return nil, err
	}

	task := models.ContactTask{
		VaultID:      vaultID,
		ParentTaskID: req.ParentTaskID,
		AuthorID:     strPtrOrNil(authorID),
		AuthorName:   "User",
		Label:        req.Label,
		Description:  strPtrOrNil(req.Description),
		Status:       resolveTaskStatusOrDefault(s.db, req.Status, vaultID),
		DueAt:        req.DueAt,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		return replaceTaskAssigneesLocked(tx, task.ID, req.ContactIDs)
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "ContactTask"
		// Feed entry is per-assignee so each contact's feed reflects the
		// task that was just created for them.
		for _, cid := range req.ContactIDs {
			s.feedRecorder.Record(cid, authorID, ActionTaskCreated, "Created task: "+req.Label, &task.ID, &entityType)
		}
	}

	resps, err := s.buildResponses([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

// Update replaces the editable fields of a vault task in one call. Used by
// the click-to-edit modal. When ContactIDs is provided, the assignee set is
// replaced; nil means "leave assignees untouched".
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
	if err := validateParentTaskPatch(s.db, req.ParentTaskID, task.ID, vaultID); err != nil {
		return nil, err
	}
	if req.ContactIDs != nil {
		if err := validateContactsBelongToVault(s.db, *req.ContactIDs, vaultID); err != nil {
			return nil, err
		}
	}

	updates := map[string]interface{}{
		"label":       req.Label,
		"description": strPtrOrNil(req.Description),
		"due_at":      req.DueAt,
	}
	if req.ParentTaskID.Present {
		updates["parent_task_id"] = req.ParentTaskID.Ptr()
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

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&task).Updates(updates).Error; err != nil {
			return err
		}
		if req.ContactIDs != nil {
			if err := replaceTaskAssigneesLocked(tx, task.ID, *req.ContactIDs); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Reload to pick up canonical values (timestamps, etc.)
	if err := s.db.Where("id = ?", task.ID).First(&task).Error; err != nil {
		return nil, err
	}

	resps, err := s.buildResponses([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

// UpdateStatus moves a task to a different kanban column. Used by drag-drop
// across columns. Validates that the target status is recognized.
func (s *VaultTaskService) UpdateStatus(id uint, vaultID string, req dto.UpdateTaskStatusRequest) (*dto.VaultTaskResponse, error) {
	if !taskStatusExistsForVault(s.db, req.Status, vaultID) {
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

	resps, err := s.buildResponses([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

// Delete removes a vault task and its entire sub-task tree. Returns
// ErrTaskNotFound if the task doesn't belong to the given vault.
func (s *VaultTaskService) Delete(id uint, vaultID string) error {
	var task models.ContactTask
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return err
	}
	return deleteTaskCascade(s.db, &task)
}

// UpdatePosition reorders a task within (or across) columns.
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
	destinationStatus := task.Status
	if req.Status != "" {
		destinationStatus = req.Status
	}
	if err := s.resequenceTaskPositions(task, destinationStatus, req.Position); err != nil {
		return nil, err
	}
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&task).Error; err != nil {
		return nil, err
	}

	resps, err := s.buildResponses([]models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

func (s *VaultTaskService) resequenceTaskPositions(task models.ContactTask, destinationStatus string, destinationPosition int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ContactTask{}).
			Where("id = ? AND vault_id = ?", task.ID, task.VaultID).
			Updates(map[string]interface{}{"status": destinationStatus}).Error; err != nil {
			return err
		}

		if destinationStatus != task.Status {
			if err := resequenceExistingTaskColumn(tx, task.VaultID, task.ID, task.Status); err != nil {
				return err
			}
		}
		return resequenceTaskColumn(tx, task.VaultID, task.ID, destinationStatus, destinationStatus, destinationPosition)
	})
}

func resequenceExistingTaskColumn(tx *gorm.DB, vaultID string, movedTaskID uint, status string) error {
	var tasks []models.ContactTask
	if err := tx.Where("vault_id = ? AND status = ? AND id != ?", vaultID, status, movedTaskID).
		Order("position ASC, created_at ASC, id ASC").
		Find(&tasks).Error; err != nil {
		return err
	}

	orderedIDs := make([]uint, 0, len(tasks))
	for _, t := range tasks {
		orderedIDs = append(orderedIDs, t.ID)
	}
	return updateTaskColumnPositions(tx, orderedIDs)
}

func resequenceTaskColumn(tx *gorm.DB, vaultID string, movedTaskID uint, status, destinationStatus string, destinationPosition int) error {
	var tasks []models.ContactTask
	if err := tx.Where("vault_id = ? AND status = ? AND id != ?", vaultID, status, movedTaskID).
		Order("position ASC, created_at ASC, id ASC").
		Find(&tasks).Error; err != nil {
		return err
	}

	if status == destinationStatus {
		insertAt := destinationPosition
		if insertAt < 0 {
			insertAt = 0
		}
		if insertAt > len(tasks) {
			insertAt = len(tasks)
		}

		orderedIDs := make([]uint, 0, len(tasks)+1)
		for i, t := range tasks {
			if i == insertAt {
				orderedIDs = append(orderedIDs, movedTaskID)
			}
			orderedIDs = append(orderedIDs, t.ID)
		}
		if insertAt == len(tasks) {
			orderedIDs = append(orderedIDs, movedTaskID)
		}
		return updateTaskColumnPositions(tx, orderedIDs)
	}

	orderedIDs := make([]uint, 0, len(tasks))
	for _, t := range tasks {
		orderedIDs = append(orderedIDs, t.ID)
	}
	return updateTaskColumnPositions(tx, orderedIDs)
}

func updateTaskColumnPositions(tx *gorm.DB, orderedIDs []uint) error {
	for position, id := range orderedIDs {
		if err := tx.Model(&models.ContactTask{}).
			Where("id = ?", id).
			Update("position", position).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *VaultTaskService) buildResponses(tasks []models.ContactTask) ([]dto.VaultTaskResponse, error) {
	if len(tasks) == 0 {
		return []dto.VaultTaskResponse{}, nil
	}
	ids := make([]uint, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	assignees, err := taskAssignees(s.db, ids)
	if err != nil {
		return nil, err
	}
	out := make([]dto.VaultTaskResponse, len(tasks))
	for i, t := range tasks {
		out[i] = toVaultTaskResponse(&t, assignees[t.ID])
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

func toVaultTaskResponse(t *models.ContactTask, contacts []dto.TaskContactRef) dto.VaultTaskResponse {
	if contacts == nil {
		contacts = []dto.TaskContactRef{}
	}
	desc := ""
	if t.Description != nil {
		desc = *t.Description
	}
	return dto.VaultTaskResponse{
		ID:           t.ID,
		VaultID:      t.VaultID,
		AuthorName:   t.AuthorName,
		Label:        t.Label,
		Description:  desc,
		Status:       t.Status,
		Position:     t.Position,
		Completed:    t.Completed,
		CompletedAt:  t.CompletedAt,
		DueAt:        t.DueAt,
		ParentTaskID: t.ParentTaskID,
		Contacts:     contacts,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
