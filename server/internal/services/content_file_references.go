package services

import (
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func syncContentFileReferences(tx *gorm.DB, vaultID, ownerType string, ownerID uint, content, format string) error {
	fileIDs := []uint(nil)
	if markdown.NormalizeFormat(format) == markdown.FormatMarkdown {
		fileIDs = markdown.ExtractFileIDs(content)
		if len(fileIDs) > 0 {
			lockResult := tx.Exec("UPDATE files SET updated_at = updated_at WHERE vault_id = ? AND id IN ?", vaultID, fileIDs)
			if lockResult.Error != nil {
				return lockResult.Error
			}
			if lockResult.RowsAffected != int64(len(fileIDs)) {
				return ErrFileNotFound
			}
		}
	}
	if err := tx.Where("vault_id = ? AND owner_type = ? AND owner_id = ?", vaultID, ownerType, ownerID).
		Delete(&models.ContentFileReference{}).Error; err != nil {
		return err
	}
	if len(fileIDs) == 0 {
		return nil
	}
	rows := make([]models.ContentFileReference, len(fileIDs))
	for index, fileID := range fileIDs {
		rows[index] = models.ContentFileReference{
			VaultID: vaultID, FileID: fileID, OwnerType: ownerType, OwnerID: ownerID,
		}
	}
	return tx.Create(&rows).Error
}
