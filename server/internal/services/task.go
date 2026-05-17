package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrTaskNotFound = errors.New("task not found")
var ErrInvalidTaskStatus = errors.New("invalid task status")
var ErrInvalidParentTask = errors.New("invalid parent task")
var ErrTaskHasSubTasks = errors.New("task has sub-tasks")

type TaskService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{db: db}
}

func (s *TaskService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

// List returns the tasks for which the given contact is an assignee, ordered
// by position then most-recent-created.
func (s *TaskService) List(contactID, vaultID string) ([]dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var tasks []models.ContactTask
	if err := s.db.
		Joins("JOIN task_contacts tc ON tc.contact_task_id = contact_tasks.id").
		Where("tc.contact_id = ? AND contact_tasks.vault_id = ?", contactID, vaultID).
		Order("contact_tasks.position ASC, contact_tasks.created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return buildTaskResponses(s.db, tasks)
}

func (s *TaskService) ListCompleted(contactID, vaultID string) ([]dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var tasks []models.ContactTask
	if err := s.db.
		Joins("JOIN task_contacts tc ON tc.contact_task_id = contact_tasks.id").
		Where("tc.contact_id = ? AND contact_tasks.vault_id = ? AND contact_tasks.completed = ?", contactID, vaultID, true).
		Order("contact_tasks.completed_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return buildTaskResponses(s.db, tasks)
}

func (s *TaskService) Create(contactID, vaultID, authorID string, req dto.CreateTaskRequest) (*dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	extras := append([]string{contactID}, req.ContactIDs...)
	if err := validateContactsBelongToVault(s.db, extras, vaultID); err != nil {
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
		return replaceTaskAssigneesLocked(tx, task.ID, extras)
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "ContactTask"
		s.feedRecorder.Record(contactID, authorID, ActionTaskCreated, "Created task: "+req.Label, &task.ID, &entityType)
	}

	resps, err := buildTaskResponses(s.db, []models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

func (s *TaskService) Update(id uint, contactID, vaultID string, req dto.UpdateTaskRequest) (*dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	var task models.ContactTask
	if err := s.db.
		Joins("JOIN task_contacts tc ON tc.contact_task_id = contact_tasks.id").
		Where("contact_tasks.id = ? AND contact_tasks.vault_id = ? AND tc.contact_id = ?", id, vaultID, contactID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	parentPatch := req.ParentTaskID
	if err := validateParentTaskPatch(s.db, parentPatch, task.ID, vaultID); err != nil {
		return nil, err
	}

	task.Label = req.Label
	task.Description = strPtrOrNil(req.Description)
	task.DueAt = req.DueAt
	if parentPatch.Present {
		task.ParentTaskID = parentPatch.Ptr()
	}
	if req.Status != "" {
		task.Status = req.Status
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&task).Error; err != nil {
			return err
		}
		if req.ContactIDs == nil {
			return nil
		}
		next := append([]string{contactID}, (*req.ContactIDs)...)
		if err := validateContactsBelongToVault(tx, next, vaultID); err != nil {
			return err
		}
		return replaceTaskAssigneesLocked(tx, task.ID, next)
	})
	if err != nil {
		return nil, err
	}

	resps, err := buildTaskResponses(s.db, []models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

func (s *TaskService) ToggleCompleted(id uint, contactID, vaultID string) (*dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var task models.ContactTask
	if err := s.db.
		Joins("JOIN task_contacts tc ON tc.contact_task_id = contact_tasks.id").
		Where("contact_tasks.id = ? AND contact_tasks.vault_id = ? AND tc.contact_id = ?", id, vaultID, contactID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	task.Completed = !task.Completed
	if task.Completed {
		now := time.Now()
		task.CompletedAt = &now
		task.Status = models.TaskStatusDone
	} else {
		task.CompletedAt = nil
		if task.Status == models.TaskStatusDone {
			task.Status = models.TaskStatusTodo
		}
	}
	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil && task.Completed {
		entityType := "ContactTask"
		s.feedRecorder.Record(contactID, "", ActionTaskCompleted, "Completed task: "+task.Label, &task.ID, &entityType)
	}

	resps, err := buildTaskResponses(s.db, []models.ContactTask{task})
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

func (s *TaskService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	var task models.ContactTask
	if err := s.db.
		Joins("JOIN task_contacts tc ON tc.contact_task_id = contact_tasks.id").
		Where("contact_tasks.id = ? AND contact_tasks.vault_id = ? AND tc.contact_id = ?", id, vaultID, contactID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return err
	}
	return deleteTaskCascade(s.db, &task)
}

// validateParentTask ensures the parent exists in the same vault and that
// the resulting hierarchy stays acyclic. selfID is 0 on create.
//
// Cycle detection: walk parentID -> parent.ParentTaskID -> ... up to a
// hard bound (defends against pre-existing cycles in the data). Rejects
// when the walk encounters selfID or exceeds the bound.
func validateParentTask(db *gorm.DB, parentID *uint, selfID uint, vaultID string) error {
	if parentID == nil {
		return nil
	}
	if selfID != 0 && *parentID == selfID {
		return ErrInvalidParentTask
	}
	const maxDepth = 256
	current := *parentID
	visited := make(map[uint]struct{}, 8)
	for i := 0; i < maxDepth; i++ {
		if _, loop := visited[current]; loop {
			return ErrInvalidParentTask
		}
		visited[current] = struct{}{}

		var ancestor models.ContactTask
		if err := db.Select("id", "vault_id", "parent_task_id").
			Where("id = ? AND vault_id = ?", current, vaultID).
			First(&ancestor).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidParentTask
			}
			return err
		}
		if ancestor.ParentTaskID == nil {
			return nil
		}
		if selfID != 0 && *ancestor.ParentTaskID == selfID {
			return ErrInvalidParentTask
		}
		current = *ancestor.ParentTaskID
	}
	return ErrInvalidParentTask
}

// validateParentTaskPatch is the NullableUint flavor for Update endpoints
// that need to distinguish "clear parent" from "leave parent unchanged".
func validateParentTaskPatch(db *gorm.DB, patch dto.NullableUint, selfID uint, vaultID string) error {
	if !patch.Present || patch.Cleared() {
		return nil
	}
	return validateParentTask(db, patch.Ptr(), selfID, vaultID)
}

// deleteTaskCascade removes a task and every descendant in the sub-task tree
// in one transaction. Pivot rows are wiped first so the FK direction is safe.
// Uses a breadth-first walk over parent_task_id to avoid recursive CTE
// portability concerns between SQLite and Postgres.
func deleteTaskCascade(db *gorm.DB, task *models.ContactTask) error {
	return db.Transaction(func(tx *gorm.DB) error {
		ids := []uint{task.ID}
		frontier := []uint{task.ID}
		for len(frontier) > 0 {
			var children []uint
			if err := tx.Model(&models.ContactTask{}).
				Where("parent_task_id IN ?", frontier).
				Pluck("id", &children).Error; err != nil {
				return err
			}
			if len(children) == 0 {
				break
			}
			ids = append(ids, children...)
			frontier = children
		}
		if err := tx.Where("contact_task_id IN ?", ids).Delete(&models.TaskContact{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&models.ContactTask{}).Error
	})
}

func buildTaskResponses(db *gorm.DB, tasks []models.ContactTask) ([]dto.TaskResponse, error) {
	ids := make([]uint, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	assignees, err := taskAssignees(db, ids)
	if err != nil {
		return nil, err
	}
	out := make([]dto.TaskResponse, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskResponse(&t, assignees[t.ID])
	}
	return out, nil
}

func toTaskResponse(t *models.ContactTask, contacts []dto.TaskContactRef) dto.TaskResponse {
	if contacts == nil {
		contacts = []dto.TaskContactRef{}
	}
	return dto.TaskResponse{
		ID:           t.ID,
		VaultID:      t.VaultID,
		AuthorID:     ptrToStr(t.AuthorID),
		Label:        t.Label,
		Description:  ptrToStr(t.Description),
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
