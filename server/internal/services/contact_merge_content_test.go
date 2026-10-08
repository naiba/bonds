package services

import (
	"errors"
	"fmt"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"testing"
)

func TestContactMergePreservesContentReferenceLifecycle(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Retained"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	noteService := NewNoteService(svc.db)
	body := "Met @[Duplicate](contact:" + source.ID + "). Keep **formatting**."
	note, err := noteService.Create(source.ID, vault, user, dto.CreateNoteRequest{Title: "Synthetic meeting", Body: body, BodyFormat: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	var kind models.ActivityType
	if err := svc.db.Joins("JOIN activity_categories ON activity_categories.id = activity_types.activity_category_id").Where("activity_categories.vault_id = ?", vault).First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	activities := NewActivityService(svc.db)
	event, err := activities.Create(vault, dto.ActivityUpsertRequest{Title: "Synthetic walk", ActivityTypeID: kind.ID, PrimaryContactID: source.ID, Description: body, DescriptionFormat: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := activities.Get(vault, user, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.MentionedContacts) != 1 || before.MentionedContacts[0].ID != source.ID {
		t.Fatal("fixture lacks a live activity mention")
	}
	if _, err := svc.GetContact(source.ID, user, vault); err != nil {
		t.Fatal("source link did not resolve before merge", err)
	}
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	after, err := activities.Get(vault, user, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Participants) != 1 || after.Participants[0].ID != target.ID {
		t.Fatal("participant migration failed")
	}
	if _, err := svc.GetContact(source.ID, user, vault); err != ErrContactNotFound {
		t.Fatalf("expected removed source, got %v", err)
	}
	var stored models.Note
	if err := svc.db.Where("id = ? AND contact_id = ?", note.ID, target.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	expected := strings.ReplaceAll(body, source.ID, target.ID)
	if stored.Body != expected {
		t.Errorf("migrated note retains dead contact reference: got %q want %q", stored.Body, expected)
	}
	if after.Description != expected || len(after.MentionedContacts) != 1 || after.MentionedContacts[0].ID != target.ID {
		t.Errorf("migrated activity has dead reference: body=%q mentioned=%+v participants=%+v", after.Description, after.MentionedContacts, after.Participants)
	}
	if !strings.Contains(after.RenderedDescription, target.ID) || strings.Contains(after.RenderedDescription, source.ID) {
		t.Errorf("rendered activity links to removed source: %s", after.RenderedDescription)
	}
}

type contactMergeContentFixture struct {
	svc                                         *ContactService
	vault, user, account, target, source, owner string
	note                                        *dto.NoteResponse
	activity                                    *dto.ActivityResponse
	activityType                                uint
	body                                        string
}

func setupContactMergeContent(t *testing.T) contactMergeContentFixture {
	t.Helper()
	svc, vault, user, account := setupContactTest(t)
	contacts := make([]models.Contact, 3)
	for i, name := range []string{"Retained", "Duplicate", "Observer"} {
		contacts[i] = models.Contact{VaultID: vault, FirstName: strPtrOrNil(name)}
		if err := svc.db.Create(&contacts[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	body := "Remember [Duplicate](contact:" + contacts[1].ID + ")"
	note, err := NewNoteService(svc.db).Create(contacts[2].ID, vault, user, dto.CreateNoteRequest{Title: "Observer's note", Body: body, BodyFormat: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	var kind models.ActivityType
	if err := svc.db.Joins("JOIN activity_categories ON activity_categories.id = activity_types.activity_category_id").Where("activity_categories.vault_id = ?", vault).First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	empty := []string{}
	activity, err := NewActivityService(svc.db).CreateForUser(vault, user, dto.ActivityUpsertRequest{Title: "Personal memory", ActivityTypeID: kind.ID, Description: body, DescriptionFormat: "markdown", ParticipantIDs: &empty})
	if err != nil {
		t.Fatal(err)
	}
	return contactMergeContentFixture{svc, vault, user, account, contacts[0].ID, contacts[1].ID, contacts[2].ID, note, activity, kind.ID, body}
}

func (f contactMergeContentFixture) activityRequest(body string) dto.ActivityUpsertRequest {
	empty := []string{}
	return dto.ActivityUpsertRequest{Title: "Personal memory", ActivityTypeID: f.activityType, Description: body, DescriptionFormat: "markdown", ParticipantIDs: &empty}
}

func TestContactMergeContentPreservesScopeAndBytes(t *testing.T) {
	for _, format := range []string{"plain", "markdown"} {
		t.Run(format, func(t *testing.T) {
			f := setupContactMergeContent(t)
			file := models.File{VaultID: f.vault, UUID: "mention-attachment", Name: "Synthetic picture", MimeType: "image/png", Type: "photo"}
			if err := f.svc.db.Create(&file).Error; err != nil {
				t.Fatal(err)
			}
			prefix := "Raw " + f.source + " [link](https://example.test/" + f.source + ") <span>" + f.source + "</span>\n"
			name := `A\]lice\\ ` + f.source
			body := prefix + "@[" + name + "](contact:" + strings.ToUpper(f.source) + ")\n[Again](contact:" + f.source + ") @[Observer](contact:" + f.owner + ")" + fmt.Sprintf(" ![Photo](bonds-file:%d)", file.ID)
			want := prefix + "@[" + name + "](contact:" + f.target + ")\n[Again](contact:" + f.target + ") @[Observer](contact:" + f.owner + ")" + fmt.Sprintf(" ![Photo](bonds-file:%d)", file.ID)
			if _, err := NewNoteService(f.svc.db).Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Title: "Observer's note", Body: body, BodyFormat: format}); err != nil {
				t.Fatal(err)
			}
			req := f.activityRequest(body)
			req.DescriptionFormat = format
			if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, req); err != nil {
				t.Fatal(err)
			}
			foreign, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Other vault"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			foreignOwner := models.Contact{VaultID: foreign.ID}
			if err := f.svc.db.Create(&foreignOwner).Error; err != nil {
				t.Fatal(err)
			}
			foreignNote := models.Note{VaultID: foreign.ID, ContactID: foreignOwner.ID, Body: body}
			foreignActivity := models.Activity{VaultID: foreign.ID, Title: "History", Description: &body}
			malformed := models.Note{VaultID: f.vault, ContactID: f.owner, Body: "[]](contact:" + f.source + ") raw " + f.source}
			for _, row := range []any{&foreignNote, &foreignActivity, &malformed} {
				if err := f.svc.db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			// No permission to the foreign vault: textual matches must neither block nor edit it.
			if err := f.svc.db.Where("vault_id = ? AND user_id = ?", foreign.ID, f.user).Delete(&models.UserVault{}).Error; err != nil {
				t.Fatal(err)
			}
			var attachmentsBefore []models.ContentFileReference
			if err := f.svc.db.Where("vault_id = ?", f.vault).Order("id").Find(&attachmentsBefore).Error; err != nil {
				t.Fatal(err)
			}
			if format == "markdown" && len(attachmentsBefore) != 2 {
				t.Fatalf("attachment fixture: %+v", attachmentsBefore)
			}
			merge := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			review, err := f.svc.PreviewContactMerge(f.vault, f.user, merge)
			if err != nil {
				t.Fatal(err)
			}
			if review.Effects["note_mentions"] != 1 || review.Effects["activity_mentions"] != 1 {
				t.Fatalf("missing mention review: %+v", review.Effects)
			}
			if _, err := f.svc.MergeContacts(f.vault, f.user, merge); err != nil {
				t.Fatal(err)
			}
			var note models.Note
			if err := f.svc.db.First(&note, f.note.ID).Error; err != nil {
				t.Fatal(err)
			}
			activity, err := NewActivityService(f.svc.db).Get(f.vault, f.user, f.activity.ID)
			if err != nil {
				t.Fatal(err)
			}
			if note.Body != want || note.ContactID != f.owner || note.BodyFormat != format || ptrToStr(note.Title) != "Observer's note" {
				t.Fatalf("note changed incorrectly: %+v", note)
			}
			if activity.Description != want || activity.DescriptionFormat != format || len(activity.Participants) != 0 || len(activity.MentionedContacts) != 2 {
				t.Fatalf("activity content/associations changed: %+v", activity)
			}
			for _, row := range []any{&foreignNote, &foreignActivity, &malformed} {
				if err := f.svc.db.First(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			if foreignNote.Body != body || ptrToStr(foreignActivity.Description) != body || malformed.Body != "[]](contact:"+f.source+") raw "+f.source {
				t.Fatal("foreign or malformed content rewritten")
			}
			var attachmentsAfter []models.ContentFileReference
			if err := f.svc.db.Where("vault_id = ?", f.vault).Order("id").Find(&attachmentsAfter).Error; err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(attachmentsBefore, attachmentsAfter) {
				t.Fatal("attachment identities changed")
			}
			// A refreshed editor works; an editor opened before merge cannot restore dead links.
			if _, err := NewNoteService(f.svc.db).Update(note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: want, BodyFormat: format}); err != nil {
				t.Fatal(err)
			}
			if _, err := NewNoteService(f.svc.db).Update(note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body, BodyFormat: format}); !errors.Is(err, ErrContactNotFound) {
				t.Fatalf("stale note restored source: %v", err)
			}
			req.Description = want
			if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, activity.ID, req); err != nil {
				t.Fatal(err)
			}
			req.Description = body
			if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, activity.ID, req); !errors.Is(err, ErrContactNotFound) {
				t.Fatalf("stale activity restored source: %v", err)
			}
		})
	}
}

func TestContactMergeContentReviewAndRollback(t *testing.T) {
	for _, kind := range []string{"note", "activity", "rollback"} {
		t.Run(kind, func(t *testing.T) {
			f := setupContactMergeContent(t)
			ownedNote := models.Note{VaultID: f.vault, ContactID: f.source, Body: f.body}
			participant := models.ActivityParticipant{ActivityID: f.activity.ID, ContactID: f.source}
			for _, row := range []any{&ownedNote, &participant} {
				if err := f.svc.db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			merge := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			if kind == "note" {
				if _, err := NewNoteService(f.svc.db).Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: "Edited " + f.body}); err != nil {
					t.Fatal(err)
				}
			} else if kind == "activity" {
				edit := f.activityRequest("Edited " + f.body)
				edit.ParticipantIDs = nil // Preserve membership while changing only the description.
				if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, edit); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "rollback" {
				if _, err := f.svc.MergeContacts(f.vault, f.user, merge); !errors.Is(err, ErrContactMergeReviewChanged) {
					t.Fatalf("same-count body change allowed: %v", err)
				}
			} else {
				// Fail after both content updates and source deletion at the final audit write.
				rejected := errors.New("content merge audit rejected")
				hook := "content_merge:reject_audit"
				if err := f.svc.db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
					if tx.Statement.Table == "contact_feed_items" {
						tx.AddError(rejected)
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer f.svc.db.Callback().Create().Remove(hook)
				if _, err := f.svc.MergeContacts(f.vault, f.user, merge); !errors.Is(err, rejected) {
					t.Fatalf("missing injected error: %v", err)
				}
			}
			var note models.Note
			var activity models.Activity
			var source models.Contact
			for _, row := range []struct {
				value any
				id    any
			}{{&note, f.note.ID}, {&activity, f.activity.ID}, {&source, f.source}} {
				if err := f.svc.db.Where("id = ?", row.id).First(row.value).Error; err != nil {
					t.Fatal(err)
				}
			}
			noteWant, activityWant := f.body, f.body
			if kind == "note" {
				noteWant = "Edited " + noteWant
			}
			if kind == "activity" {
				activityWant = "Edited " + activityWant
			}
			if note.Body != noteWant || note.ContactID != f.owner || ptrToStr(activity.Description) != activityWant {
				t.Fatal("rejected merge changed content")
			}
			if err := f.svc.db.First(&ownedNote, ownedNote.ID).Error; err != nil {
				t.Fatal(err)
			}
			var participants []models.ActivityParticipant
			if err := f.svc.db.Where("activity_id = ?", f.activity.ID).Find(&participants).Error; err != nil {
				t.Fatal(err)
			}
			if len(participants) != 1 || participants[0].ContactID != f.source {
				t.Fatalf("membership escaped rejected merge: %+v", participants)
			}
			if kind != "activity" && participants[0].ID != participant.ID {
				t.Fatal("membership identity changed on rollback")
			}
			if ownedNote.ContactID != f.source || ownedNote.Body != f.body {
				t.Fatal("rejected merge changed note ownership or activity participation")
			}
			var count int64
			if err := f.svc.db.Model(&models.ContactFeedItem{}).Where("contact_id = ?", f.target).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("audit escaped rollback")
			}
		})
	}
}

func TestContentMentionEditsPreserveHistoricalReferences(t *testing.T) {
	f := setupContactMergeContent(t)
	foreign, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Moved to"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.db.Model(&models.Contact{}).Where("id = ?", f.source).Update("vault_id", foreign.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewNoteService(f.svc.db).Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Title: "Retitled", Body: f.body}); err != nil {
		t.Fatal("historical note edit blocked", err)
	}
	if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, f.activityRequest(f.body)); err != nil {
		t.Fatal("historical activity edit blocked", err)
	}
	if _, err := NewNoteService(f.svc.db).Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: f.body}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("new foreign note reference allowed: %v", err)
	}
	if _, err := NewActivityService(f.svc.db).CreateForUser(f.vault, f.user, f.activityRequest(f.body)); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("new foreign activity reference allowed: %v", err)
	}
	var count int64
	if err := f.svc.db.Model(&models.Note{}).Where("vault_id = ?", f.vault).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed create persisted note: %d", count)
	}
	if err := f.svc.db.Model(&models.Activity{}).Where("vault_id = ?", f.vault).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed create persisted activity: %d", count)
	}
}

type contentMergeSearchRecorder struct {
	search.NoopEngine
	notes []models.Note
}

func (e *contentMergeSearchRecorder) IndexNote(id, vault, contact, title, body string) error {
	e.notes = append(e.notes, models.Note{VaultID: vault, ContactID: contact, Title: &title, Body: body})
	return nil
}

func TestContactMergeIndexesCommittedMentionBodies(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprint(rollback), func(t *testing.T) {
			f := setupContactMergeContent(t)
			engine := &contentMergeSearchRecorder{}
			f.svc.SetSearchService(NewSearchServiceWithDB(f.svc.db, engine))
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			rejected := errors.New("merge indexing rollback")
			if rollback {
				hook := "content_merge:reject_before_index"
				if err := f.svc.db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
					if tx.Statement.Table == "contact_feed_items" {
						tx.AddError(rejected)
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer f.svc.db.Callback().Create().Remove(hook)
			}
			_, err := f.svc.MergeContacts(f.vault, f.user, request)
			if rollback {
				if !errors.Is(err, rejected) || len(engine.notes) != 0 {
					t.Fatalf("rollback reached search: error=%v notes=%+v", err, engine.notes)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if len(engine.notes) != 1 || engine.notes[0].ContactID != f.owner || engine.notes[0].VaultID != f.vault || engine.notes[0].Body != strings.ReplaceAll(f.body, f.source, f.target) {
					t.Fatalf("other owner's note index stale: %+v", engine.notes)
				}
			}
		})
	}
}
