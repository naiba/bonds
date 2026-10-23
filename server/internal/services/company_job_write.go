package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Company coordination must allow the KEY SHARE lock taken by real foreign-key
// checks. FOR UPDATE would deadlock with a merge holding contacts and inheriting
// CompanyID while a company writer waits for those contacts. NO KEY UPDATE still
// serializes company/job writers; the actual DELETE later acquires its key lock.
const companyWriteLockStrength = "NO KEY UPDATE"

// lockCompanyForJobWrite must run before locking contacts or jobs and stay held
// until the association commits. Otherwise rename/delete can discover recipients
// before a new employee commits and leave that employee's remote card stale.
// Recheck ownership under the lock: a company may be deleted while we wait.
func lockCompanyForJobWrite(tx *gorm.DB, companyID uint, vaultID string) error {
	var company models.Company
	err := tx.Clauses(clause.Locking{Strength: companyWriteLockStrength}).Where("id = ? AND vault_id = ?", companyID, vaultID).First(&company).Error
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
		if err := createContactRecord(tx, job, contactID, vaultID); err != nil {
			return err
		}
		// Capture the response while the company and contact are still locked.
		// A post-commit reload can lose the row to a successful company deletion
		// and report failure even though this job was committed and pushed.
		return tx.Preload("Company").First(job, job.ID).Error
	})
}
