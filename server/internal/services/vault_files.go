package services

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrFileNotFound = errors.New("file not found")
	ErrFileInUse    = errors.New("file is in use")
)

type VaultFileService struct {
	db           *gorm.DB
	uploadDir    string
	feedRecorder *FeedRecorder
}

func NewVaultFileService(db *gorm.DB, uploadDir string) *VaultFileService {
	return &VaultFileService{db: db, uploadDir: uploadDir}
}

func (s *VaultFileService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
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

func (s *VaultFileService) Get(id uint, vaultID string) (*dto.VaultFileResponse, error) {
	var file models.File
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	resp := toVaultFileResponse(&file)
	return &resp, nil
}

// ResolvePath returns the local path for a vault file. New files always use
// UploadDir/UUID. OriginalURL is only consulted for legacy Monica imports and
// only when it resolves to a local file contained by UploadDir.
func (s *VaultFileService) ResolvePath(id uint, vaultID string) (*dto.VaultFileResponse, string, error) {
	var file models.File
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrFileNotFound
		}
		return nil, "", err
	}
	resp := toVaultFileResponse(&file)
	return &resp, s.localPath(&file), nil
}

func (s *VaultFileService) localPath(file *models.File) string {
	canonical := filepath.Join(s.uploadDir, file.UUID)
	if _, err := os.Stat(canonical); err == nil || file.OriginalURL == nil {
		return canonical
	}
	legacy, ok := safeLegacyFilePath(s.uploadDir, *file.OriginalURL)
	if !ok {
		return canonical
	}
	if _, err := os.Stat(legacy); err != nil {
		return canonical
	}
	return legacy
}

func safeLegacyFilePath(uploadDir, candidate string) (string, bool) {
	if uploadDir == "" || candidate == "" || strings.Contains(candidate, "://") {
		return "", false
	}
	root, err := filepath.Abs(uploadDir)
	if err != nil {
		return "", false
	}
	path := candidate
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err = filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return path, true
}

// MigrateLegacyPaths moves legacy Monica-imported files into the canonical
// flat storage layout. It is idempotent and leaves external OriginalURL values
// untouched.
func (s *VaultFileService) MigrateLegacyPaths() (int, error) {
	var files []models.File
	if err := s.db.Where("original_url IS NOT NULL AND original_url <> ''").Find(&files).Error; err != nil {
		return 0, err
	}
	migrated := 0
	for i := range files {
		legacy, ok := safeLegacyFilePath(s.uploadDir, *files[i].OriginalURL)
		if !ok {
			continue
		}
		canonical := filepath.Join(s.uploadDir, files[i].UUID)
		if _, err := os.Stat(canonical); err == nil {
			if err := s.db.Model(&files[i]).Update("original_url", nil).Error; err != nil {
				return migrated, err
			}
			continue
		}
		if _, err := os.Stat(legacy); err != nil {
			continue
		}
		if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
			return migrated, err
		}
		if err := os.Rename(legacy, canonical); err != nil {
			return migrated, fmt.Errorf("migrate legacy file %d: %w", files[i].ID, err)
		}
		if err := s.db.Model(&files[i]).Update("original_url", nil).Error; err != nil {
			_ = os.Rename(canonical, legacy)
			return migrated, err
		}
		migrated++
	}
	return migrated, nil
}

func (s *VaultFileService) Upload(vaultID string, contactID string, authorID string, fileType string, filename string, mimeType string, size int64, data io.Reader) (*dto.VaultFileResponse, error) {
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

	if s.feedRecorder != nil && contactID != "" {
		entityType := "File"
		s.feedRecorder.Record(contactID, authorID, ActionFileUploaded, "Uploaded "+fileType+": "+filename, &file.ID, &entityType)
	}

	resp := toVaultFileResponse(&file)
	return &resp, nil
}

func (s *VaultFileService) Delete(id uint, vaultID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		file, err := lockFileForMutation(tx, id, vaultID)
		if err != nil {
			return err
		}
		if err := ensureFileNotUsed(tx, file.ID, true); err != nil {
			return err
		}
		return s.deleteFileRecord(tx, &file)
	})
}

func assignFileToQuickFact(db *gorm.DB, fileID uint, quickFactID uint, vaultID string) error {
	quickFactType := "QuickFact"
	return db.Model(&models.File{}).Where("id = ? AND vault_id = ?", fileID, vaultID).Updates(map[string]interface{}{
		"fileable_type": quickFactType,
		"fileable_id":   quickFactID,
	}).Error
}

func ensureFileNotUsed(db *gorm.DB, fileID uint, includeQuickFacts bool) error {
	if includeQuickFacts {
		var quickFactCount int64
		if err := db.Model(&models.QuickFact{}).Where("file_id = ?", fileID).Count(&quickFactCount).Error; err != nil {
			return err
		}
		if quickFactCount > 0 {
			return ErrFileInUse
		}
	}
	var contentReferenceCount int64
	if err := db.Model(&models.ContentFileReference{}).Where("file_id = ?", fileID).Count(&contentReferenceCount).Error; err != nil {
		return err
	}
	if contentReferenceCount > 0 {
		return ErrFileInUse
	}
	return nil
}

func (s *VaultFileService) ForceDeleteFile(id uint, vaultID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		file, err := lockFileForMutation(tx, id, vaultID)
		if err != nil {
			return err
		}
		if err := ensureFileNotUsed(tx, file.ID, false); err != nil {
			return err
		}
		return s.deleteFileRecord(tx, &file)
	})
}

func lockFileForMutation(tx *gorm.DB, id uint, vaultID string) (models.File, error) {
	result := tx.Exec("UPDATE files SET updated_at = updated_at WHERE id = ? AND vault_id = ?", id, vaultID)
	if result.Error != nil {
		return models.File{}, result.Error
	}
	if result.RowsAffected != 1 {
		return models.File{}, ErrFileNotFound
	}
	var file models.File
	if err := tx.Where("id = ? AND vault_id = ?", id, vaultID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.File{}, ErrFileNotFound
		}
		return models.File{}, err
	}
	return file, nil
}

func (s *VaultFileService) deleteFileRecord(tx *gorm.DB, file *models.File) error {
	destPath := s.localPath(file)
	if err := os.Remove(destPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove file %s: %w", file.UUID, err)
	}
	return tx.Delete(file).Error
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
