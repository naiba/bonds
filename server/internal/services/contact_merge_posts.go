package services

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contactMergePostPlan struct {
	PostIDs  []uint
	Sections []models.PostSection
}

// Legacy edits and moving a contact away and back can remove the pivot while
// retaining an authoritative inline mention. Discover both representations;
// scope text reads to this vault, and parse candidates with the editor grammar
// so a bare UUID or a malformed marker never authorizes rewriting a post.
func discoverContactMergePostIDs(tx *gorm.DB, vaultID string, sourceIDs []string) ([]uint, error) {
	var associated []uint
	if err := tx.Model(&models.ContactPost{}).Where("contact_id IN ?", sourceIDs).Pluck("post_id", &associated).Error; err != nil {
		return nil, err
	}
	ids := make(map[uint]bool, len(associated))
	for _, id := range associated {
		ids[id] = true
	}
	sources := contactMergeSourceSet(sourceIDs)
	var sections []models.PostSection
	// Do not prefilter by literal UUID: Markdown destinations may encode any of
	// its characters. Keep the vault boundary and let the shared parser decide.
	if err := tx.Model(&models.PostSection{}).Select("post_sections.post_id", "post_sections.content", "post_sections.content_format").
		Joins("JOIN posts ON posts.id = post_sections.post_id").
		Joins("JOIN journals ON journals.id = posts.journal_id").
		Where("journals.vault_id = ?", vaultID).Find(&sections).Error; err != nil {
		return nil, err
	}
	for _, section := range sections {
		for _, id := range contactMentionIDs(ptrToStr(section.Content), section.ContentFormat) {
			if sources[id] {
				ids[section.PostID] = true
				break
			}
		}
	}
	result := make([]uint, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	return result, nil
}

// Post edits and deletion lock journals before posts. Hold the same parents
// before reading sections: an editor can remove mentions without locking the
// old contact. Rediscover after waiting so deleted/replaced bodies are not used.
func lockContactMergePosts(tx *gorm.DB, vaultID string, sourceIDs []string) (contactMergePostPlan, error) {
	plan := contactMergePostPlan{}
	postIDs, err := discoverContactMergePostIDs(tx, vaultID, sourceIDs)
	if err != nil {
		return plan, err
	}
	var candidates []models.Post
	if err := tx.Where("id IN ?", postIDs).Find(&candidates).Error; err != nil {
		return plan, err
	}
	if len(candidates) == 0 {
		return plan, nil
	}
	journalIDs := make([]uint, 0, len(candidates))
	for _, post := range candidates {
		journalIDs = append(journalIDs, post.JournalID)
	}
	var journals []models.Journal
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id IN ?", journalIDs).Order("id ASC").Find(&journals).Error; err != nil {
		return plan, err
	}
	lockedJournals := make(map[uint]bool, len(journals))
	for _, journal := range journals {
		// Journal mentions only support contacts in the journal's vault. A legacy
		// foreign association cannot authorize rewriting text outside that vault.
		if journal.VaultID != vaultID {
			return plan, ErrVaultForbidden
		}
		lockedJournals[journal.ID] = true
	}
	postIDs, err = discoverContactMergePostIDs(tx, vaultID, sourceIDs)
	if err != nil {
		return plan, err
	}
	var posts []models.Post
	if err := tx.Where("id IN ?", postIDs).Find(&posts).Error; err != nil {
		return plan, err
	}
	for _, post := range posts {
		if !lockedJournals[post.JournalID] {
			return plan, ErrContactMergeReviewChanged
		}
	}
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id IN ?", postIDs).Order("id ASC").Find(&posts).Error; err != nil {
		return plan, err
	}
	for _, post := range posts {
		plan.PostIDs = append(plan.PostIDs, post.ID)
	}
	if len(plan.PostIDs) > 0 {
		if err := tx.Where("post_id IN ?", plan.PostIDs).Order("id ASC").Find(&plan.Sections).Error; err != nil {
			return plan, err
		}
	}
	return plan, nil
}

func mergeContactPostMentions(tx *gorm.DB, sections []models.PostSection, sourceIDs []string, targetID string) error {
	sources := make(map[string]bool, len(sourceIDs))
	for _, id := range sourceIDs {
		sources[strings.ToLower(id)] = true
	}
	linkedPosts := make(map[uint]bool)
	for _, section := range sections {
		content := ptrToStr(section.Content)
		body := redirectContactMentions(content, section.ContentFormat, sources, targetID)
		if body == content {
			continue
		}
		// The association and its authoritative inline reference must commit
		// together, otherwise even a title edit fails after the source is deleted.
		if err := tx.Model(&models.PostSection{}).Where("id = ? AND post_id = ?", section.ID, section.PostID).Update("content", body).Error; err != nil {
			return err
		}
		if !linkedPosts[section.PostID] {
			// Existing pivots have already moved in place. Repair only the missing
			// association exposed by a migrated body; keep its other links unchanged.
			association := models.ContactPost{PostID: section.PostID, ContactID: targetID}
			if err := tx.Where("post_id = ? AND contact_id = ?", section.PostID, targetID).FirstOrCreate(&association).Error; err != nil {
				return err
			}
			linkedPosts[section.PostID] = true
		}
	}
	return nil
}
