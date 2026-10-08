package services

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Rewrite only the ID capture shared by the editors. Names, ordinary UUIDs,
// escaping, formatting and attachment references must remain byte-for-byte intact.
func redirectContactMentions(content string, sources map[string]bool, targetID string) string {
	var body strings.Builder
	start := 0
	for _, match := range contactMentionPattern.FindAllStringSubmatchIndex(content, -1) {
		if sources[strings.ToLower(content[match[2]:match[3]])] {
			body.WriteString(content[start:match[2]])
			body.WriteString(targetID)
			start = match[3]
		}
	}
	if start == 0 {
		return content
	}
	body.WriteString(content[start:])
	return body.String()
}

// Content can mention a contact without owning it or listing it as a participant.
// Lock live mentions with the required parents in one ID order, before child
// locks, so a concurrent merge either discovers the write or rejects its stale ID.
func lockContentContacts(tx *gorm.DB, vaultID string, required []string, content string) (map[string]bool, error) {
	ids := mergeContactIDs(required, contactMentionIDs(content))
	var contacts []models.Contact
	if len(ids) > 0 {
		if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("vault_id = ? AND id IN ?", vaultID, ids).Order("id ASC").Find(&contacts).Error; err != nil {
			return nil, err
		}
	}
	live := make(map[string]bool, len(contacts))
	for _, contact := range contacts {
		live[contact.ID] = true
	}
	for _, id := range dedupeContactIDs(required) {
		if !live[id] {
			return nil, ErrContactNotFound
		}
	}
	return live, nil
}

// Existing historical mentions may refer to moved/deleted contacts. Preserve
// those on ordinary edits, but never introduce a new dead or foreign reference.
// Read previous content under its owner/row lock: a stale editor must not restore
// the source ID that a merge has just redirected.
func validateContentMentions(content, previous string, live map[string]bool) error {
	historical := map[string]bool{}
	for _, id := range contactMentionIDs(previous) {
		historical[id] = true
	}
	for _, id := range contactMentionIDs(content) {
		if !live[id] && !historical[id] {
			return ErrContactNotFound
		}
	}
	return nil
}
