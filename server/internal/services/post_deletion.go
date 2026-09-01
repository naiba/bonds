package services

import (
	"os"
	"path/filepath"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *PostService) SetUploadDir(uploadDir string) {
	s.uploadDir = uploadDir
}

func (s *PostService) Delete(id uint, journalID uint, vaultID string) error {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return err
	}
	if err := validatePostBelongsToJournal(s.db, id, journalID); err != nil {
		return err
	}

	var files []models.File
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockPostJournal(tx, journalID, vaultID); err != nil {
			return err
		}
		post, err := lockJournalPost(tx, id, journalID)
		if err != nil {
			return err
		}
		postFiles, err := deletePostDependents(tx, vaultID, []uint{post.ID})
		if err != nil {
			return err
		}
		files = postFiles
		return tx.Delete(post).Error
	}); err != nil {
		return err
	}
	removeCommittedPostFiles(s.uploadDir, files)
	return nil
}

func deletePostDependents(tx *gorm.DB, vaultID string, postIDs []uint) ([]models.File, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}

	const postFileType = "Post"
	var files []models.File
	if err := tx.Where("vault_id = ? AND fileable_type = ? AND fileable_id IN ?", vaultID, postFileType, postIDs).Find(&files).Error; err != nil {
		return nil, err
	}
	if len(files) > 0 {
		fileIDs := make([]uint, len(files))
		for index := range files {
			fileIDs[index] = files[index].ID
		}
		if err := tx.Model(&models.SliceOfLife{}).Where("file_cover_image_id IN ?", fileIDs).Update("file_cover_image_id", nil).Error; err != nil {
			return nil, err
		}
		if err := tx.Unscoped().Model(&models.Contact{}).Where("file_id IN ?", fileIDs).Update("file_id", nil).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(&models.QuickFact{}).Where("file_id IN ?", fileIDs).Update("file_id", nil).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Where("post_id IN ?", postIDs).Delete(&models.PostMetric{}).Error; err != nil {
		return nil, err
	}
	var sectionIDs []uint
	if err := tx.Model(&models.PostSection{}).Where("post_id IN ?", postIDs).Pluck("id", &sectionIDs).Error; err != nil {
		return nil, err
	}
	if len(sectionIDs) > 0 {
		if err := tx.Where("owner_type = ? AND owner_id IN ?", models.ContentOwnerPostSection, sectionIDs).
			Delete(&models.ContentFileReference{}).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Where("post_id IN ?", postIDs).Delete(&models.PostSection{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id IN ?", postIDs).Delete(&models.PostTag{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id IN ?", postIDs).Delete(&models.ContactPost{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("vault_id = ? AND fileable_type = ? AND fileable_id IN ?", vaultID, postFileType, postIDs).Delete(&models.File{}).Error; err != nil {
		return nil, err
	}
	return files, nil
}

func removeCommittedPostFiles(uploadDir string, files []models.File) {
	if uploadDir == "" {
		return
	}
	// Physical cleanup happens only after the database transaction commits.
	for _, file := range files {
		_ = os.Remove(filepath.Join(uploadDir, file.UUID))
	}
}
