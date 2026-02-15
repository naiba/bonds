package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *VaultFileService) ListPostPhotos(postID uint, vaultID string) ([]dto.VaultFileResponse, error) {
	fileableType := "Post"
	var files []models.File
	if err := s.db.Where("fileable_type = ? AND fileable_id = ? AND vault_id = ?", fileableType, postID, vaultID).
		Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}
	return result, nil
}

func (s *VaultFileService) UploadPostPhoto(postID uint, vaultID string, filename string, mimeType string, size int64, data io.Reader) (*dto.VaultFileResponse, error) {
	fileUUID := uuid.New().String()

	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	destPath := filepath.Join(s.uploadDir, fileUUID)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, data); err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	fileableType := "Post"
	file := models.File{
		VaultID:      vaultID,
		UUID:         fileUUID,
		Name:         filename,
		MimeType:     mimeType,
		Type:         "photo",
		Size:         int(size),
		FileableType: &fileableType,
		FileableID:   &postID,
	}
	if err := s.db.Create(&file).Error; err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	resp := toVaultFileResponse(&file)
	return &resp, nil
}

func (s *VaultFileService) DeletePostPhoto(fileID, postID uint, vaultID string) error {
	fileableType := "Post"
	var file models.File
	if err := s.db.Where("id = ? AND fileable_type = ? AND fileable_id = ? AND vault_id = ?",
		fileID, fileableType, postID, vaultID).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrFileNotFound
		}
		return err
	}

	destPath := filepath.Join(s.uploadDir, file.UUID)
	os.Remove(destPath)

	return s.db.Delete(&file).Error
}
