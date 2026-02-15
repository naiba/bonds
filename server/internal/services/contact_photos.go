package services

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *VaultFileService) ListContactPhotos(contactID, vaultID string) ([]dto.VaultFileResponse, error) {
	var files []models.File
	if err := s.db.Where("ufileable_id = ? AND type = ? AND vault_id = ?", contactID, "photo", vaultID).
		Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}
	return result, nil
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

func (s *VaultFileService) ListContactDocuments(contactID, vaultID string) ([]dto.VaultFileResponse, error) {
	var files []models.File
	if err := s.db.Where("ufileable_id = ? AND type = ? AND vault_id = ?", contactID, "document", vaultID).
		Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}
	return result, nil
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
