package services

import "github.com/naiba/bonds/internal/models"

type contactMergeRelationshipChange struct {
	original  models.Relationship
	contactID string
	relatedID string
	remove    bool
}

// The same plan drives authorization, preview counts and writes. An unchanged
// target reference needs no external Editor access, but deleting even an already
// duplicated target reference does. Inspecting only source IDs misses that case.
func planContactMergeRelationships(relationships []models.Relationship, targetID string, sourceIDs []string) []contactMergeRelationshipChange {
	sources := make(map[string]bool, len(sourceIDs))
	for _, id := range sourceIDs {
		sources[id] = true
	}
	type relationshipKey struct {
		from, to string
		typeID   uint
	}
	seen := map[relationshipKey]bool{}
	changes := []contactMergeRelationshipChange{}
	for _, row := range relationships {
		from, to := row.ContactID, row.RelatedContactID
		if sources[from] {
			from = targetID
		}
		if sources[to] {
			to = targetID
		}
		key := relationshipKey{from, to, row.RelationshipTypeID}
		remove := from == to || seen[key]
		if remove || from != row.ContactID || to != row.RelatedContactID {
			changes = append(changes, contactMergeRelationshipChange{original: row, contactID: from, relatedID: to, remove: remove})
		}
		seen[key] = true
	}
	return changes
}
