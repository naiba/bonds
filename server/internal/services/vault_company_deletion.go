package services

import (
	"errors"
	"slices"

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
	var contactIDs []string
	if err := vaultCompanyOwnerQuery(tx, vaultID, companyIDs).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Pluck("id", &contactIDs).Error; err != nil {
		return err
	}
	// A company created after discovery can be held by a job writer waiting
	// for our contacts (#297). Never lock that new company in reverse order.
	// Exclude further creation, then roll back and rediscover the entire set.
	// Try only: resolvers can already hold contacts while waiting for this lock.
	if err := tryCompanyResolution(tx, vaultID); err != nil {
		return err
	}
	// A new contact can already own a vault FK KEY SHARE before requesting
	// company resolution. Never wait for its owner lock while holding exclusion:
	// roll back first, then wait without company/contact/name locks (#297).
	if err := lockVaultCompanyDeletionOwner(tx, vaultID, true); err != nil {
		return err
	}
	var currentCompanyIDs []uint
	if err := tx.Model(&models.Company{}).Where("vault_id = ?", vaultID).Order("id ASC").Pluck("id", &currentCompanyIDs).Error; err != nil {
		return err
	}
	if !slices.Equal(companyIDs, currentCompanyIDs) {
		return errCompanyDiscoveryChanged
	}
	var currentContactIDs []string
	if err := vaultCompanyOwnerQuery(tx, vaultID, companyIDs).Pluck("id", &currentContactIDs).Error; err != nil {
		return err
	}
	if !slices.Equal(contactIDs, currentContactIDs) {
		return errCompanyDiscoveryChanged
	}
	return nil
}

func vaultCompanyOwnerQuery(tx *gorm.DB, vaultID string, companyIDs []uint) *gorm.DB {
	jobs := tx.Model(&models.ContactCompany{}).Select("contact_id").Where("company_id IN ?", companyIDs)
	return tx.Unscoped().Model(&models.Contact{}).
		Where("vault_id = ? OR company_id IN ? OR id IN (?)", vaultID, companyIDs, jobs).
		Order("id ASC")
}

// nowait is used inside the cascade; blocking acquisition is only allowed in
// the separate retry wait transaction after all cascade locks were released.
func lockVaultCompanyDeletionOwner(tx *gorm.DB, vaultID string, nowait bool) error {
	options := ""
	if nowait {
		options = "NOWAIT"
	}
	var vault models.Vault
	err := tx.Select("id").Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate, Options: options}).
		Where("id = ?", vaultID).Find(&vault).Error
	var state interface{ SQLState() string }
	if nowait && errors.As(err, &state) && state.SQLState() == "55P03" {
		return &companyCoordinationBusyError{vaultID: vaultID, vaultOwner: true}
	}
	return err
}
