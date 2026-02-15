package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrTargetVaultNotFound = errors.New("target vault not found")

type ContactMoveService struct {
	db *gorm.DB
}

func NewContactMoveService(db *gorm.DB) *ContactMoveService {
	return &ContactMoveService{db: db}
}

func (s *ContactMoveService) Move(contactID, currentVaultID, targetVaultID, userID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, currentVaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var targetVault models.Vault
	if err := s.db.Where("id = ?", targetVaultID).First(&targetVault).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTargetVaultNotFound
		}
		return nil, err
	}

	contact.VaultID = targetVaultID
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}
