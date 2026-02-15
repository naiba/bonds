package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

// ListByType returns files filtered by type (photo, document, avatar)
func (s *VaultFileService) ListByType(vaultID, fileType string) ([]dto.VaultFileResponse, error) {
	var files []models.File
	if err := s.db.Where("vault_id = ? AND type = ?", vaultID, fileType).Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}
	return result, nil
}
