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

func (s *ContactAvatarService) UpdateAvatar(contactID, vaultID, userID string, fileID uint) (*dto.ContactResponse, error) {
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
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactAvatarService) DeleteAvatar(contactID, vaultID, userID string) (*dto.ContactResponse, error) {
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
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
