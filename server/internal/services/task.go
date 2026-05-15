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

func (s *TaskService) List(contactID, vaultID string) ([]dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var tasks []models.ContactTask
	if err := s.db.Where("contact_id = ?", contactID).Order("position ASC, created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	result := make([]dto.TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = toTaskResponse(&t)
	}
	return result, nil
}

func (s *TaskService) ListCompleted(contactID, vaultID string) ([]dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var tasks []models.ContactTask
	if err := s.db.Where("contact_id = ? AND completed = ?", contactID, true).Order("completed_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	result := make([]dto.TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = toTaskResponse(&t)
	}
	return result, nil
}

func (s *TaskService) Create(contactID, vaultID, authorID string, req dto.CreateTaskRequest) (*dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	status := resolveTaskStatusOrDefault(s.db, req.Status, vaultID)
	contactPtr := contactID
	task := models.ContactTask{
		VaultID:     vaultID,
		ContactID:   &contactPtr,
		AuthorID:    strPtrOrNil(authorID),
		AuthorName:  "User",
		Label:       req.Label,
		Description: strPtrOrNil(req.Description),
		Status:      status,
		DueAt:       req.DueAt,
	}
	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "ContactTask"
		s.feedRecorder.Record(contactID, authorID, ActionTaskCreated, "Created task: "+req.Label, &task.ID, &entityType)
	}

	resp := toTaskResponse(&task)
	return &resp, nil
}

func (s *TaskService) Update(id uint, contactID, vaultID string, req dto.UpdateTaskRequest) (*dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if req.Status != "" && !taskStatusExistsForVault(s.db, req.Status, vaultID) {
		return nil, ErrInvalidTaskStatus
	}
	var task models.ContactTask
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	task.Label = req.Label
	task.Description = strPtrOrNil(req.Description)
	task.DueAt = req.DueAt
	if req.Status != "" {
		task.Status = req.Status
	}
	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}
	resp := toTaskResponse(&task)
	return &resp, nil
}

func (s *TaskService) ToggleCompleted(id uint, contactID, vaultID string) (*dto.TaskResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var task models.ContactTask
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&task).Error; err != nil {
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

	resp := toTaskResponse(&task)
	return &resp, nil
}

func (s *TaskService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactTask{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func toTaskResponse(t *models.ContactTask) dto.TaskResponse {
	return dto.TaskResponse{
		ID:          t.ID,
		ContactID:   ptrToStr(t.ContactID),
		VaultID:     t.VaultID,
		AuthorID:    ptrToStr(t.AuthorID),
		Label:       t.Label,
		Description: ptrToStr(t.Description),
		Status:      t.Status,
		Position:    t.Position,
		Completed:   t.Completed,
		CompletedAt: t.CompletedAt,
		DueAt:       t.DueAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
