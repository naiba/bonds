package services

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contactMergeContentPlan struct {
	Notes      []models.Note
	Activities []models.Activity
}

func contactMergeMentionCandidates(column string, sourceIDs []string) (string, []any) {
	conditions := make([]string, 0, len(sourceIDs))
	values := make([]any, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		conditions = append(conditions, "LOWER("+column+") LIKE ?")
		values = append(values, "%contact:"+strings.ToLower(id)+"%")
	}
	return "(" + strings.Join(conditions, " OR ") + ")", values
}

func contactMergeSourceSet(sourceIDs []string) map[string]bool {
	sources := make(map[string]bool, len(sourceIDs))
	for _, id := range sourceIDs {
		sources[strings.ToLower(id)] = true
	}
	return sources
}

func containsMergeSourceMention(content, format string, sources map[string]bool) bool {
	for _, id := range contactMentionIDs(content, format) {
		if sources[id] {
			return true
		}
	}
	return false
}

// Notes have both a vault and a contact owner. Do not use a textual match to
// authorize changing another vault's history (including legacy mismatched rows).
func discoverContactMergeNotes(tx *gorm.DB, vaultID string, sourceIDs []string) ([]models.Note, error) {
	condition, values := contactMergeMentionCandidates("notes.body", sourceIDs)
	var candidates []models.Note
	if err := tx.Model(&models.Note{}).Select("notes.*").
		Joins("JOIN contacts ON contacts.id = notes.contact_id").
		Where("notes.vault_id = ? AND contacts.vault_id = ? AND contacts.deleted_at IS NULL", vaultID, vaultID).
		Where(condition, values...).Order("notes.id ASC").Find(&candidates).Error; err != nil {
		return nil, err
	}
	sources := contactMergeSourceSet(sourceIDs)
	notes := []models.Note{}
	for _, note := range candidates {
		if containsMergeSourceMention(note.Body, note.BodyFormat, sources) {
			notes = append(notes, note)
		}
	}
	return notes, nil
}

func lockContactMergeContent(tx *gorm.DB, vaultID string, sourceIDs []string) (contactMergeContentPlan, []models.Activity, error) {
	plan := contactMergeContentPlan{}
	notes, err := discoverContactMergeNotes(tx, vaultID, sourceIDs)
	if err != nil {
		return plan, nil, err
	}
	plan.Notes = notes // Their parent contacts are locked before the merge review.
	// Lock all affected activity rows in the same order, before participant rows.
	// A description may mention a source even when participant_ids is explicitly empty.
	condition, values := contactMergeMentionCandidates("description", sourceIDs)
	participants := tx.Model(&models.ActivityParticipant{}).Select("activity_id").Where("contact_id IN ?", sourceIDs)
	var activities []models.Activity
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("paid_by_contact_id IN ? OR id IN (?) OR (vault_id = ? AND "+condition+")", append([]any{sourceIDs, participants, vaultID}, values...)...).
		Order("id ASC").Find(&activities).Error; err != nil {
		return plan, nil, err
	}
	sources := contactMergeSourceSet(sourceIDs)
	payers := []models.Activity{}
	for _, activity := range activities {
		if sources[ptrToStr(activity.PaidByContactID)] {
			payers = append(payers, activity)
		}
		if activity.VaultID == vaultID && containsMergeSourceMention(ptrToStr(activity.Description), activity.DescriptionFormat, sources) {
			plan.Activities = append(plan.Activities, activity)
		}
	}
	return plan, payers, nil
}

// Moving ownership alone leaves rendered links pointing at deleted contacts.
// Apply the reviewed bodies in the same transaction as associations and deletion;
// no snapshots or new associations are needed for notes or activity mentions.
func mergeContactContent(tx *gorm.DB, plan contactMergeContentPlan, sourceIDs []string, targetID string) error {
	sources := contactMergeSourceSet(sourceIDs)
	for _, note := range plan.Notes {
		if err := tx.Model(&models.Note{}).Where("id = ? AND vault_id = ?", note.ID, note.VaultID).
			Update("body", redirectContactMentions(note.Body, note.BodyFormat, sources, targetID)).Error; err != nil {
			return err
		}
	}
	for _, activity := range plan.Activities {
		if err := tx.Model(&models.Activity{}).Where("id = ? AND vault_id = ?", activity.ID, activity.VaultID).
			Update("description", redirectContactMentions(ptrToStr(activity.Description), activity.DescriptionFormat, sources, targetID)).Error; err != nil {
			return err
		}
	}
	return nil
}
