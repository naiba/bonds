package services

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// taskStatusExistsForVault returns true if the given slug matches a
// TaskStatus row for the account that owns the given vault. Empty input is
// treated as valid (it will get resolved to the default later).
func taskStatusExistsForVault(db *gorm.DB, slug, vaultID string) bool {
	if slug == "" {
		return true
	}
	var count int64
	db.Model(&models.TaskStatus{}).
		Where("slug = ? AND account_id = (SELECT account_id FROM vaults WHERE id = ?)", slug, vaultID).
		Count(&count)
	return count > 0
}

// resolveTaskStatusOrDefault returns the input slug when it's a configured
// status, otherwise the account's default status slug. Falls back to "todo"
// only when no statuses exist at all (which shouldn't happen after seed).
func resolveTaskStatusOrDefault(db *gorm.DB, input, vaultID string) string {
	if input != "" && taskStatusExistsForVault(db, input, vaultID) {
		return input
	}
	var slug string
	row := db.Raw(`
		SELECT ts.slug FROM task_statuses ts
		WHERE ts.account_id = (SELECT account_id FROM vaults WHERE id = ?)
		AND ts.is_default = ?
		ORDER BY ts.position ASC
		LIMIT 1
	`, vaultID, true).Row()
	_ = row.Scan(&slug)
	if slug != "" {
		return slug
	}
	// Final fallback: any seeded status, ordered by position.
	row = db.Raw(`
		SELECT ts.slug FROM task_statuses ts
		WHERE ts.account_id = (SELECT account_id FROM vaults WHERE id = ?)
		ORDER BY ts.position ASC
		LIMIT 1
	`, vaultID).Row()
	_ = row.Scan(&slug)
	if slug != "" {
		return slug
	}
	return "todo"
}
