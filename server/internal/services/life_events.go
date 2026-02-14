package services

import (
	"errors"
	"math"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

var (
	ErrTimelineEventNotFound = errors.New("timeline event not found")
	ErrLifeEventNotFound     = errors.New("life event not found")
)

type LifeEventService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewLifeEventService(db *gorm.DB) *LifeEventService {
	return &LifeEventService{db: db}
}

func (s *LifeEventService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *LifeEventService) ListTimelineEvents(contactID string, page, perPage int) ([]dto.TimelineEventResponse, response.Meta, error) {
	var participantIDs []uint
	s.db.Model(&models.TimelineEventParticipant{}).Where("contact_id = ?", contactID).Pluck("timeline_event_id", &participantIDs)

	query := s.db.Where("id IN ?", participantIDs)

	var total int64
	if err := query.Model(&models.TimelineEvent{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	offset := (page - 1) * perPage

	var events []models.TimelineEvent
	if err := query.Preload("LifeEvents").Offset(offset).Limit(perPage).Order("started_at DESC").Find(&events).Error; err != nil {
		return nil, response.Meta{}, err
	}

	result := make([]dto.TimelineEventResponse, len(events))
	for i, e := range events {
		result[i] = toTimelineEventResponse(&e)
	}

	meta := response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}
	return result, meta, nil
}

func (s *LifeEventService) CreateTimelineEvent(contactID, vaultID string, req dto.CreateTimelineEventRequest) (*dto.TimelineEventResponse, error) {
	label := req.Label
	event := models.TimelineEvent{
		VaultID:   vaultID,
		StartedAt: req.StartedAt,
		Label:     strPtrOrNil(label),
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		p := models.TimelineEventParticipant{
			ContactID:       contactID,
			TimelineEventID: event.ID,
		}
		return tx.Create(&p).Error
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "TimelineEvent"
		s.feedRecorder.Record(contactID, "", ActionLifeEventCreated, "Created a life event", &event.ID, &entityType)
	}

	resp := toTimelineEventResponse(&event)
	return &resp, nil
}

func (s *LifeEventService) AddLifeEvent(timelineEventID uint, req dto.CreateLifeEventRequest) (*dto.LifeEventResponse, error) {
	var te models.TimelineEvent
	if err := s.db.First(&te, timelineEventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimelineEventNotFound
		}
		return nil, err
	}

	le := models.LifeEvent{
		TimelineEventID:   timelineEventID,
		LifeEventTypeID:   req.LifeEventTypeID,
		HappenedAt:        req.HappenedAt,
		Summary:           strPtrOrNil(req.Summary),
		Description:       strPtrOrNil(req.Description),
		Costs:             req.Costs,
		CurrencyID:        req.CurrencyID,
		DurationInMinutes: req.DurationInMinutes,
		Distance:          req.Distance,
		DistanceUnit:      strPtrOrNil(req.DistanceUnit),
		FromPlace:         strPtrOrNil(req.FromPlace),
		ToPlace:           strPtrOrNil(req.ToPlace),
		Place:             strPtrOrNil(req.Place),
	}
	if err := s.db.Create(&le).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventResponse(&le)
	return &resp, nil
}

func (s *LifeEventService) UpdateLifeEvent(timelineEventID, lifeEventID uint, req dto.UpdateLifeEventRequest) (*dto.LifeEventResponse, error) {
	var le models.LifeEvent
	if err := s.db.Where("id = ? AND timeline_event_id = ?", lifeEventID, timelineEventID).First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeEventNotFound
		}
		return nil, err
	}

	if req.LifeEventTypeID != 0 {
		le.LifeEventTypeID = req.LifeEventTypeID
	}
	if !req.HappenedAt.IsZero() {
		le.HappenedAt = req.HappenedAt
	}
	le.Summary = strPtrOrNil(req.Summary)
	le.Description = strPtrOrNil(req.Description)
	le.Costs = req.Costs
	le.CurrencyID = req.CurrencyID
	le.DurationInMinutes = req.DurationInMinutes
	le.Distance = req.Distance
	le.DistanceUnit = strPtrOrNil(req.DistanceUnit)
	le.FromPlace = strPtrOrNil(req.FromPlace)
	le.ToPlace = strPtrOrNil(req.ToPlace)
	le.Place = strPtrOrNil(req.Place)

	if err := s.db.Save(&le).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventResponse(&le)
	return &resp, nil
}

func (s *LifeEventService) DeleteTimelineEvent(id uint) error {
	var te models.TimelineEvent
	if err := s.db.First(&te, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimelineEventNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("timeline_event_id = ?", id).Delete(&models.LifeEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("timeline_event_id = ?", id).Delete(&models.TimelineEventParticipant{}).Error; err != nil {
			return err
		}
		return tx.Delete(&te).Error
	})
}

func (s *LifeEventService) DeleteLifeEvent(timelineEventID, lifeEventID uint) error {
	var le models.LifeEvent
	if err := s.db.Where("id = ? AND timeline_event_id = ?", lifeEventID, timelineEventID).First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeEventNotFound
		}
		return err
	}
	return s.db.Delete(&le).Error
}

func toTimelineEventResponse(e *models.TimelineEvent) dto.TimelineEventResponse {
	resp := dto.TimelineEventResponse{
		ID:        e.ID,
		VaultID:   e.VaultID,
		StartedAt: e.StartedAt,
		Label:     ptrToStr(e.Label),
		Collapsed: e.Collapsed,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.LifeEvents != nil {
		les := make([]dto.LifeEventResponse, len(e.LifeEvents))
		for i, le := range e.LifeEvents {
			les[i] = toLifeEventResponse(&le)
		}
		resp.LifeEvents = les
	}
	return resp
}

func toLifeEventResponse(le *models.LifeEvent) dto.LifeEventResponse {
	return dto.LifeEventResponse{
		ID:                le.ID,
		TimelineEventID:   le.TimelineEventID,
		LifeEventTypeID:   le.LifeEventTypeID,
		HappenedAt:        le.HappenedAt,
		Summary:           ptrToStr(le.Summary),
		Description:       ptrToStr(le.Description),
		Costs:             le.Costs,
		CurrencyID:        le.CurrencyID,
		DurationInMinutes: le.DurationInMinutes,
		Distance:          le.Distance,
		DistanceUnit:      ptrToStr(le.DistanceUnit),
		FromPlace:         ptrToStr(le.FromPlace),
		ToPlace:           ptrToStr(le.ToPlace),
		Place:             ptrToStr(le.Place),
		CreatedAt:         le.CreatedAt,
		UpdatedAt:         le.UpdatedAt,
	}
}
