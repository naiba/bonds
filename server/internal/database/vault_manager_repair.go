package database

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// repairManagerlessVaults restores vaults affected by older versions that
// allowed the final manager to demote themselves. The oldest suitable member
// is promoted, preferring active account administrators and then active users.
// Vaults without any membership are left untouched because there is no safe
// user to grant access to automatically.
func repairManagerlessVaults(db *gorm.DB) error {
	var vaultIDs []string
	if err := db.Model(&models.Vault{}).
		Where("EXISTS (SELECT 1 FROM user_vault uv WHERE uv.vault_id = vaults.id)").
		Where("NOT EXISTS (SELECT 1 FROM user_vault uv JOIN users u ON u.id = uv.user_id WHERE uv.vault_id = vaults.id AND uv.permission = ? AND u.disabled = ?)", models.PermissionManager, false).
		Order("vaults.id ASC").
		Pluck("vaults.id", &vaultIDs).Error; err != nil {
		return err
	}

	for _, vaultID := range vaultIDs {
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("UPDATE vaults SET updated_at = updated_at WHERE id = ?", vaultID).Error; err != nil {
				return err
			}
			var managerCount int64
			if err := tx.Model(&models.UserVault{}).
				Joins("JOIN users ON users.id = user_vault.user_id").
				Where("user_vault.vault_id = ? AND user_vault.permission = ? AND users.disabled = ?", vaultID, models.PermissionManager, false).
				Count(&managerCount).Error; err != nil {
				return err
			}
			if managerCount > 0 {
				return nil
			}

			var membership models.UserVault
			err := tx.Table("user_vault").
				Select("user_vault.*").
				Joins("JOIN users ON users.id = user_vault.user_id").
				Where("user_vault.vault_id = ?", vaultID).
				Order("users.disabled ASC").
				Order("users.is_account_administrator DESC").
				Order("user_vault.created_at ASC").
				Order("user_vault.id ASC").
				First(&membership).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			return tx.Model(&models.UserVault{}).
				Where("id = ? AND vault_id = ?", membership.ID, vaultID).
				Update("permission", models.PermissionManager).Error
		}); err != nil {
			return err
		}
	}
	return nil
}
