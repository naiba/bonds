package services

import (
	"math"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

func (s *VaultFileService) ListContactPhotos(contactID, vaultID string, page, perPage int) ([]dto.VaultFileResponse, response.Meta, error) {
	query := s.db.Where("ufileable_id = ? AND type IN (?, ?, ?) AND vault_id = ? AND (fileable_type IS NULL OR fileable_type <> ?)", contactID, "photo", "avatar", "video", vaultID, "QuickFact")

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
	if err := s.db.Where("id = ? AND ufileable_id = ? AND type IN (?, ?, ?) AND vault_id = ?",
		fileID, contactID, "photo", "avatar", "video", vaultID).First(&file).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	resp := toVaultFileResponse(&file)
	return &resp, nil
}

func (s *VaultFileService) DeleteContactPhoto(fileID uint, contactID, vaultID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		file, err := lockFileForMutation(tx, fileID, vaultID)
		if err != nil {
			return err
		}
		if file.UfileableID == nil || *file.UfileableID != contactID || (file.Type != "photo" && file.Type != "avatar" && file.Type != "video") {
			return ErrFileNotFound
		}
		if err := ensureFileNotUsed(tx, file.ID, true); err != nil {
			return err
		}
		// If this file is the contact's avatar, unset the reference in the
		// same transaction so a failed physical removal preserves both rows.
		if file.Type == "avatar" {
			if err := tx.Model(&models.Contact{}).Where("id = ? AND file_id = ?", contactID, file.ID).
				Update("file_id", nil).Error; err != nil {
				return err
			}
		}
		return s.deleteFileRecord(tx, &file)
	})
}

func (s *VaultFileService) ListContactDocuments(contactID, vaultID string, page, perPage int) ([]dto.VaultFileResponse, response.Meta, error) {
	query := s.db.Where("ufileable_id = ? AND type = ? AND vault_id = ? AND (fileable_type IS NULL OR fileable_type <> ?)", contactID, "document", vaultID, "QuickFact")

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
	return s.db.Transaction(func(tx *gorm.DB) error {
		file, err := lockFileForMutation(tx, fileID, vaultID)
		if err != nil {
			return err
		}
		if file.UfileableID == nil || *file.UfileableID != contactID || file.Type != "document" {
			return ErrFileNotFound
		}
		if err := ensureFileNotUsed(tx, file.ID, true); err != nil {
			return err
		}
		return s.deleteFileRecord(tx, &file)
	})
}
