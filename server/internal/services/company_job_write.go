package services

import (
	"errors"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// lockCompanyForJobWrite must run before locking contacts or jobs and stay held
// until the association commits. Otherwise rename/delete can discover recipients
// before a new employee commits and leave that employee's remote card stale.
// Recheck ownership under the lock: a company may be deleted while we wait.
func lockCompanyForJobWrite(tx *gorm.DB, companyID uint, vaultID string) error {
	var company models.Company
	err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id = ? AND vault_id = ?", companyID, vaultID).First(&company).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCompanyNotFound
	}
	return err
}

func createContactJobRecord(db *gorm.DB, job *models.ContactCompany, contactID, vaultID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockCompanyForJobWrite(tx, job.CompanyID, vaultID); err != nil {
			return err
		}
		return createContactRecord(tx, job, contactID, vaultID)
	})
}
