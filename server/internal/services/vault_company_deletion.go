package services

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// A cascade must join company -> contact -> job ordering before deleting any
// children. Starting at the job rows deadlocks with company deletion and also
// lets a job-only deletion invert the contact lock held by DAV field updates.
// Include tombstones and cross-vault company references that the cascade later
// detaches, without loading or copying their profiles.
func lockVaultCompanyOwnersForDeletion(tx *gorm.DB, vaultID string) error {
	var companyIDs []uint
	if err := tx.Model(&models.Company{}).
		Clauses(clause.Locking{Strength: companyWriteLockStrength}).
		Where("vault_id = ?", vaultID).Order("id ASC").Pluck("id", &companyIDs).Error; err != nil {
		return err
	}
	jobs := tx.Model(&models.ContactCompany{}).Select("contact_id").Where("company_id IN ?", companyIDs)
	var contactIDs []string
	return tx.Unscoped().Model(&models.Contact{}).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("vault_id = ? OR company_id IN ? OR id IN (?)", vaultID, companyIDs, jobs).
		Order("id ASC").Pluck("id", &contactIDs).Error
}
