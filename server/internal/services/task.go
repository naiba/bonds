package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrTaskNotFound = errors.New("task not found")

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

func (s *TaskService) List(contactID string) ([]dto.TaskResponse, error) {
	var tasks []models.ContactTask
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	result := make([]dto.TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = toTaskResponse(&t)
	}
	return result, nil
}

func (s *TaskService) Create(contactID, authorID string, req dto.CreateTaskRequest) (*dto.TaskResponse, error) {
	task := models.ContactTask{
		ContactID:   contactID,
		AuthorID:    strPtrOrNil(authorID),
		AuthorName:  "User",
		Label:       req.Label,
		Description: strPtrOrNil(req.Description),
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

func (s *TaskService) Update(id uint, contactID string, req dto.UpdateTaskRequest) (*dto.TaskResponse, error) {
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
	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}
	resp := toTaskResponse(&task)
	return &resp, nil
}

func (s *TaskService) ToggleCompleted(id uint, contactID string) (*dto.TaskResponse, error) {
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
	} else {
		task.CompletedAt = nil
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

func (s *TaskService) Delete(id uint, contactID string) error {
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
		ContactID:   t.ContactID,
		AuthorID:    ptrToStr(t.AuthorID),
		Label:       t.Label,
		Description: ptrToStr(t.Description),
		Completed:   t.Completed,
		CompletedAt: t.CompletedAt,
		DueAt:       t.DueAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
