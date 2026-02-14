package services

import (
	"errors"
	"math"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

var ErrCallNotFound = errors.New("call not found")

type CallService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewCallService(db *gorm.DB) *CallService {
	return &CallService{db: db}
}

func (s *CallService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *CallService) List(contactID string, page, perPage int) ([]dto.CallResponse, response.Meta, error) {
	query := s.db.Where("contact_id = ?", contactID)

	var total int64
	if err := query.Model(&models.Call{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	offset := (page - 1) * perPage

	var calls []models.Call
	if err := query.Offset(offset).Limit(perPage).Order("called_at DESC").Find(&calls).Error; err != nil {
		return nil, response.Meta{}, err
	}

	result := make([]dto.CallResponse, len(calls))
	for i, c := range calls {
		result[i] = toCallResponse(&c)
	}

	meta := response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}

	return result, meta, nil
}

func (s *CallService) Create(contactID, authorID string, req dto.CreateCallRequest) (*dto.CallResponse, error) {
	answered := true
	if req.Answered != nil {
		answered = *req.Answered
	}
	call := models.Call{
		ContactID:    contactID,
		AuthorID:     strPtrOrNil(authorID),
		AuthorName:   "User",
		CalledAt:     req.CalledAt,
		Type:         req.Type,
		WhoInitiated: req.WhoInitiated,
		Description:  strPtrOrNil(req.Description),
		Duration:     req.Duration,
		Answered:     answered,
		CallReasonID: req.CallReasonID,
	}
	if err := s.db.Create(&call).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "Call"
		s.feedRecorder.Record(contactID, authorID, ActionCallLogged, "Logged a call", &call.ID, &entityType)
	}

	resp := toCallResponse(&call)
	return &resp, nil
}

func (s *CallService) Update(id uint, contactID string, req dto.UpdateCallRequest) (*dto.CallResponse, error) {
	var call models.Call
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&call).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCallNotFound
		}
		return nil, err
	}
	call.CalledAt = req.CalledAt
	call.Type = req.Type
	call.WhoInitiated = req.WhoInitiated
	call.Description = strPtrOrNil(req.Description)
	call.Duration = req.Duration
	call.CallReasonID = req.CallReasonID
	if req.Answered != nil {
		call.Answered = *req.Answered
	}
	if err := s.db.Save(&call).Error; err != nil {
		return nil, err
	}
	resp := toCallResponse(&call)
	return &resp, nil
}

func (s *CallService) Delete(id uint, contactID string) error {
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.Call{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCallNotFound
	}
	return nil
}

func toCallResponse(c *models.Call) dto.CallResponse {
	return dto.CallResponse{
		ID:           c.ID,
		ContactID:    c.ContactID,
		AuthorID:     ptrToStr(c.AuthorID),
		AuthorName:   c.AuthorName,
		CallReasonID: c.CallReasonID,
		CalledAt:     c.CalledAt,
		Duration:     c.Duration,
		Type:         c.Type,
		Description:  ptrToStr(c.Description),
		Answered:     c.Answered,
		WhoInitiated: c.WhoInitiated,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
