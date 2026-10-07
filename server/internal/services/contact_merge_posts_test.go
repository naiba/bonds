package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// Acceptance regression: editing an existing journal entry must work after its mentioned contact is merged.
func TestContactMergeJournalMentionRemainsEditable(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alicia"})
	if err != nil {
		t.Fatal(err)
	}
	journal, err := NewJournalService(svc.db).Create(vaultID, dto.CreateJournalRequest{Name: "Synthetic journal"})
	if err != nil {
		t.Fatal(err)
	}
	posts := NewPostService(svc.db)
	marker := "@[Alicia](contact:" + source.ID + ")"
	post, err := posts.Create(journal.ID, vaultID, dto.CreatePostRequest{Title: "Walk", WrittenAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Sections: []dto.PostSectionInput{{Position: 1, ContentFormat: "markdown", Content: "Walked with " + marker}}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := posts.Update(post.ID, journal.ID, vaultID, dto.UpdatePostRequest{Title: "Walk before merge", WrittenAt: post.WrittenAt, Sections: []dto.PostSectionInput{{Position: 1, ContentFormat: "markdown", Content: "Walked with " + marker}}, ContactIDs: []string{source.ID}})
	if err != nil || before.Title != "Walk before merge" {
		t.Fatalf("control edit before merge failed: %+v %v", before, err)
	}
	t.Log("control: same journal entry editable before merge")
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	persisted, err := posts.Get(post.ID, journal.ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.Contacts) != 1 || persisted.Contacts[0].ID != target.ID {
		t.Fatalf("post association was not migrated: %+v", persisted.Contacts)
	}
	if len(persisted.Sections) != 1 {
		t.Fatalf("sections lost: %+v", persisted.Sections)
	}
	// A refreshed editor must receive the migrated marker, without altering prose.
	body := persisted.Sections[0].Content
	t.Logf("stored section after merge: %s", body)
	if body != "Walked with @[Alicia](contact:"+target.ID+")" {
		t.Fatalf("wrong migrated body: %q", body)
	}
	updated, err := posts.Update(post.ID, journal.ID, vaultID, dto.UpdatePostRequest{Title: "Evening walk", WrittenAt: post.WrittenAt, Sections: []dto.PostSectionInput{{Position: 1, ContentFormat: "markdown", Content: body}}, ContactIDs: []string{target.ID}})
	if err != nil {
		t.Fatalf("editing the refreshed merged journal entry failed: %v", err)
	}
	if updated.Title != "Evening walk" || len(updated.Contacts) != 1 || updated.Contacts[0].ID != target.ID {
		t.Fatalf("updated journal result incorrect: %+v", updated)
	}
}

func TestContactMergePostMentionsPreserveContent(t *testing.T) {
	for _, format := range []string{"plain", "markdown"} {
		t.Run(format, func(t *testing.T) {
			svc, vault, user, _ := setupContactTest(t)
			contacts := make([]models.Contact, 4)
			for i := range contacts {
				contacts[i] = models.Contact{VaultID: vault, FirstName: strPtrOrNil(fmt.Sprintf("Journal friend %d", i))}
				if err := svc.db.Create(&contacts[i]).Error; err != nil {
					t.Fatal(err)
				}
			}
			target, source, duplicate, other := contacts[0], contacts[1], contacts[2], contacts[3]
			journal := models.Journal{VaultID: vault, Name: "Synthetic diary"}
			if err := svc.db.Create(&journal).Error; err != nil {
				t.Fatal(err)
			}
			file := models.File{VaultID: vault, UUID: "journal-mention-attachment", Name: "Synthetic image", MimeType: "image/png", Type: "photo"}
			if err := svc.db.Create(&file).Error; err != nil {
				t.Fatal(err)
			}
			// Names, raw UUIDs, URLs, HTML and escaped delimiters are prose, not ID slots.
			prefix := "# Walk\nRaw " + source.ID + " [reference](https://example.test/" + source.ID + ") <span>" + source.ID + "</span>\n"
			markerName := `A\]lice\\ ` + source.ID
			body := prefix + "@[" + markerName + "](contact:" + strings.ToUpper(source.ID) + ") and @[Again](contact:" + source.ID + ")\n@[Duplicate](contact:" + duplicate.ID + ") @[Retained](contact:" + target.ID + ") @[Other](contact:" + other.ID + ")" + fmt.Sprintf("\n![Attachment](bonds-file:%d)", file.ID)
			want := prefix + "@[" + markerName + "](contact:" + target.ID + ") and @[Again](contact:" + target.ID + ")\n@[Duplicate](contact:" + target.ID + ") @[Retained](contact:" + target.ID + ") @[Other](contact:" + other.ID + ")" + fmt.Sprintf("\n![Attachment](bonds-file:%d)", file.ID)
			posts := NewPostService(svc.db)
			post, err := posts.Create(journal.ID, vault, dto.CreatePostRequest{Title: "Walk", WrittenAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Sections: []dto.PostSectionInput{
				{Position: 1, Label: "Story", Content: body, ContentFormat: format},
				{Position: 2, Label: "Untouched", Content: "A plain reference " + source.ID, ContentFormat: format},
				{Position: 3, Label: "Empty", ContentFormat: format},
			}})
			if err != nil {
				t.Fatal(err)
			}
			original := append([]dto.PostSectionResponse(nil), post.Sections...)
			var refsBefore []models.ContentFileReference
			if err := svc.db.Where("owner_id = ? AND owner_type = ?", post.Sections[0].ID, models.ContentOwnerPostSection).Find(&refsBefore).Error; err != nil {
				t.Fatal(err)
			}
			if format == "markdown" && len(refsBefore) != 1 {
				t.Fatalf("expected attachment reference: %+v", refsBefore)
			}
			request := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID, duplicate.ID)
			if _, err := svc.MergeContacts(vault, user, request); err != nil {
				t.Fatal(err)
			}
			got, err := posts.Get(post.ID, journal.ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != "Walk" || !got.WrittenAt.Equal(post.WrittenAt) || len(got.Sections) != 3 || len(got.Contacts) != 2 {
				t.Fatalf("post metadata/associations changed: %+v", got)
			}
			for i, section := range got.Sections {
				if section.ID != original[i].ID || section.Label != original[i].Label || section.Position != original[i].Position || section.ContentFormat != format {
					t.Fatalf("section identity/format changed: %+v", section)
				}
				expected := original[i].Content
				if i == 0 {
					expected = want
				}
				if section.Content != expected {
					t.Fatalf("section content=%q want=%q", section.Content, expected)
				}
			}
			for _, c := range got.Contacts {
				if c.ID != target.ID && c.ID != other.ID {
					t.Fatalf("stale contact: %+v", c)
				}
			}
			var refsAfter []models.ContentFileReference
			if err := svc.db.Where("owner_id = ? AND owner_type = ?", post.Sections[0].ID, models.ContentOwnerPostSection).Find(&refsAfter).Error; err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(refsAfter) != fmt.Sprint(refsBefore) {
				t.Fatalf("attachment references changed: before=%+v after=%+v", refsBefore, refsAfter)
			}
			// A form opened before the merge cannot restore the deleted identity.
			if _, err := posts.Update(post.ID, journal.ID, vault, dto.UpdatePostRequest{Title: "Stale edit", Sections: []dto.PostSectionInput{{Position: 1, Content: body, ContentFormat: format}}}); !errors.Is(err, ErrContactNotFound) {
				t.Fatalf("stale edit must fail: %v", err)
			}
			updated, err := posts.Update(post.ID, journal.ID, vault, dto.UpdatePostRequest{Title: "Fresh edit", Sections: []dto.PostSectionInput{{Position: 1, Label: "Story", Content: want, ContentFormat: format}}})
			if err != nil {
				t.Fatal(err)
			}
			if updated.Title != "Fresh edit" || updated.Sections[0].Content != want || len(updated.Contacts) != 2 {
				t.Fatalf("refreshed edit failed: %+v", updated)
			}
		})
	}
}

type contactMergePostFixture struct {
	svc                                  *ContactService
	posts                                *PostService
	vault, user, account, target, source string
	journal                              uint
	post                                 *dto.PostResponse
	body                                 string
}

func setupContactMergePost(t *testing.T) contactMergePostFixture {
	t.Helper()
	svc, vault, user, account := setupContactTest(t)
	target := models.Contact{VaultID: vault, FirstName: strPtrOrNil("Retained friend")}
	source := models.Contact{VaultID: vault, FirstName: strPtrOrNil("Mentioned friend")}
	for _, contact := range []*models.Contact{&target, &source} {
		if err := svc.db.Create(contact).Error; err != nil {
			t.Fatal(err)
		}
	}
	journal := models.Journal{VaultID: vault, Name: "Synthetic memories"}
	if err := svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	posts := NewPostService(svc.db)
	body := "Walked with @[Mentioned friend](contact:" + source.ID + ")"
	post, err := posts.Create(journal.ID, vault, dto.CreatePostRequest{Title: "Walk", WrittenAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Sections: []dto.PostSectionInput{{Position: 1, Label: "Story", Content: body, ContentFormat: "markdown"}}})
	if err != nil {
		t.Fatal(err)
	}
	return contactMergePostFixture{svc, posts, vault, user, account, target.ID, source.ID, journal.ID, post, body}
}

func TestContactMergePostMentionsRollback(t *testing.T) {
	for _, failure := range []string{"section_write", "source_delete"} {
		t.Run(failure, func(t *testing.T) {
			f := setupContactMergePost(t)
			note := models.Note{ContactID: f.source, VaultID: f.vault, Body: "Preserve note"}
			if err := f.svc.db.Create(&note).Error; err != nil {
				t.Fatal(err)
			}
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			rejected := errors.New("journal merge write rejected")
			hook := "journal_merge:reject_write"
			callback := func(tx *gorm.DB) {
				if (failure == "section_write" && tx.Statement.Table == "post_sections") || (failure == "source_delete" && tx.Statement.Table == "contacts") {
					tx.AddError(rejected)
				}
			}
			if failure == "section_write" {
				if err := f.svc.db.Callback().Update().Before("gorm:update").Register(hook, callback); err != nil {
					t.Fatal(err)
				}
				defer f.svc.db.Callback().Update().Remove(hook)
			} else {
				if err := f.svc.db.Callback().Delete().Before("gorm:delete").Register(hook, callback); err != nil {
					t.Fatal(err)
				}
				defer f.svc.db.Callback().Delete().Remove(hook)
			}
			if _, err := f.svc.MergeContacts(f.vault, f.user, request); !errors.Is(err, rejected) {
				t.Fatalf("expected injected failure, got %v", err)
			}
			var section models.PostSection
			if err := f.svc.db.First(&section, f.post.Sections[0].ID).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(section.Content) != f.body {
				t.Fatalf("body not rolled back: %+v", section)
			}
			var associations []models.ContactPost
			if err := f.svc.db.Where("post_id = ?", f.post.ID).Find(&associations).Error; err != nil {
				t.Fatal(err)
			}
			if len(associations) != 1 || associations[0].ContactID != f.source {
				t.Fatalf("association not rolled back: %+v", associations)
			}
			var source models.Contact
			if err := f.svc.db.First(&source, "id = ?", f.source).Error; err != nil {
				t.Fatal("source deleted after rollback:", err)
			}
			if err := f.svc.db.First(&note, note.ID).Error; err != nil {
				t.Fatal(err)
			}
			if note.ContactID != f.source || note.Body != "Preserve note" {
				t.Fatalf("note not rolled back: %+v", note)
			}
			var count int64
			if err := f.svc.db.Model(&models.ContactFeedItem{}).Where("contact_id = ?", f.target).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("merge audit escaped rollback: %d", count)
			}
		})
	}
}

func TestContactMergePostReviewRejectsChangedBody(t *testing.T) {
	f := setupContactMergePost(t)
	request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
	changed := "Evening: " + f.body
	if _, err := f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Edited", Sections: []dto.PostSectionInput{{Position: 1, Content: changed, ContentFormat: "markdown"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.MergeContacts(f.vault, f.user, request); !errors.Is(err, ErrContactMergeReviewChanged) {
		t.Fatalf("changed body must invalidate review: %v", err)
	}
	request = reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
	if _, err := f.svc.MergeContacts(f.vault, f.user, request); err != nil {
		t.Fatal(err)
	}
	got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Edited" || got.Sections[0].Content != strings.ReplaceAll(changed, f.source, f.target) {
		t.Fatalf("edit was lost: %+v", got)
	}
}

func TestContactMergePostVaultBoundary(t *testing.T) {
	for _, sourceReference := range []bool{true, false} {
		t.Run(fmt.Sprint(sourceReference), func(t *testing.T) {
			f := setupContactMergePost(t)
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			foreign, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Separate journal vault"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			journal := models.Journal{VaultID: foreign.ID, Name: "Foreign journal"}
			if err := f.svc.db.Create(&journal).Error; err != nil {
				t.Fatal(err)
			}
			post := models.Post{JournalID: journal.ID, Title: strPtrOrNil("Private post"), WrittenAt: time.Now()}
			if err := f.svc.db.Create(&post).Error; err != nil {
				t.Fatal(err)
			}
			id := f.target
			if sourceReference {
				id = f.source
			}
			section := models.PostSection{PostID: post.ID, Position: 1, Content: strPtrOrNil("@[Private](contact:" + id + ")")}
			if err := f.svc.db.Create(&section).Error; err != nil {
				t.Fatal(err)
			}
			if err := f.svc.db.Create(&models.ContactPost{ContactID: id, PostID: post.ID}).Error; err != nil {
				t.Fatal(err)
			}
			// Even an editor cannot create cross-vault post mentions through merging.
			_, err = f.svc.PreviewContactMerge(f.vault, f.user, request)
			if sourceReference && !errors.Is(err, ErrVaultForbidden) {
				t.Fatalf("foreign source preview=%v", err)
			}
			if !sourceReference && err != nil {
				t.Fatal(err)
			}
			_, err = f.svc.MergeContacts(f.vault, f.user, request)
			if sourceReference && !errors.Is(err, ErrVaultForbidden) {
				t.Fatalf("foreign source merge=%v", err)
			}
			if !sourceReference && err != nil {
				t.Fatal("unchanged external target reference should not block:", err)
			}
			var stored models.PostSection
			if err := f.svc.db.First(&stored, section.ID).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(stored.Content) != ptrToStr(section.Content) {
				t.Fatal("foreign journal was rewritten")
			}
			if sourceReference {
				var source models.Contact
				if err := f.svc.db.First(&source, "id = ?", f.source).Error; err != nil {
					t.Fatal("rejected merge deleted source:", err)
				}
			}
		})
	}
}

func TestContactMergePostWaitsForConcurrentMutation(t *testing.T) {
	for _, operation := range []string{"title_only", "remove_mention", "delete"} {
		t.Run(operation, func(t *testing.T) {
			f := setupContactMergePost(t)
			if f.svc.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL journal row locking")
			}
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			locked, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "journal_merge:pause_mutation"
			if err := f.svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				if tx.Error != nil {
					return
				}
				if _, ok := tx.Statement.Dest.(*models.Journal); !ok {
					return
				}
				if _, ok := tx.Statement.Clauses["FOR"]; !ok {
					return
				}
				if paused.CompareAndSwap(false, true) {
					close(locked)
					select {
					case <-resume:
					case <-time.After(15 * time.Second):
						tx.AddError(errors.New("journal mutation barrier timed out"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer f.svc.db.Callback().Query().Remove(hook)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			edited := make(chan error, 1)
			go func() {
				posts := NewPostService(f.svc.db.WithContext(ctx))
				if operation == "delete" {
					edited <- posts.Delete(f.post.ID, f.journal, f.vault)
					return
				}
				req := dto.UpdatePostRequest{Title: "Concurrent edit"}
				if operation == "remove_mention" {
					req.Sections = []dto.PostSectionInput{{Position: 1, Content: "A walk alone", ContentFormat: "markdown"}}
				}
				_, err := posts.Update(f.post.ID, f.journal, f.vault, req)
				edited <- err
			}()
			awaitMergeBoundarySignal(t, locked)
			merged := make(chan error, 1)
			go func() {
				_, err := NewContactService(f.svc.db.WithContext(ctx)).MergeContacts(f.vault, f.user, request)
				merged <- err
			}()
			waited := waitForJournalMergeLock(f.svc.db)
			close(resume)
			editErr, mergeErr := awaitMergeBoundaryError(t, edited), awaitMergeBoundaryError(t, merged)
			if editErr != nil {
				t.Fatal(editErr)
			}
			if !waited {
				t.Fatal("merge did not wait for journal mutation lock")
			}
			if operation == "title_only" {
				if mergeErr != nil {
					t.Fatal(mergeErr)
				}
			} else {
				if !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
					t.Fatalf("changed post must require review: %v", mergeErr)
				}
				request = reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
				if _, err := f.svc.MergeContacts(f.vault, f.user, request); err != nil {
					t.Fatal(err)
				}
			}
			got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
			if operation == "delete" {
				if !errors.Is(err, ErrPostNotFound) {
					t.Fatalf("deleted post was resurrected: %+v %v", got, err)
				}
				var count int64
				if err := f.svc.db.Model(&models.PostSection{}).Where("post_id = ?", f.post.ID).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatalf("deleted sections returned: %d", count)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != "Concurrent edit" {
				t.Fatalf("concurrent title lost: %q", got.Title)
			}
			if operation == "remove_mention" {
				if got.Sections[0].Content != "A walk alone" || len(got.Contacts) != 0 {
					t.Fatalf("removed mention restored: %+v", got)
				}
			} else if got.Sections[0].Content != strings.ReplaceAll(f.body, f.source, f.target) || len(got.Contacts) != 1 || got.Contacts[0].ID != f.target {
				t.Fatalf("title edit and merge diverged: %+v", got)
			}
		})
	}
}

func TestContactMergePostRejectsConcurrentStaleEditor(t *testing.T) {
	f := setupContactMergePost(t)
	if f.svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL contact and journal row locking")
	}
	request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
	locked, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	hook := "journal_merge:pause_merge"
	if err := f.svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		if tx.Error != nil {
			return
		}
		if _, ok := tx.Statement.Dest.(*[]models.Journal); !ok {
			return
		}
		if _, ok := tx.Statement.Clauses["FOR"]; !ok {
			return
		}
		if paused.CompareAndSwap(false, true) {
			close(locked)
			select {
			case <-resume:
			case <-time.After(15 * time.Second):
				tx.AddError(errors.New("journal merge barrier timed out"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer f.svc.db.Callback().Query().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	merged := make(chan error, 1)
	go func() {
		_, err := NewContactService(f.svc.db.WithContext(ctx)).MergeContacts(f.vault, f.user, request)
		merged <- err
	}()
	awaitMergeBoundarySignal(t, locked)
	edited := make(chan error, 1)
	go func() {
		_, err := NewPostService(f.svc.db.WithContext(ctx)).Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Stale edit", Sections: []dto.PostSectionInput{{Position: 1, Content: f.body, ContentFormat: "markdown"}}})
		edited <- err
	}()
	waited := waitForContactMergeLock(f.svc.db)
	close(resume)
	mergeErr, editErr := awaitMergeBoundaryError(t, merged), awaitMergeBoundaryError(t, edited)
	if mergeErr != nil || !errors.Is(editErr, ErrContactNotFound) {
		t.Fatalf("merge=%v stale editor=%v", mergeErr, editErr)
	}
	if !waited {
		t.Fatal("editor did not wait for merge contact lock")
	}
	got, err := f.posts.Get(f.post.ID, f.journal, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Walk" || got.Sections[0].Content != strings.ReplaceAll(f.body, f.source, f.target) || len(got.Contacts) != 1 || got.Contacts[0].ID != f.target {
		t.Fatalf("stale edit changed merged post: %+v", got)
	}
	refreshed, err := f.posts.Update(f.post.ID, f.journal, f.vault, dto.UpdatePostRequest{Title: "Refreshed edit", Sections: []dto.PostSectionInput{{Position: 1, Content: got.Sections[0].Content, ContentFormat: "markdown"}}})
	if err != nil || refreshed.Title != "Refreshed edit" {
		t.Fatalf("refresh did not recover: %+v %v", refreshed, err)
	}
}

func waitForJournalMergeLock(db *gorm.DB) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE '%journals%FOR UPDATE%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='journals'::regclass)`).Scan(&count).Error
		if err != nil {
			return false
		}
		if count > 0 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
