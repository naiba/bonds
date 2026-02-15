package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type MoodTrackingService struct {
	db *gorm.DB
}

func NewMoodTrackingService(db *gorm.DB) *MoodTrackingService {
	return &MoodTrackingService{db: db}
}

func (s *MoodTrackingService) Create(contactID, vaultID string, req dto.CreateMoodTrackingEventRequest) (*dto.MoodTrackingEventResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	event := models.MoodTrackingEvent{
		ContactID:               contactID,
		MoodTrackingParameterID: req.MoodTrackingParameterID,
		RatedAt:                 req.RatedAt,
		Note:                    strPtrOrNil(req.Note),
		NumberOfHoursSlept:      req.NumberOfHoursSlept,
	}
	if err := s.db.Create(&event).Error; err != nil {
		return nil, err
	}
	resp := toMoodTrackingEventResponse(&event)
	return &resp, nil
}

func (s *MoodTrackingService) List(contactID, vaultID string) ([]dto.MoodTrackingEventResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var events []models.MoodTrackingEvent
	if err := s.db.Where("contact_id = ?", contactID).Order("rated_at DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	result := make([]dto.MoodTrackingEventResponse, len(events))
	for i, e := range events {
		result[i] = toMoodTrackingEventResponse(&e)
	}
	return result, nil
}

func toMoodTrackingEventResponse(e *models.MoodTrackingEvent) dto.MoodTrackingEventResponse {
	return dto.MoodTrackingEventResponse{
		ID:                      e.ID,
		ContactID:               e.ContactID,
		MoodTrackingParameterID: e.MoodTrackingParameterID,
		RatedAt:                 e.RatedAt,
		Note:                    ptrToStr(e.Note),
		NumberOfHoursSlept:      e.NumberOfHoursSlept,
		CreatedAt:               e.CreatedAt,
		UpdatedAt:               e.UpdatedAt,
	}
}
