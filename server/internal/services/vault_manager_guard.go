package services

import (
	"sort"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// lockVaultMemberships serializes membership mutations for one vault across
// SQLite and PostgreSQL. Updating a column to itself leaves the vault unchanged
// while still taking the database write/row lock needed to make the
// last-manager check race-free.
func lockVaultMemberships(tx *gorm.DB, vaultID string) error {
	return tx.Exec("UPDATE vaults SET updated_at = updated_at WHERE id = ?", vaultID).Error
}

func ensureAnotherVaultManager(tx *gorm.DB, vaultID string, excludedMembershipID uint) error {
	var count int64
	if err := tx.Model(&models.UserVault{}).
		Joins("JOIN users ON users.id = user_vault.user_id").
		Where("user_vault.vault_id = ? AND user_vault.permission = ? AND user_vault.id <> ? AND users.disabled = ?", vaultID, models.PermissionManager, excludedMembershipID, false).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrLastVaultManager
	}
	return nil
}

// ensureUserIsNotSoleVaultManager protects account- and instance-level user
// deletion paths. Vault rows are locked in a stable order to avoid deadlocks
// when two users share more than one vault.
func ensureUserIsNotSoleVaultManager(tx *gorm.DB, userID string) error {
	var memberships []models.UserVault
	if err := tx.Where("user_id = ?", userID).
		Order("vault_id ASC").Find(&memberships).Error; err != nil {
		return err
	}

	vaultIDs := make([]string, 0, len(memberships))
	seen := make(map[string]struct{}, len(memberships))
	for _, membership := range memberships {
		if _, ok := seen[membership.VaultID]; ok {
			continue
		}
		seen[membership.VaultID] = struct{}{}
		vaultIDs = append(vaultIDs, membership.VaultID)
	}
	sort.Strings(vaultIDs)

	for _, vaultID := range vaultIDs {
		if err := lockVaultMemberships(tx, vaultID); err != nil {
			return err
		}
	}
	for _, membership := range memberships {
		var current models.UserVault
		if err := tx.Where("id = ? AND user_id = ? AND vault_id = ?", membership.ID, userID, membership.VaultID).
			First(&current).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		if current.Permission != models.PermissionManager {
			continue
		}
		if err := ensureAnotherVaultManager(tx, current.VaultID, current.ID); err != nil {
			return err
		}
	}
	return nil
}
