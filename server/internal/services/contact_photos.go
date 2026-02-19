package services

import (
	"errors"
	"math"
	"os"
	"path/filepath"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

func (s *VaultFileService) ListContactPhotos(contactID, vaultID string, page, perPage int) ([]dto.VaultFileResponse, response.Meta, error) {
	query := s.db.Where("ufileable_id = ? AND type = ? AND vault_id = ?", contactID, "photo", vaultID)

	var total int64
	if err := query.Model(&models.File{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
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

func (s *VaultFileService) GetContactPhoto(fileID uint, contactID, vaultID string) (*dto.VaultFileResponse, error) {
	var file models.File
	if err := s.db.Where("id = ? AND ufileable_id = ? AND type = ? AND vault_id = ?",
		fileID, contactID, "photo", vaultID).First(&file).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	resp := toVaultFileResponse(&file)
	return &resp, nil
}

func (s *VaultFileService) DeleteContactPhoto(fileID uint, contactID, vaultID string) error {
	var file models.File
	if err := s.db.Where("id = ? AND ufileable_id = ? AND type = ? AND vault_id = ?",
		fileID, contactID, "photo", vaultID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileNotFound
		}
		return err
	}

	destPath := filepath.Join(s.uploadDir, file.UUID)
	os.Remove(destPath)

	return s.db.Delete(&file).Error
}

func (s *VaultFileService) ListContactDocuments(contactID, vaultID string, page, perPage int) ([]dto.VaultFileResponse, response.Meta, error) {
	query := s.db.Where("ufileable_id = ? AND type = ? AND vault_id = ?", contactID, "document", vaultID)

	var total int64
	if err := query.Model(&models.File{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
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

func (s *VaultFileService) DeleteContactDocument(fileID uint, contactID, vaultID string) error {
	var file models.File
	if err := s.db.Where("id = ? AND ufileable_id = ? AND type = ? AND vault_id = ?",
		fileID, contactID, "document", vaultID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileNotFound
		}
		return err
	}

	destPath := filepath.Join(s.uploadDir, file.UUID)
	os.Remove(destPath)

	return s.db.Delete(&file).Error
}
