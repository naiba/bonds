package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrCurrencyNotFound = errors.New("currency not found")

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

func (s *CurrencyService) Toggle(accountID string, currencyID uint) error {
	var currency models.Currency
	if err := s.db.First(&currency, currencyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCurrencyNotFound
		}
		return err
	}

	var ac models.AccountCurrency
	err := s.db.Where("account_id = ? AND currency_id = ?", accountID, currencyID).First(&ac).Error
	if err == nil {
		return s.db.Delete(&ac).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	ac = models.AccountCurrency{
		AccountID:  accountID,
		CurrencyID: currencyID,
	}
	return s.db.Create(&ac).Error
}

func (s *CurrencyService) EnableAll(accountID string) error {
	var existingIDs []uint
	if err := s.db.Model(&models.AccountCurrency{}).
		Where("account_id = ?", accountID).
		Pluck("currency_id", &existingIDs).Error; err != nil {
		return err
	}

	query := s.db.Model(&models.Currency{})
	if len(existingIDs) > 0 {
		query = query.Where("id NOT IN ?", existingIDs)
	}
	var missing []models.Currency
	if err := query.Find(&missing).Error; err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}

	items := make([]models.AccountCurrency, len(missing))
	for i, c := range missing {
		items[i] = models.AccountCurrency{CurrencyID: c.ID, AccountID: accountID}
	}
	return s.db.CreateInBatches(&items, 50).Error
}

func (s *CurrencyService) DisableAll(accountID string) error {
	return s.db.Where("account_id = ?", accountID).Delete(&models.AccountCurrency{}).Error
}
