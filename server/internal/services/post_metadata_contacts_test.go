package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestMetadataOnlyPostUpdatePreservesMoveBoundary(t *testing.T) {
	f := setupContactMergePost(t)
	body := f.body + " and @[Retained friend](contact:" + f.target + ")"
	if _, err := f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Walk", Sections: []dto.PostSectionInput{{Position: 1, Content: body, ContentFormat: "markdown"}}}); err != nil {
		t.Fatal(err)
	}
	other, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Other space"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactMoveService(f.svc.db).Move(f.source, f.vault, other.ID, f.user); err != nil {
		t.Fatal(err)
	}
	writtenAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	updated, err := f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Metadata only", WrittenAt: writtenAt, UpdateLastContacted: true})
	if err != nil {
		t.Fatalf("metadata-only update rejected after moving contact: %v", err)
	}
	if updated.Title != "Metadata only" || len(updated.Sections) != 1 || updated.Sections[0].Content != body || len(updated.Contacts) != 1 || updated.Contacts[0].ID != f.target {
		t.Fatalf("wrong partial update: %+v", updated)
	}
	var target, source models.Contact
	if err := f.svc.db.First(&target, "id = ?", f.target).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.svc.db.First(&source, "id = ?", f.source).Error; err != nil {
		t.Fatal(err)
	}
	if target.LastTalkedTo == nil || !target.LastTalkedTo.Equal(writtenAt) || source.LastTalkedTo != nil {
		t.Fatalf("wrong contact timestamps: target=%v source=%v", target.LastTalkedTo, source.LastTalkedTo)
	}
	// Explicit association/body submissions must not inherit this tolerance for
	// untouched historical text. Reject them without partially saving metadata.
	for _, request := range []dto.UpdatePostRequest{
		{Title: "Invalid body", Sections: []dto.PostSectionInput{{Position: 1, Content: body, ContentFormat: "markdown"}}},
		{Title: "Invalid empty association update", ContactIDs: []string{}},
		{Title: "Invalid replacement association", ContactIDs: []string{f.target}},
	} {
		request.WrittenAt = writtenAt.AddDate(0, 1, 0)
		request.UpdateLastContacted = true
		if _, err := f.posts.Update(f.post.ID, f.journal, f.vault, request); !errors.Is(err, ErrContactNotFound) {
			t.Fatalf("explicit submission accepted invalid mention: %v", err)
		}
	}
	preserved, err := f.posts.Get(f.post.ID, f.journal, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if preserved.Title != updated.Title || !preserved.WrittenAt.Equal(writtenAt) || len(preserved.Sections) != 1 || preserved.Sections[0] != updated.Sections[0] || len(preserved.Contacts) != 1 || preserved.Contacts[0].ID != f.target {
		t.Fatalf("rejected submission partially changed post: %+v", preserved)
	}
	// No eligible contacts is also a valid metadata update, not authority to
	// recreate the associations that moving the contacts deliberately removed.
	if _, err := NewContactMoveService(f.svc.db).Move(f.target, f.vault, other.ID, f.user); err != nil {
		t.Fatal(err)
	}
	updated, err = f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "No local contacts", WrittenAt: writtenAt.AddDate(0, 2, 0), UpdateLastContacted: true})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "No local contacts" || len(updated.Contacts) != 0 || len(updated.Sections) != 1 || updated.Sections[0] != preserved.Sections[0] {
		t.Fatalf("metadata update changed historical text or links: %+v", updated)
	}
	if err := f.svc.db.First(&target, "id = ?", f.target).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.svc.db.First(&source, "id = ?", f.source).Error; err != nil {
		t.Fatal(err)
	}
	if target.LastTalkedTo == nil || !target.LastTalkedTo.Equal(writtenAt) || source.LastTalkedTo != nil || target.VaultID != other.ID || source.VaultID != other.ID {
		t.Fatalf("metadata update wrote foreign contacts: target=%+v source=%+v", target.LastTalkedTo, source.LastTalkedTo)
	}
}
