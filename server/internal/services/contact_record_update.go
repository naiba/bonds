package services

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// updateContactRecord rejects edits loaded before a merge or move. Save would
// restore stale ownership (or upsert a deleted pivot); update only editable fields
// while the record still belongs to the live contact in the requested vault.
func updateContactRecord(db *gorm.DB, record any, contactID, vaultID string, notFound error, columns ...string) error {
	liveContact := db.Session(&gorm.Session{NewDB: true}).Model(&models.Contact{}).Select("1").Where("id = ? AND vault_id = ?", contactID, vaultID)
	result := db.Model(record).Where("contact_id = ? AND EXISTS (?)", contactID, liveContact).
		Select(append(columns, "updated_at")).Omit(clause.Associations).Updates(record)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return notFound
	}
	return nil
}

// A profile edit must never recreate a source soft-deleted by a concurrent merge.
// Limit writes to the requested fields so an unrelated edit cannot undo fields
// adopted by the surviving contact either.
func updateContactProfile(db *gorm.DB, contact *models.Contact, vaultID string, columns ...string) error {
	result := db.Model(contact).Where("vault_id = ?", vaultID).
		Select(append(columns, "updated_at")).Omit(clause.Associations).Updates(contact)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrContactNotFound
	}
	return nil
}
