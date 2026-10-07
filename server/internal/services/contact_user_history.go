package services

import (
	"errors"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func lockContactHistoryOwner(tx *gorm.DB, contactID, vaultID string) error {
	var contact models.Contact
	err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Select("id").Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrContactNotFound
	}
	return err
}

func recordContactView(db *gorm.DB, contactID, userID, vaultID string) (bool, error) {
	favorite := false
	err := db.Transaction(func(tx *gorm.DB) error {
		// Serialize history writes with merges. An unlocked read followed by Save
		// can discard successful views or favorites while histories are aggregated.
		if err := lockContactHistoryOwner(tx, contactID, vaultID); err != nil {
			return err
		}
		var history models.ContactVaultUser
		err := tx.Where("contact_id = ? AND user_id = ?", contactID, userID).First(&history).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		favorite = history.IsFavorite
		return tx.Model(&history).Where("contact_id = ?", contactID).Update("number_of_views", gorm.Expr("number_of_views + 1")).Error
	})
	return favorite, err
}
