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
	if err := s.db.Where("id = ? AND vault_id = ? AND NOT (can_be_deleted = ? AND listed = ?)", contactID, currentVaultID, false, false).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var sourceVault models.Vault
	if err := s.db.Where("id = ?", currentVaultID).First(&sourceVault).Error; err != nil {
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
	if targetVault.AccountID != sourceVault.AccountID {
		return nil, ErrVaultForbidden
	}
	if err := NewVaultService(s.db).CheckUserVaultAccess(userID, targetVaultID, models.PermissionEditor); err != nil {
		return nil, err
	}

	contact.VaultID = targetVaultID
	if contact.FirstMetThroughContactID != nil {
		if err := validateContactBelongsToVault(s.db, *contact.FirstMetThroughContactID, targetVaultID); err != nil {
			// Moving across vaults must not retain a source-vault introducer pointer,
			// otherwise contact responses can leak the introducer's name and ID.
			contact.FirstMetThroughContactID = nil
		}
	}
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, targetVaultID); err != nil {
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
