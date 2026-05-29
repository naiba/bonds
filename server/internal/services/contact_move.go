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
	if err := s.db.Joins("Vault").Where("contacts.id = ? AND contacts.vault_id = ? AND contacts.can_be_deleted = ?", contactID, currentVaultID, true).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var targetUserVault models.UserVault
	if err := s.db.Joins("JOIN vaults ON vaults.id = user_vault.vault_id").Where("user_vault.user_id = ? AND user_vault.vault_id = ? AND user_vault.permission <= ? AND vaults.account_id = ?", userID, targetVaultID, models.PermissionEditor, contact.Vault.AccountID).First(&targetUserVault).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTargetVaultNotFound
		}
		return nil, err
	}

	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	contact.VaultID = targetVaultID
	resp := toContactResponse(&contact, false)
	return &resp, nil
}
