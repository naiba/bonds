package services

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestPostContactOnlyUpdateHonorsStoredMentions(t *testing.T) {
	f := setupContactMergePost(t)
	updated, err := f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Association-only edit", ContactIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Contacts) != 1 || updated.Contacts[0].ID != f.source || len(updated.Sections) != 1 || updated.Sections[0].Content != f.body || updated.Sections[0].ID != f.post.Sections[0].ID {
		t.Fatalf("contact-only update diverged from stored body: %+v", updated)
	}
	updated, err = f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Explicit fallback", ContactIDs: []string{f.target}})
	if err != nil || len(updated.Contacts) != 1 || updated.Contacts[0].ID != f.source || updated.Sections[0].Content != f.body {
		t.Fatalf("fallback overrode an existing mention: %+v %v", updated, err)
	}
	updated, err = f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Removed mention", Sections: []dto.PostSectionInput{{Position: 1, Content: "A walk alone", ContentFormat: "markdown"}}, ContactIDs: []string{}})
	if err != nil || len(updated.Contacts) != 0 || updated.Sections[0].Content != "A walk alone" {
		t.Fatalf("explicit body edit failed to detach: %+v %v", updated, err)
	}
	updated, err = f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Legacy association", ContactIDs: []string{f.target}})
	if err != nil || len(updated.Contacts) != 1 || updated.Contacts[0].ID != f.target {
		t.Fatalf("legacy fallback no longer works: %+v %v", updated, err)
	}
	updated, err = f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Cleared fallback", ContactIDs: []string{}})
	if err != nil || len(updated.Contacts) != 0 || updated.Sections[0].Content != "A walk alone" {
		t.Fatalf("clearing legacy fallback failed: %+v %v", updated, err)
	}
}

func TestContactMergeFindsPostBodyWithoutAssociation(t *testing.T) {
	for _, scenario := range []string{"legacy_missing_pivot", "vault_round_trip"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupContactMergePost(t)
			if scenario == "vault_round_trip" {
				other, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Contact travel"}, "en")
				if err != nil {
					t.Fatal(err)
				}
				mover := NewContactMoveService(f.svc.db)
				if _, err := mover.Move(f.source, f.vault, other.ID, f.user); err != nil {
					t.Fatal(err)
				}
				if _, err := mover.Move(f.source, other.ID, f.vault, f.user); err != nil {
					t.Fatal(err)
				}
			} else if err := f.svc.db.Where("post_id = ?", f.post.ID).Delete(&models.ContactPost{}).Error; err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := f.svc.db.Model(&models.ContactPost{}).Where("post_id = ?", f.post.ID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("setup must have no association")
			}
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			preview, err := f.svc.PreviewContactMerge(f.vault, f.user, request)
			if err != nil {
				t.Fatal(err)
			}
			if preview.Effects["posts"] != 1 {
				t.Fatalf("body-only post absent from preview: %+v", preview.Effects)
			}
			if _, err := f.svc.MergeContacts(f.vault, f.user, request); err != nil {
				t.Fatal(err)
			}
			got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
			if err != nil {
				t.Fatal(err)
			}
			expected := strings.ReplaceAll(f.body, f.source, f.target)
			if len(got.Sections) != 1 || got.Sections[0].ID != f.post.Sections[0].ID || got.Sections[0].Content != expected || len(got.Contacts) != 1 || got.Contacts[0].ID != f.target {
				t.Fatalf("body-only merge lost reference: %+v", got)
			}
			edited, err := f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Editable after merge", Sections: []dto.PostSectionInput{{Position: 1, Content: got.Sections[0].Content, ContentFormat: "markdown"}}})
			if err != nil || edited.Title != "Editable after merge" {
				t.Fatalf("refreshed edit failed: %+v %v", edited, err)
			}
			_, err = f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Stale", Sections: []dto.PostSectionInput{{Position: 1, Content: f.body}}})
			if !errors.Is(err, ErrContactNotFound) {
				t.Fatalf("stale form unexpectedly accepted: %v", err)
			}
		})
	}
}

func TestContactMergeBodyDiscoveryPreservesScopeAndGrammar(t *testing.T) {
	f := setupContactMergePost(t)
	if err := f.svc.db.Where("post_id = ?", f.post.ID).Delete(&models.ContactPost{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.svc.db.Model(&models.PostSection{}).Where("id = ?", f.post.Sections[0].ID).Update("content", strings.ReplaceAll(f.body, f.source, strings.ToUpper(f.source))).Error; err != nil {
		t.Fatal(err)
	}
	foreign, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Private journal space"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	journal := models.Journal{VaultID: foreign.ID, Name: "Unrelated diary"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	originals := []models.PostSection{}
	for _, entry := range []struct {
		journal uint
		body    string
	}{
		{f.journal, "UUID " + f.source + " and malformed [Name](contact:" + f.source + ")"},
		{journal.ID, f.body},
	} {
		post := models.Post{JournalID: entry.journal, WrittenAt: time.Now()}
		if err := f.svc.db.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
		section := models.PostSection{PostID: post.ID, Position: 1, Content: &entry.body}
		if err := f.svc.db.Create(&section).Error; err != nil {
			t.Fatal(err)
		}
		// Compare persisted timestamps: PostgreSQL stores microsecond precision,
		// while GORM's newly created value still contains nanoseconds.
		if err := f.svc.db.First(&section, section.ID).Error; err != nil {
			t.Fatal(err)
		}
		originals = append(originals, section)
	}
	// A foreign body without a pivot is not part of this vault's merge authority.
	if err := f.svc.db.Where("vault_id = ? AND user_id = ?", foreign.ID, f.user).Delete(&models.UserVault{}).Error; err != nil {
		t.Fatal(err)
	}
	request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
	preview, err := f.svc.PreviewContactMerge(f.vault, f.user, request)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Effects["posts"] != 1 {
		t.Fatalf("discovery included foreign text or non-mentions: %+v", preview.Effects)
	}
	if _, err := f.svc.MergeContacts(f.vault, f.user, request); err != nil {
		t.Fatal(err)
	}
	for _, before := range originals {
		var after models.PostSection
		if err := f.svc.db.First(&after, before.ID).Error; err != nil {
			t.Fatal(err)
		}
		if ptrToStr(after.Content) != ptrToStr(before.Content) || !after.UpdatedAt.Equal(before.UpdatedAt) {
			t.Fatalf("unrelated text changed: before=%+v after=%+v", before, after)
		}
		var count int64
		if err := f.svc.db.Model(&models.ContactPost{}).Where("post_id = ?", before.PostID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("unrelated post acquired a contact")
		}
	}
	got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sections[0].Content != strings.ReplaceAll(f.body, f.source, f.target) || len(got.Contacts) != 1 || got.Contacts[0].ID != f.target {
		t.Fatalf("case-insensitive actual mention not migrated: %+v", got)
	}
}

func TestContactMergeMissingPostAssociationRollsBackOnRepairFailure(t *testing.T) {
	f := setupContactMergePost(t)
	if err := f.svc.db.Where("post_id = ?", f.post.ID).Delete(&models.ContactPost{}).Error; err != nil {
		t.Fatal(err)
	}
	note := models.Note{ContactID: f.source, VaultID: f.vault, Body: "Keep the source note"}
	if err := f.svc.db.Create(&note).Error; err != nil {
		t.Fatal(err)
	}
	request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
	rejected := errors.New("post association repair rejected")
	hook := "post_mention:reject_repair"
	var sawMigratedBody bool
	if err := f.svc.db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table != "contact_post" {
			return
		}
		var section models.PostSection
		if err := tx.Session(&gorm.Session{NewDB: true}).First(&section, f.post.Sections[0].ID).Error; err != nil {
			tx.AddError(err)
			return
		}
		sawMigratedBody = ptrToStr(section.Content) == strings.ReplaceAll(f.body, f.source, f.target)
		tx.AddError(rejected)
	}); err != nil {
		t.Fatal(err)
	}
	defer f.svc.db.Callback().Create().Remove(hook)
	if _, err := f.svc.MergeContacts(f.vault, f.user, request); !errors.Is(err, rejected) {
		t.Fatalf("repair failure=%v", err)
	}
	if !sawMigratedBody {
		t.Fatal("failure did not occur after body migration")
	}
	got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sections[0].Content != f.body || len(got.Contacts) != 0 {
		t.Fatalf("repair failure left partial post: %+v", got)
	}
	var source models.Contact
	if err := f.svc.db.First(&source, "id = ?", f.source).Error; err != nil {
		t.Fatal("source not restored:", err)
	}
	if err := f.svc.db.First(&note, note.ID).Error; err != nil {
		t.Fatal(err)
	}
	if note.ContactID != f.source {
		t.Fatal("note migration escaped rollback")
	}
}

func TestPostContactOnlyUpdateRechecksBodyAfterConcurrentWrite(t *testing.T) {
	for _, operation := range []string{"merge", "replace_mention"} {
		t.Run(operation, func(t *testing.T) {
			f := setupContactMergePost(t)
			if f.svc.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL contact and journal lock waits")
			}
			if err := f.svc.db.Where("post_id = ?", f.post.ID).Delete(&models.ContactPost{}).Error; err != nil {
				t.Fatal(err)
			}
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			locked, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "post_mention:pause_authoritative_writer"
			if err := f.svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				if tx.Error != nil {
					return
				}
				if _, ok := tx.Statement.Clauses["FOR"]; !ok {
					return
				}
				switch tx.Statement.Dest.(type) {
				case *models.Journal:
					if operation != "replace_mention" {
						return
					}
				case *[]models.Journal:
					if operation != "merge" {
						return
					}
				default:
					return
				}
				if paused.CompareAndSwap(false, true) {
					close(locked)
					select {
					case <-resume:
					case <-time.After(15 * time.Second):
						tx.AddError(errors.New("post contact update barrier timed out"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer f.svc.db.Callback().Query().Remove(hook)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			changed := make(chan error, 1)
			go func() {
				var err error
				if operation == "merge" {
					_, err = NewContactService(f.svc.db.WithContext(ctx)).MergeContacts(f.vault, f.user, request)
				} else {
					_, err = NewPostService(f.svc.db.WithContext(ctx)).Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Body writer", Sections: []dto.PostSectionInput{{Position: 1, Content: strings.ReplaceAll(f.body, f.source, f.target), ContentFormat: "markdown"}}})
				}
				changed <- err
			}()
			awaitMergeBoundarySignal(t, locked)
			updated := make(chan error, 1)
			writtenAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
			go func() {
				_, err := NewPostService(f.svc.db.WithContext(ctx)).Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Partial update", WrittenAt: writtenAt, ContactIDs: []string{}, UpdateLastContacted: true})
				updated <- err
			}()
			var waited bool
			if operation == "merge" {
				waited = waitForContactMergeLock(f.svc.db)
			} else {
				waited = waitForJournalMergeLock(f.svc.db)
			}
			close(resume)
			writeErr, updateErr := awaitMergeBoundaryError(t, changed), awaitMergeBoundaryError(t, updated)
			if writeErr != nil || updateErr != nil {
				t.Fatalf("writer=%v partial update=%v", writeErr, updateErr)
			}
			if !waited {
				t.Fatal("partial update did not exercise the lock wait")
			}
			got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != "Partial update" || got.Sections[0].Content != strings.ReplaceAll(f.body, f.source, f.target) || len(got.Contacts) != 1 || got.Contacts[0].ID != f.target {
				t.Fatalf("stored-body retry used stale contacts: %+v", got)
			}
			var target, source models.Contact
			if err := f.svc.db.First(&target, "id = ?", f.target).Error; err != nil {
				t.Fatal(err)
			}
			if err := f.svc.db.Unscoped().First(&source, "id = ?", f.source).Error; err != nil {
				t.Fatal(err)
			}
			if target.LastTalkedTo == nil || !target.LastTalkedTo.Equal(writtenAt) || source.LastTalkedTo != nil {
				t.Fatalf("last-contacted updated wrong identity: target=%+v source=%+v", target.LastTalkedTo, source.LastTalkedTo)
			}
			if source.DeletedAt.Valid != (operation == "merge") {
				t.Fatal("partial update changed source lifecycle")
			}
		})
	}
}
