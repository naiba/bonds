package services

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrFileNotFound = errors.New("file not found")

type VaultFileService struct {
	db        *gorm.DB
	uploadDir string
}

func NewVaultFileService(db *gorm.DB, uploadDir string) *VaultFileService {
	return &VaultFileService{db: db, uploadDir: uploadDir}
}

func (s *VaultFileService) UploadDir() string {
	return s.uploadDir
}

func (s *VaultFileService) List(vaultID string) ([]dto.VaultFileResponse, error) {
	var files []models.File
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}
	return result, nil
}

func (s *VaultFileService) Get(id uint) (*dto.VaultFileResponse, error) {
	var file models.File
	if err := s.db.First(&file, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	resp := toVaultFileResponse(&file)
	return &resp, nil
}

func (s *VaultFileService) Upload(vaultID string, contactID string, fileType string, filename string, mimeType string, size int64, data io.Reader) (*dto.VaultFileResponse, error) {
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

	file := models.File{
		VaultID:  vaultID,
		UUID:     fileUUID,
		Name:     filename,
		MimeType: mimeType,
		Type:     fileType,
		Size:     int(size),
	}

	if contactID != "" {
		fileableType := "Contact"
		file.FileableType = &fileableType
		file.UfileableID = &contactID
	}

	if err := s.db.Create(&file).Error; err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	resp := toVaultFileResponse(&file)
	return &resp, nil
}

func (s *VaultFileService) Delete(id uint) error {
	var file models.File
	if err := s.db.First(&file, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileNotFound
		}
		return err
	}

	destPath := filepath.Join(s.uploadDir, file.UUID)
	os.Remove(destPath)

	return s.db.Delete(&file).Error
}

func toVaultFileResponse(f *models.File) dto.VaultFileResponse {
	return dto.VaultFileResponse{
		ID:        f.ID,
		VaultID:   f.VaultID,
		UUID:      f.UUID,
		Name:      f.Name,
		MimeType:  f.MimeType,
		Type:      f.Type,
		Size:      f.Size,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}
