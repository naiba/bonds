package services

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Post edits and deletion lock journals before posts. Hold the same parents
// before reading sections: an editor can replace sections or remove mentions
// without locking the old contact. Contact locks alone do not protect the body.
func lockContactMergePostSections(tx *gorm.DB, vaultID string, sourceIDs []string) ([]models.PostSection, error) {
	postIDs := tx.Model(&models.ContactPost{}).Select("post_id").Where("contact_id IN ?", sourceIDs)
	var candidates []models.Post
	if err := tx.Where("id IN (?)", postIDs).Find(&candidates).Error; err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	journalIDs := make([]uint, 0, len(candidates))
	for _, post := range candidates {
		journalIDs = append(journalIDs, post.JournalID)
	}
	var journals []models.Journal
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id IN ?", journalIDs).Order("id ASC").Find(&journals).Error; err != nil {
		return nil, err
	}
	lockedJournals := make(map[uint]bool, len(journals))
	for _, journal := range journals {
		// Journal mentions only support contacts in the journal's vault. Do not
		// turn a legacy foreign association into authority to rewrite its text.
		if journal.VaultID != vaultID {
			return nil, ErrVaultForbidden
		}
		lockedJournals[journal.ID] = true
	}
	var posts []models.Post
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id IN (?)", postIDs).Order("id ASC").Find(&posts).Error; err != nil {
		return nil, err
	}
	lockedPostIDs := make([]uint, 0, len(posts))
	for _, post := range posts {
		if !lockedJournals[post.JournalID] {
			return nil, ErrContactMergeReviewChanged
		}
		lockedPostIDs = append(lockedPostIDs, post.ID)
	}
	var sections []models.PostSection
	if len(lockedPostIDs) > 0 {
		if err := tx.Where("post_id IN ?", lockedPostIDs).Order("id ASC").Find(&sections).Error; err != nil {
			return nil, err
		}
	}
	return sections, nil
}

func mergeContactPostMentions(tx *gorm.DB, sections []models.PostSection, sourceIDs []string, targetID string) error {
	sources := make(map[string]bool, len(sourceIDs))
	for _, id := range sourceIDs {
		sources[strings.ToLower(id)] = true
	}
	for _, section := range sections {
		content := ptrToStr(section.Content)
		var body strings.Builder
		start := 0
		// Change only the ID capture of the existing mention grammar. Keeping
		// display names and every other byte preserves prose, escapes, formatting
		// and file references; a global UUID replacement would corrupt plain text.
		for _, match := range contactMentionPattern.FindAllStringSubmatchIndex(content, -1) {
			if sources[strings.ToLower(content[match[2]:match[3]])] {
				body.WriteString(content[start:match[2]])
				body.WriteString(targetID)
				start = match[3]
			}
		}
		if start == 0 {
			continue
		}
		body.WriteString(content[start:])
		// The association and its authoritative inline reference must commit
		// together, otherwise even a title edit fails after the source is deleted.
		if err := tx.Model(&models.PostSection{}).Where("id = ? AND post_id = ?", section.ID, section.PostID).Update("content", body.String()).Error; err != nil {
			return err
		}
	}
	return nil
}
