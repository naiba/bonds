package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrCallReasonTypeNotFound = errors.New("call reason type not found")
var ErrCallReasonNotFound = errors.New("call reason not found")

type CallReasonService struct {
	db *gorm.DB
}

func NewCallReasonService(db *gorm.DB) *CallReasonService {
	return &CallReasonService{db: db}
}

func (s *CallReasonService) List(accountID string, callReasonTypeID uint) ([]dto.CallReasonResponse, error) {
	var crt models.CallReasonType
	if err := s.db.Where("id = ? AND account_id = ?", callReasonTypeID, accountID).First(&crt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCallReasonTypeNotFound
		}
		return nil, err
	}
	var reasons []models.CallReason
	if err := s.db.Where("call_reason_type_id = ?", callReasonTypeID).Order("id ASC").Find(&reasons).Error; err != nil {
		return nil, err
	}
	result := make([]dto.CallReasonResponse, len(reasons))
	for i, r := range reasons {
		result[i] = toCallReasonResponse(&r)
	}
	return result, nil
}

func (s *CallReasonService) Create(accountID string, callReasonTypeID uint, req dto.CreateCallReasonRequest) (*dto.CallReasonResponse, error) {
	var crt models.CallReasonType
	if err := s.db.Where("id = ? AND account_id = ?", callReasonTypeID, accountID).First(&crt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCallReasonTypeNotFound
		}
		return nil, err
	}
	cr := models.CallReason{
		CallReasonTypeID: callReasonTypeID,
		Label:            strPtrOrNil(req.Label),
	}
	if err := s.db.Create(&cr).Error; err != nil {
		return nil, err
	}
	resp := toCallReasonResponse(&cr)
	return &resp, nil
}

func (s *CallReasonService) Update(accountID string, callReasonTypeID uint, reasonID uint, req dto.UpdateCallReasonRequest) (*dto.CallReasonResponse, error) {
	var crt models.CallReasonType
	if err := s.db.Where("id = ? AND account_id = ?", callReasonTypeID, accountID).First(&crt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCallReasonTypeNotFound
		}
		return nil, err
	}
	var cr models.CallReason
	if err := s.db.Where("id = ? AND call_reason_type_id = ?", reasonID, callReasonTypeID).First(&cr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCallReasonNotFound
		}
		return nil, err
	}
	cr.Label = strPtrOrNil(req.Label)
	if err := s.db.Save(&cr).Error; err != nil {
		return nil, err
	}
	resp := toCallReasonResponse(&cr)
	return &resp, nil
}

func (s *CallReasonService) Delete(accountID string, callReasonTypeID uint, reasonID uint) error {
	var crt models.CallReasonType
	if err := s.db.Where("id = ? AND account_id = ?", callReasonTypeID, accountID).First(&crt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCallReasonTypeNotFound
		}
		return err
	}
	result := s.db.Where("id = ? AND call_reason_type_id = ?", reasonID, callReasonTypeID).Delete(&models.CallReason{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCallReasonNotFound
	}
	return nil
}

func toCallReasonResponse(cr *models.CallReason) dto.CallReasonResponse {
	return dto.CallReasonResponse{
		ID:               cr.ID,
		CallReasonTypeID: cr.CallReasonTypeID,
		Label:            ptrToStr(cr.Label),
		CreatedAt:        cr.CreatedAt,
		UpdatedAt:        cr.UpdatedAt,
	}
}
