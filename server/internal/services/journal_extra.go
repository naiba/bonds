package services

import (
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// GetByYear returns posts for a specific year in a journal
func (s *JournalService) GetByYear(journalID uint, vaultID string, year int) ([]dto.PostResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}

	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	var posts []models.Post
	if err := s.db.Where("journal_id = ? AND written_at >= ? AND written_at < ?", journalID, startOfYear, endOfYear).
		Order("written_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}

	result := make([]dto.PostResponse, len(posts))
	for i, p := range posts {
		result[i] = toPostResponse(&p)
	}
	return result, nil
}

// GetPhotos returns files associated with posts in a journal
func (s *JournalService) GetPhotos(journalID uint, vaultID string) ([]dto.VaultFileResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}

	var postIDs []uint
	if err := s.db.Model(&models.Post{}).Where("journal_id = ?", journalID).Pluck("id", &postIDs).Error; err != nil {
		return nil, err
	}

	if len(postIDs) == 0 {
		return []dto.VaultFileResponse{}, nil
	}

	fileableType := "Post"
	var files []models.File
	if err := s.db.Where("fileable_type = ? AND fileable_id IN ?", fileableType, postIDs).
		Order("created_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}

	result := make([]dto.VaultFileResponse, len(files))
	for i, f := range files {
		result[i] = toVaultFileResponse(&f)
	}
	return result, nil
}
