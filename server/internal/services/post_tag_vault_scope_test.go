package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

// The route only proves the caller is a member of :vault_id. The journal and
// post IDs in the path are caller supplied, so every post-tag operation has to
// tie the journal back to that vault itself.
func TestPostTag_RejectsJournalFromAnotherVault(t *testing.T) {
	svc, postID, journalID, ownerVaultID := setupPostTagTest(t)
	ownedTag, err := svc.Add(postID, journalID, ownerVaultID, dto.AddPostTagRequest{Name: "private-topic"})
	if err != nil {
		t.Fatalf("owner could not tag the post: %v", err)
	}

	foreignVault := models.Vault{AccountID: "foreign-account", Name: "Foreign Vault"}
	if err := svc.db.Create(&foreignVault).Error; err != nil {
		t.Fatalf("create foreign vault: %v", err)
	}

	if _, err := svc.List(postID, journalID, foreignVault.ID); !errors.Is(err, ErrJournalNotFound) {
		t.Errorf("List through a foreign vault: err = %v, want ErrJournalNotFound", err)
	}
	if _, err := svc.Add(postID, journalID, foreignVault.ID, dto.AddPostTagRequest{Name: "injected"}); !errors.Is(err, ErrJournalNotFound) {
		t.Errorf("Add through a foreign vault: err = %v, want ErrJournalNotFound", err)
	}
	if _, err := svc.Update(ownedTag.ID, postID, journalID, foreignVault.ID, dto.UpdatePostTagRequest{Name: "renamed"}); !errors.Is(err, ErrJournalNotFound) {
		t.Errorf("Update through a foreign vault: err = %v, want ErrJournalNotFound", err)
	}
	if err := svc.Remove(ownedTag.ID, postID, journalID, foreignVault.ID); !errors.Is(err, ErrJournalNotFound) {
		t.Errorf("Remove through a foreign vault: err = %v, want ErrJournalNotFound", err)
	}

	tags, err := svc.List(postID, journalID, ownerVaultID)
	if err != nil {
		t.Fatalf("owner List failed: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "private-topic" {
		t.Fatalf("owner tags = %+v, want the single untouched tag", tags)
	}
	var foreignTagCount int64
	if err := svc.db.Model(&models.Tag{}).Where("vault_id = ?", foreignVault.ID).Count(&foreignTagCount).Error; err != nil {
		t.Fatalf("count foreign tags: %v", err)
	}
	if foreignTagCount != 0 {
		t.Fatalf("a rejected Add still created %d tag(s) in the foreign vault", foreignTagCount)
	}
}
