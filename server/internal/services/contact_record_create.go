package services

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// createContactRecord rechecks ownership in the insertion transaction. A contact
// validated earlier may already have been merged: locking the live parent keeps
// the new record either in the merge's input or out of the deleted source.
func createContactRecord(db *gorm.DB, record any, contactID, vaultID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		// Association creation must hold the employer key through insertion, so
		// company deletion can freeze and revalidate its employee set.
		if job, ok := record.(*models.ContactCompany); ok {
			if err := lockCompanyForEmployment(tx, job.CompanyID, vaultID); err != nil {
				return err
			}
		}
		return tx.Create(record).Error
	})
}
