package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// UpdateDefaultTab updates the default activity tab for a vault
func (s *VaultService) UpdateDefaultTab(vaultID string, tab string) error {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVaultNotFound
		}
		return err
	}
	return s.db.Model(&vault).Update("default_activity_tab", tab).Error
}
