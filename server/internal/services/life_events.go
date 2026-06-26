package services

import (
	"errors"
	"math"
	"sort"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/utils"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (s *LifeEventService) ListTimelineEvents(contactID, vaultID string, page, perPage int) ([]dto.TimelineEventResponse, response.Meta, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, response.Meta{}, err
	}
	participantIDs, err := timelineIDsForContact(s.db, contactID)
	if err != nil {
		return nil, response.Meta{}, err
	}

	query := s.db.Where("id IN ? AND vault_id = ?", participantIDs, vaultID)

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
	if err := query.Preload("Participants").Preload("LifeEvents.Participants").Offset(offset).Limit(perPage).Order("started_at DESC").Find(&events).Error; err != nil {
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

func (s *LifeEventService) ListVaultTimelineEvents(vaultID string, page, perPage int) ([]dto.TimelineEventResponse, response.Meta, error) {
	query := s.db.Where("vault_id = ?", vaultID)

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
	if err := s.db.Where("vault_id = ?", vaultID).Preload("Participants").Preload("LifeEvents.Participants").Offset(offset).Limit(perPage).Order("started_at DESC").Find(&events).Error; err != nil {
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

func (s *LifeEventService) CreateDashboardLifeEvent(vaultID string, req dto.CreateLifeEventRequest) (*dto.TimelineEventResponse, error) {
	participantIDs := dedupeContactIDs(req.Participants)
	if err := validateContactsBelongToVault(s.db, participantIDs, vaultID); err != nil {
		return nil, err
	}
	if err := validateLifeEventTypeBelongsToVault(s.db, req.LifeEventTypeID, vaultID); err != nil {
		return nil, err
	}

	timelineEvent := models.TimelineEvent{
		VaultID:   vaultID,
		StartedAt: req.HappenedAt,
		Label:     strPtrOrNil(req.Summary),
		Collapsed: false,
	}
	lifeEvent := lifeEventFromCreateRequest(req)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&timelineEvent).Error; err != nil {
			return err
		}
		lifeEvent.TimelineEventID = timelineEvent.ID
		if err := tx.Create(&lifeEvent).Error; err != nil {
			return err
		}
		if err := replaceLifeEventParticipantsLocked(tx, lifeEvent.ID, participantIDs); err != nil {
			return err
		}
		return syncTimelineParticipantsToLifeEvents(tx, timelineEvent.ID)
	})
	if err != nil {
		return nil, err
	}

	if err := s.db.Preload("Participants").Preload("LifeEvents.Participants").First(&timelineEvent, timelineEvent.ID).Error; err != nil {
		return nil, err
	}
	resp := toTimelineEventResponse(&timelineEvent)
	return &resp, nil
}

func (s *LifeEventService) UpdateDashboardLifeEvent(vaultID string, lifeEventID uint, req dto.UpdateLifeEventRequest) (*dto.LifeEventResponse, error) {
	var le models.LifeEvent
	if err := s.db.Joins("JOIN timeline_events ON timeline_events.id = life_events.timeline_event_id").
		Where("life_events.id = ? AND timeline_events.vault_id = ?", lifeEventID, vaultID).
		First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeEventNotFound
		}
		return nil, err
	}

	participantIDs := []string(nil)
	if req.Participants != nil {
		participantIDs = dedupeContactIDs(req.Participants)
		if err := validateContactsBelongToVault(s.db, participantIDs, vaultID); err != nil {
			return nil, err
		}
	}
	if req.LifeEventTypeID != 0 {
		if err := validateLifeEventTypeBelongsToVault(s.db, req.LifeEventTypeID, vaultID); err != nil {
			return nil, err
		}
	}

	applyUpdateLifeEventRequest(&le, req)
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&le).Error; err != nil {
			return err
		}
		if req.Participants != nil {
			if err := replaceLifeEventParticipantsLocked(tx, le.ID, participantIDs); err != nil {
				return err
			}
		}
		return syncTimelineParticipantsToLifeEvents(tx, le.TimelineEventID)
	}); err != nil {
		return nil, err
	}
	if err := s.db.Preload("Participants").First(&le, le.ID).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventResponse(&le)
	return &resp, nil
}

func (s *LifeEventService) DeleteDashboardLifeEvent(vaultID string, lifeEventID uint) error {
	var le models.LifeEvent
	if err := s.db.Joins("JOIN timeline_events ON timeline_events.id = life_events.timeline_event_id").
		Where("life_events.id = ? AND timeline_events.vault_id = ?", lifeEventID, vaultID).
		First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeEventNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("life_event_id = ?", le.ID).Delete(&models.LifeEventParticipant{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&le).Error; err != nil {
			return err
		}
		var remaining int64
		if err := tx.Model(&models.LifeEvent{}).Where("timeline_event_id = ?", le.TimelineEventID).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			if err := tx.Where("timeline_event_id = ?", le.TimelineEventID).Delete(&models.TimelineEventParticipant{}).Error; err != nil {
				return err
			}
			return tx.Delete(&models.TimelineEvent{}, le.TimelineEventID).Error
		}
		return syncTimelineParticipantsToLifeEvents(tx, le.TimelineEventID)
	})
}

func (s *LifeEventService) CreateTimelineEvent(contactID, vaultID string, req dto.CreateTimelineEventRequest) (*dto.TimelineEventResponse, error) {
	participantIDs := mergeContactIDs([]string{contactID}, req.Participants)
	if err := validateContactsBelongToVault(s.db, participantIDs, vaultID); err != nil {
		return nil, err
	}

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
		return replaceTimelineEventParticipantsLocked(tx, event.ID, participantIDs)
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "TimelineEvent"
		s.feedRecorder.Record(contactID, "", ActionLifeEventCreated, "Created a life event", &event.ID, &entityType)
	}

	if err := s.db.Preload("Participants").First(&event, event.ID).Error; err != nil {
		return nil, err
	}
	resp := toTimelineEventResponse(&event)
	return &resp, nil
}

func (s *LifeEventService) AddLifeEvent(contactID string, timelineEventID uint, vaultID string, req dto.CreateLifeEventRequest) (*dto.LifeEventResponse, error) {
	var te models.TimelineEvent
	if err := s.db.Where("id = ? AND vault_id = ?", timelineEventID, vaultID).First(&te).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimelineEventNotFound
		}
		return nil, err
	}

	timelineParticipantIDs, err := timelineEventParticipantIDs(s.db, timelineEventID)
	if err != nil {
		return nil, err
	}
	requiredParticipantIDs := mergeContactIDs([]string{contactID}, timelineParticipantIDs)
	participantIDs := mergeContactIDs(requiredParticipantIDs, req.Participants)
	if err := validateContactsBelongToVault(s.db, participantIDs, vaultID); err != nil {
		return nil, err
	}

	le := lifeEventFromCreateRequest(req)
	le.TimelineEventID = timelineEventID
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&le).Error; err != nil {
			return err
		}
		return replaceLifeEventParticipantsLocked(tx, le.ID, participantIDs)
	}); err != nil {
		return nil, err
	}
	if err := s.db.Preload("Participants").First(&le, le.ID).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventResponse(&le)
	return &resp, nil
}

func (s *LifeEventService) UpdateLifeEvent(contactID string, timelineEventID, lifeEventID uint, vaultID string, req dto.UpdateLifeEventRequest) (*dto.LifeEventResponse, error) {
	var te models.TimelineEvent
	if err := s.db.Where("id = ? AND vault_id = ?", timelineEventID, vaultID).First(&te).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimelineEventNotFound
		}
		return nil, err
	}

	var le models.LifeEvent
	if err := s.db.Where("id = ? AND timeline_event_id = ?", lifeEventID, timelineEventID).First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeEventNotFound
		}
		return nil, err
	}

	timelineParticipantIDs, err := timelineEventParticipantIDs(s.db, timelineEventID)
	if err != nil {
		return nil, err
	}
	requiredParticipantIDs := mergeContactIDs([]string{contactID}, timelineParticipantIDs)
	participantIDs := requiredParticipantIDs
	if req.Participants != nil {
		participantIDs = mergeContactIDs(requiredParticipantIDs, req.Participants)
	} else {
		existingParticipantIDs, err := lifeEventParticipantIDs(s.db, lifeEventID)
		if err != nil {
			return nil, err
		}
		participantIDs = mergeContactIDs(requiredParticipantIDs, existingParticipantIDs)
	}
	if err := validateContactsBelongToVault(s.db, participantIDs, vaultID); err != nil {
		return nil, err
	}

	applyUpdateLifeEventRequest(&le, req)

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&le).Error; err != nil {
			return err
		}
		return replaceLifeEventParticipantsLocked(tx, le.ID, participantIDs)
	}); err != nil {
		return nil, err
	}
	if err := s.db.Preload("Participants").First(&le, le.ID).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventResponse(&le)
	return &resp, nil
}

func (s *LifeEventService) DeleteTimelineEvent(id uint, vaultID string) error {
	var te models.TimelineEvent
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&te).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimelineEventNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("life_event_id IN (?)", tx.Model(&models.LifeEvent{}).Select("id").Where("timeline_event_id = ?", id)).Delete(&models.LifeEventParticipant{}).Error; err != nil {
			return err
		}
		if err := tx.Where("timeline_event_id = ?", id).Delete(&models.LifeEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("timeline_event_id = ?", id).Delete(&models.TimelineEventParticipant{}).Error; err != nil {
			return err
		}
		return tx.Delete(&te).Error
	})
}

func (s *LifeEventService) DeleteLifeEvent(timelineEventID, lifeEventID uint, vaultID string) error {
	var te models.TimelineEvent
	// Authorize through the route vault; numeric timeline IDs are guessable across vaults.
	if err := s.db.Where("id = ? AND vault_id = ?", timelineEventID, vaultID).First(&te).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimelineEventNotFound
		}
		return err
	}

	var le models.LifeEvent
	if err := s.db.Where("id = ? AND timeline_event_id = ?", lifeEventID, timelineEventID).First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeEventNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("life_event_id = ?", lifeEventID).Delete(&models.LifeEventParticipant{}).Error; err != nil {
			return err
		}
		return tx.Delete(&le).Error
	})
}

func (s *LifeEventService) ToggleTimelineEvent(id uint, vaultID string) error {
	var te models.TimelineEvent
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&te).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimelineEventNotFound
		}
		return err
	}
	te.Collapsed = !te.Collapsed
	return s.db.Save(&te).Error
}

func (s *LifeEventService) ToggleLifeEvent(timelineEventID, lifeEventID uint, vaultID string) error {
	var te models.TimelineEvent
	if err := s.db.Where("id = ? AND vault_id = ?", timelineEventID, vaultID).First(&te).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimelineEventNotFound
		}
		return err
	}

	var le models.LifeEvent
	if err := s.db.Where("id = ? AND timeline_event_id = ?", lifeEventID, timelineEventID).First(&le).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeEventNotFound
		}
		return err
	}
	le.Collapsed = !le.Collapsed
	return s.db.Save(&le).Error
}

func lifeEventFromCreateRequest(req dto.CreateLifeEventRequest) models.LifeEvent {
	le := models.LifeEvent{
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
		EmotionID:         req.EmotionID,
	}
	applyTimeCalendarFields(&le.CalendarType, &le.OriginalDay, &le.OriginalMonth, &le.OriginalYear,
		&le.HappenedAt, req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	return le
}

func applyUpdateLifeEventRequest(le *models.LifeEvent, req dto.UpdateLifeEventRequest) {
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
	le.EmotionID = req.EmotionID
	applyTimeCalendarFields(&le.CalendarType, &le.OriginalDay, &le.OriginalMonth, &le.OriginalYear,
		&le.HappenedAt, req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
}

func validateLifeEventTypeBelongsToVault(db *gorm.DB, lifeEventTypeID uint, vaultID string) error {
	var count int64
	if err := db.Model(&models.LifeEventType{}).
		Joins("JOIN life_event_categories ON life_event_categories.id = life_event_types.life_event_category_id").
		Where("life_event_types.id = ? AND life_event_categories.vault_id = ?", lifeEventTypeID, vaultID).
		Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return ErrLifeEventNotFound
	}
	return nil
}

func syncTimelineParticipantsToLifeEvents(tx *gorm.DB, timelineEventID uint) error {
	var contactIDs []string
	if err := tx.Model(&models.LifeEventParticipant{}).
		Joins("JOIN life_events ON life_events.id = life_event_participants.life_event_id").
		Where("life_events.timeline_event_id = ?", timelineEventID).
		Pluck("DISTINCT life_event_participants.contact_id", &contactIDs).Error; err != nil {
		return err
	}
	return replaceTimelineEventParticipantsLocked(tx, timelineEventID, contactIDs)
}

func toTimelineEventResponse(e *models.TimelineEvent) dto.TimelineEventResponse {
	resp := dto.TimelineEventResponse{
		ID:           e.ID,
		VaultID:      e.VaultID,
		StartedAt:    e.StartedAt,
		Label:        ptrToStr(e.Label),
		Collapsed:    e.Collapsed,
		CreatedAt:    e.CreatedAt,
		Participants: contactRefs(e.Participants),
		UpdatedAt:    e.UpdatedAt,
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
		EmotionID:         le.EmotionID,
		HappenedAt:        le.HappenedAt,
		CalendarType:      le.CalendarType,
		OriginalDay:       le.OriginalDay,
		OriginalMonth:     le.OriginalMonth,
		OriginalYear:      le.OriginalYear,
		Collapsed:         le.Collapsed,
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
		Participants:      contactRefs(le.Participants),
		CreatedAt:         le.CreatedAt,
		UpdatedAt:         le.UpdatedAt,
	}
}

func timelineIDsForContact(db *gorm.DB, contactID string) ([]uint, error) {
	timelineIDSet := make(map[uint]struct{})

	var directTimelineIDs []uint
	if err := db.Model(&models.TimelineEventParticipant{}).
		Where("contact_id = ?", contactID).
		Pluck("timeline_event_id", &directTimelineIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range directTimelineIDs {
		timelineIDSet[id] = struct{}{}
	}

	var lifeEventTimelineIDs []uint
	if err := db.Model(&models.LifeEvent{}).
		Joins("JOIN life_event_participants ON life_event_participants.life_event_id = life_events.id").
		Where("life_event_participants.contact_id = ?", contactID).
		Pluck("DISTINCT life_events.timeline_event_id", &lifeEventTimelineIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range lifeEventTimelineIDs {
		timelineIDSet[id] = struct{}{}
	}

	timelineIDs := make([]uint, 0, len(timelineIDSet))
	for id := range timelineIDSet {
		timelineIDs = append(timelineIDs, id)
	}
	return timelineIDs, nil
}

func timelineEventParticipantIDs(db *gorm.DB, timelineEventID uint) ([]string, error) {
	var ids []string
	if err := db.Model(&models.TimelineEventParticipant{}).
		Where("timeline_event_id = ?", timelineEventID).
		Pluck("contact_id", &ids).Error; err != nil {
		return nil, err
	}
	return dedupeContactIDs(ids), nil
}

func lifeEventParticipantIDs(db *gorm.DB, lifeEventID uint) ([]string, error) {
	var ids []string
	if err := db.Model(&models.LifeEventParticipant{}).
		Where("life_event_id = ?", lifeEventID).
		Pluck("contact_id", &ids).Error; err != nil {
		return nil, err
	}
	return dedupeContactIDs(ids), nil
}

func replaceTimelineEventParticipants(tx *gorm.DB, timelineEventID uint, contactIDs []string) error {
	if err := tx.Where("timeline_event_id = ?", timelineEventID).Delete(&models.TimelineEventParticipant{}).Error; err != nil {
		return err
	}
	ids := dedupeContactIDs(contactIDs)
	if len(ids) == 0 {
		return nil
	}
	rows := make([]models.TimelineEventParticipant, 0, len(ids))
	for _, contactID := range ids {
		rows = append(rows, models.TimelineEventParticipant{
			ContactID:       contactID,
			TimelineEventID: timelineEventID,
		})
	}
	return tx.Create(&rows).Error
}

func replaceTimelineEventParticipantsLocked(tx *gorm.DB, timelineEventID uint, contactIDs []string) error {
	var lockTarget models.TimelineEvent
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ?", timelineEventID).
		First(&lockTarget).Error; err != nil {
		return err
	}
	return replaceTimelineEventParticipants(tx, timelineEventID, contactIDs)
}

func replaceLifeEventParticipants(tx *gorm.DB, lifeEventID uint, contactIDs []string) error {
	if err := tx.Where("life_event_id = ?", lifeEventID).Delete(&models.LifeEventParticipant{}).Error; err != nil {
		return err
	}
	ids := dedupeContactIDs(contactIDs)
	if len(ids) == 0 {
		return nil
	}
	rows := make([]models.LifeEventParticipant, 0, len(ids))
	for _, contactID := range ids {
		rows = append(rows, models.LifeEventParticipant{
			ContactID:   contactID,
			LifeEventID: lifeEventID,
		})
	}
	return tx.Create(&rows).Error
}

func replaceLifeEventParticipantsLocked(tx *gorm.DB, lifeEventID uint, contactIDs []string) error {
	var lockTarget models.LifeEvent
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ?", lifeEventID).
		First(&lockTarget).Error; err != nil {
		return err
	}
	return replaceLifeEventParticipants(tx, lifeEventID, contactIDs)
}

func mergeContactIDs(required, additional []string) []string {
	merged := make([]string, 0, len(required)+len(additional))
	merged = append(merged, required...)
	merged = append(merged, additional...)
	return dedupeContactIDs(merged)
}

func dedupeContactIDs(contactIDs []string) []string {
	seen := make(map[string]struct{}, len(contactIDs))
	result := make([]string, 0, len(contactIDs))
	for _, contactID := range contactIDs {
		if contactID == "" {
			continue
		}
		if _, exists := seen[contactID]; exists {
			continue
		}
		seen[contactID] = struct{}{}
		result = append(result, contactID)
	}
	return result
}

func contactRefs(contacts []models.Contact) []dto.TaskContactRef {
	refs := make([]dto.TaskContactRef, 0, len(contacts))
	for i := range contacts {
		refs = append(refs, dto.TaskContactRef{
			ID:   contacts[i].ID,
			Name: utils.FormatContactName("%first_name% %last_name%", &contacts[i], contacts[i].ID),
		})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Name == refs[j].Name {
			return refs[i].ID < refs[j].ID
		}
		return refs[i].Name < refs[j].Name
	})
	return refs
}
