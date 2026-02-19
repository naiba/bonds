package services

import (
	"math"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
)

// ListByType returns files filtered by type (photo, document, avatar)
func (s *VaultFileService) ListByType(vaultID, fileType string, page, perPage int) ([]dto.VaultFileResponse, response.Meta, error) {
	query := s.db.Where("vault_id = ? AND type = ?", vaultID, fileType)

	var total int64
	if err := query.Model(&models.File{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	var files []models.File
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, response.Meta{}, err
	}
	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}

	meta := response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}
	return result, meta, nil
}
