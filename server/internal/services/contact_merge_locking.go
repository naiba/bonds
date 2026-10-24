package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Read the candidate owners without child locks, then lock every parent in one
// global order. Locking only the selection allows an incoming owner to move to
// another vault after authorization, and reverses the order used by edits.
func lockContactMergeOwners(tx *gorm.DB, vaultID string, req dto.MergeContactsRequest) ([]models.Contact, map[string]bool, error) {
	ids := append([]string{req.TargetContactID}, req.SourceContactIDs...)
	// Merging transfers jobs and may inherit CompanyID. Join the company-first
	// protocol before taking contact locks: deletion must not retain a stale
	// owner set while another merge moves those jobs to a new contact.
	var companyIDs []uint
	if err := contactMergeCompanyQuery(tx, vaultID, ids).
		Clauses(clause.Locking{Strength: companyWriteLockStrength}).Pluck("id", &companyIDs).Error; err != nil {
		return nil, nil, err
	}
	var relationships []models.Relationship
	if err := tx.Where("contact_id IN ? OR related_contact_id IN ?", ids, ids).Order("id ASC").Find(&relationships).Error; err != nil {
		return nil, nil, err
	}
	ownerIDs := append([]string(nil), ids...)
	for _, change := range planContactMergeRelationships(relationships, req.TargetContactID, req.SourceContactIDs) {
		ownerIDs = append(ownerIDs, change.original.ContactID)
	}
	var introducerOwners []string
	if err := tx.Model(&models.Contact{}).Where("first_met_through_contact_id IN ?", req.SourceContactIDs).Pluck("id", &introducerOwners).Error; err != nil {
		return nil, nil, err
	}
	ownerIDs = append(ownerIDs, introducerOwners...)
	notes, err := discoverContactMergeNotes(tx, vaultID, req.SourceContactIDs)
	if err != nil {
		return nil, nil, err
	}
	for _, note := range notes {
		ownerIDs = append(ownerIDs, note.ContactID)
	}
	var owners []models.Contact
	if err := tx.Unscoped().Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id IN ?", ownerIDs).Order("id ASC").Find(&owners).Error; err != nil {
		return nil, nil, err
	}
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}
	locked := make(map[string]bool, len(owners))
	contacts := []models.Contact{}
	for _, owner := range owners {
		locked[owner.ID] = true
		if selected[owner.ID] && owner.VaultID == vaultID && !owner.DeletedAt.Valid {
			contacts = append(contacts, owner)
		}
	}
	if len(contacts) != len(ids) {
		return nil, nil, ErrContactNotFound
	}
	// Discovery was before the parent locks. Never acquire a newly found
	// company here: that would invert the ordering with an association writer.
	var currentCompanyIDs []uint
	if err := contactMergeCompanyQuery(tx, vaultID, ids).Pluck("id", &currentCompanyIDs).Error; err != nil {
		return nil, nil, err
	}
	lockedCompanies := make(map[uint]bool, len(companyIDs))
	for _, id := range companyIDs {
		lockedCompanies[id] = true
	}
	for _, id := range currentCompanyIDs {
		if !lockedCompanies[id] {
			return nil, nil, ErrContactMergeReviewChanged
		}
	}
	return contacts, locked, nil
}

// Only company IDs are needed for coordination. Read references from the merge's
// own live contacts, including legacy fields and every job that may be moved.
func contactMergeCompanyQuery(tx *gorm.DB, vaultID string, ids []string) *gorm.DB {
	legacy := tx.Model(&models.Contact{}).Select("company_id").Where("id IN ? AND vault_id = ? AND company_id IS NOT NULL", ids, vaultID)
	owners := tx.Model(&models.Contact{}).Select("id").Where("id IN ? AND vault_id = ?", ids, vaultID)
	jobs := tx.Model(&models.ContactCompany{}).Select("company_id").Where("contact_id IN (?)", owners)
	return tx.Model(&models.Company{}).Where("id IN (?) OR id IN (?)", legacy, jobs).Order("id ASC")
}
