package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrImportantDateNotFound = errors.New("important date not found")

type ImportantDateService struct {
	db *gorm.DB
}

func NewImportantDateService(db *gorm.DB) *ImportantDateService {
	return &ImportantDateService{db: db}
}

func (s *ImportantDateService) List(contactID string) ([]dto.ImportantDateResponse, error) {
	var dates []models.ContactImportantDate
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&dates).Error; err != nil {
		return nil, err
	}
	result := make([]dto.ImportantDateResponse, len(dates))
	for i, d := range dates {
		result[i] = toImportantDateResponse(&d)
	}
	return result, nil
}

func (s *ImportantDateService) Create(contactID string, req dto.CreateImportantDateRequest) (*dto.ImportantDateResponse, error) {
	date := models.ContactImportantDate{
		ContactID:                  contactID,
		Label:                      req.Label,
		Day:                        req.Day,
		Month:                      req.Month,
		Year:                       req.Year,
		ContactImportantDateTypeID: req.ContactImportantDateTypeID,
	}
	if err := s.db.Create(&date).Error; err != nil {
		return nil, err
	}
	resp := toImportantDateResponse(&date)
	return &resp, nil
}

func (s *ImportantDateService) Update(id uint, contactID string, req dto.UpdateImportantDateRequest) (*dto.ImportantDateResponse, error) {
	var date models.ContactImportantDate
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&date).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImportantDateNotFound
		}
		return nil, err
	}
	date.Label = req.Label
	date.Day = req.Day
	date.Month = req.Month
	date.Year = req.Year
	date.ContactImportantDateTypeID = req.ContactImportantDateTypeID
	if err := s.db.Save(&date).Error; err != nil {
		return nil, err
	}
	resp := toImportantDateResponse(&date)
	return &resp, nil
}

func (s *ImportantDateService) Delete(id uint, contactID string) error {
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactImportantDate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrImportantDateNotFound
	}
	return nil
}

func toImportantDateResponse(d *models.ContactImportantDate) dto.ImportantDateResponse {
	return dto.ImportantDateResponse{
		ID:                         d.ID,
		ContactID:                  d.ContactID,
		Label:                      d.Label,
		Day:                        d.Day,
		Month:                      d.Month,
		Year:                       d.Year,
		ContactImportantDateTypeID: d.ContactImportantDateTypeID,
		CreatedAt:                  d.CreatedAt,
		UpdatedAt:                  d.UpdatedAt,
	}
}
