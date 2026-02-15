package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type CurrencyService struct {
	db *gorm.DB
}

func NewCurrencyService(db *gorm.DB) *CurrencyService {
	return &CurrencyService{db: db}
}

func (s *CurrencyService) List() ([]dto.CurrencyResponse, error) {
	var currencies []models.Currency
	if err := s.db.Order("code ASC").Find(&currencies).Error; err != nil {
		return nil, err
	}
	result := make([]dto.CurrencyResponse, len(currencies))
	for i, c := range currencies {
		result[i] = dto.CurrencyResponse{
			ID:   c.ID,
			Code: c.Code,
		}
	}
	return result, nil
}
