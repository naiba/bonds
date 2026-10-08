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
	return contacts, locked, nil
}
