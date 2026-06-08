package services

import (
	"errors"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrGiftNotFound = errors.New("gift not found")
var ErrGiftNameRequired = errors.New("gift name required")
var ErrGiftOccasionNotFound = errors.New("gift occasion not found")
var ErrGiftStateNotFound = errors.New("gift state not found")

type GiftService struct {
	db *gorm.DB
}

func NewGiftService(db *gorm.DB) *GiftService {
	return &GiftService{db: db}
}

func (s *GiftService) List(contactID, vaultID string) ([]dto.GiftResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var gifts []models.Gift
	if err := s.db.Preload("GiftOccasion").Preload("GiftState").Where("contact_id = ?", contactID).Order("created_at DESC").Find(&gifts).Error; err != nil {
		return nil, err
	}
	result := make([]dto.GiftResponse, len(gifts))
	for i, gift := range gifts {
		result[i] = toGiftResponse(&gift)
	}
	return result, nil
}

func (s *GiftService) Create(contactID, vaultID string, req dto.CreateGiftRequest) (*dto.GiftResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	occasion, state, err := s.validateGiftRequest(vaultID, req.Name, req.GiftOccasionID, req.GiftStateID)
	if err != nil {
		return nil, err
	}
	occasionID := req.GiftOccasionID
	stateID := req.GiftStateID
	gift := models.Gift{
		ContactID:      contactID,
		Type:           req.Type,
		Name:           req.Name,
		Description:    strPtrOrNil(req.Description),
		EstimatedPrice: req.EstimatedPrice,
		CurrencyID:     req.CurrencyID,
		GiftOccasionID: &occasionID,
		GiftStateID:    &stateID,
		StatusDate:     req.StatusDate,
		ReceivedAt:     req.ReceivedAt,
		GivenAt:        req.GivenAt,
		BoughtAt:       req.BoughtAt,
	}
	if err := s.db.Create(&gift).Error; err != nil {
		return nil, err
	}
	gift.GiftOccasion = occasion
	gift.GiftState = state
	resp := toGiftResponse(&gift)
	return &resp, nil
}

func (s *GiftService) Update(id uint, contactID, vaultID string, req dto.UpdateGiftRequest) (*dto.GiftResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var gift models.Gift
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&gift).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGiftNotFound
		}
		return nil, err
	}
	occasion, state, err := s.validateGiftRequest(vaultID, req.Name, req.GiftOccasionID, req.GiftStateID)
	if err != nil {
		return nil, err
	}
	occasionID := req.GiftOccasionID
	stateID := req.GiftStateID
	gift.Type = req.Type
	gift.Name = req.Name
	gift.Description = strPtrOrNil(req.Description)
	gift.EstimatedPrice = req.EstimatedPrice
	gift.CurrencyID = req.CurrencyID
	gift.GiftOccasionID = &occasionID
	gift.GiftStateID = &stateID
	gift.StatusDate = req.StatusDate
	gift.ReceivedAt = req.ReceivedAt
	gift.GivenAt = req.GivenAt
	gift.BoughtAt = req.BoughtAt
	if err := s.db.Save(&gift).Error; err != nil {
		return nil, err
	}
	gift.GiftOccasion = occasion
	gift.GiftState = state
	resp := toGiftResponse(&gift)
	return &resp, nil
}

func (s *GiftService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.Gift{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGiftNotFound
	}
	return nil
}

func (s *GiftService) validateGiftRequest(vaultID, name string, occasionID, stateID uint) (*models.GiftOccasion, *models.GiftState, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil, ErrGiftNameRequired
	}
	accountID, err := s.accountIDForVault(vaultID)
	if err != nil {
		return nil, nil, err
	}
	occasion, err := s.giftOccasionForAccount(accountID, occasionID)
	if err != nil {
		return nil, nil, err
	}
	state, err := s.giftStateForAccount(accountID, stateID)
	if err != nil {
		return nil, nil, err
	}
	return occasion, state, nil
}

func (s *GiftService) accountIDForVault(vaultID string) (string, error) {
	var vault models.Vault
	if err := s.db.Select("account_id").Where("id = ?", vaultID).First(&vault).Error; err != nil {
		return "", err
	}
	return vault.AccountID, nil
}

func (s *GiftService) giftOccasionForAccount(accountID string, occasionID uint) (*models.GiftOccasion, error) {
	if occasionID == 0 {
		return nil, ErrGiftOccasionNotFound
	}
	var occasion models.GiftOccasion
	if err := s.db.Where("id = ? AND account_id = ?", occasionID, accountID).First(&occasion).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGiftOccasionNotFound
		}
		return nil, err
	}
	return &occasion, nil
}

func (s *GiftService) giftStateForAccount(accountID string, stateID uint) (*models.GiftState, error) {
	if stateID == 0 {
		return nil, ErrGiftStateNotFound
	}
	var state models.GiftState
	if err := s.db.Where("id = ? AND account_id = ?", stateID, accountID).First(&state).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGiftStateNotFound
		}
		return nil, err
	}
	return &state, nil
}

func toGiftResponse(gift *models.Gift) dto.GiftResponse {
	occasionLabel := ""
	if gift.GiftOccasion != nil {
		occasionLabel = ptrToStr(gift.GiftOccasion.Label)
	}
	stateLabel := ""
	if gift.GiftState != nil {
		stateLabel = ptrToStr(gift.GiftState.Label)
	}
	return dto.GiftResponse{
		ID:                gift.ID,
		ContactID:         gift.ContactID,
		Name:              gift.Name,
		Type:              gift.Type,
		Description:       ptrToStr(gift.Description),
		EstimatedPrice:    gift.EstimatedPrice,
		CurrencyID:        gift.CurrencyID,
		GiftOccasionID:    gift.GiftOccasionID,
		GiftOccasionLabel: occasionLabel,
		GiftStateID:       gift.GiftStateID,
		GiftStateLabel:    stateLabel,
		StatusDate:        gift.StatusDate,
		ReceivedAt:        gift.ReceivedAt,
		GivenAt:           gift.GivenAt,
		BoughtAt:          gift.BoughtAt,
		CreatedAt:         gift.CreatedAt,
		UpdatedAt:         gift.UpdatedAt,
	}
}
