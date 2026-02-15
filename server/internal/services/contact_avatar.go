package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type ContactAvatarService struct {
	db *gorm.DB
}

func NewContactAvatarService(db *gorm.DB) *ContactAvatarService {
	return &ContactAvatarService{db: db}
}

func (s *ContactAvatarService) UpdateAvatar(contactID, vaultID string, fileID uint) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.FileID = &fileID
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}

func (s *ContactAvatarService) DeleteAvatar(contactID, vaultID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.FileID = nil
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}
