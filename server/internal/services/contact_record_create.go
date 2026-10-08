package services

import "gorm.io/gorm"

// createContactRecord rechecks ownership in the insertion transaction. A contact
// validated earlier may already have been merged: locking the live parent keeps
// the new record either in the merge's input or out of the deleted source.
func createContactRecord(db *gorm.DB, record any, contactID, vaultID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}
