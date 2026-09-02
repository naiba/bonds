package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// SetSliceOfLife sets the SliceOfLifeID on a post.
func (s *PostService) SetSliceOfLife(postID uint, journalID uint, vaultID string, sliceOfLifeID uint) error {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return err
	}
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	var slice models.SliceOfLife
	if err := s.db.Where("id = ? AND journal_id = ?", sliceOfLifeID, journalID).First(&slice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSliceOfLifeNotFound
		}
		return err
	}
	return s.db.Model(&post).Update("slice_of_life_id", sliceOfLifeID).Error
}

// ClearSliceOfLife clears the SliceOfLifeID on a post.
func (s *PostService) ClearSliceOfLife(postID uint, journalID uint, vaultID string) error {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return err
	}
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	return s.db.Model(&post).Update("slice_of_life_id", nil).Error
}
