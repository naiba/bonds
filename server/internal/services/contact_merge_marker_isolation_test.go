package services

import (
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
)

func parsingMarkerCollisionBody(id string) string {
	return "`[Example](contact:" + id + ")`\n\n[Unrelated](bonds&#45;contact-reference-0-contact:" + id + ")"
}
func TestParsingMarkerCollisionPreservesLiteralContent(t *testing.T) {
	f := setupContactMergeContent(t)
	body := parsingMarkerCollisionBody(f.source)
	if strings.Contains(markdown.Render(body, "markdown"), "data-bonds-contact=") {
		t.Fatal("fixture is interactive")
	}
	notes := NewNoteService(f.svc.db)
	if _, err := notes.Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body, BodyFormat: "markdown"}); err != nil {
		t.Fatal(err)
	}
	activities := NewActivityService(f.svc.db)
	if _, err := activities.UpdateForUser(f.vault, f.user, f.activity.ID, f.activityRequest(body)); err != nil {
		t.Fatal(err)
	}
	journal := models.Journal{VaultID: f.vault, Name: "Synthetic literal preservation"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	posts := NewPostService(f.svc.db)
	post, err := posts.Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Literal example", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Contacts) != 0 {
		t.Fatal("literal created a journal association")
	}
	preview, err := f.svc.PreviewContactMerge(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Effects["note_mentions"] != 0 || preview.Effects["activity_mentions"] != 0 || preview.Effects["posts"] != 0 {
		t.Fatalf("literal affected review: %+v", preview.Effects)
	}
	if _, err := f.svc.MergeContacts(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.GetContact(f.source, f.user, f.vault); err != ErrContactNotFound {
		t.Fatalf("source not removed: %v", err)
	}
	var note models.Note
	if err := f.svc.db.First(&note, f.note.ID).Error; err != nil {
		t.Fatal(err)
	}
	activity, err := activities.Get(f.vault, f.user, f.activity.ID)
	if err != nil {
		t.Fatal(err)
	}
	post, err = posts.Get(post.ID, journal.ID, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Contacts) != 0 || len(activity.MentionedContacts) != 0 {
		t.Fatal("literal created associations after merge")
	}
	for kind, got := range map[string]string{"note": note.Body, "activity": activity.Description, "journal": post.Sections[0].Content} {
		if got != body {
			t.Errorf("%s code sample corrupted by merge: got=%q want=%q", kind, got, body)
		}
	}
}
func TestParsingMarkerCollisionAllowsNonContactText(t *testing.T) {
	f := setupContactMergeContent(t)
	body := parsingMarkerCollisionBody("550e8400-e29b-41d4-a716-446655440000")
	if strings.Contains(markdown.Render(body, "markdown"), "data-bonds-contact=") {
		t.Fatal("fixture is interactive")
	}
	if _, err := NewNoteService(f.svc.db).Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: body, BodyFormat: "markdown"}); err != nil {
		t.Errorf("noninteractive text rejected: %v", err)
	}
}

func TestParsingMarkerCollisionAllowsActivityAndJournalExamples(t *testing.T) {
	f := setupContactMergeContent(t)
	body := parsingMarkerCollisionBody("550e8400-e29b-41d4-a716-446655440000")
	activity, err := NewActivityService(f.svc.db).CreateForUser(f.vault, f.user, f.activityRequest(body))
	if err != nil {
		t.Fatal(err)
	}
	if activity.Description != body || len(activity.MentionedContacts) != 0 {
		t.Fatal("literal activity interpreted as mention")
	}
	journal := models.Journal{VaultID: f.vault, Name: "Synthetic parser marker isolation"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	post, err := NewPostService(f.svc.db).Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Literal example", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
	if err != nil {
		t.Fatal(err)
	}
	if post.Sections[0].Content != body || len(post.Contacts) != 0 {
		t.Fatal("literal journal interpreted as mention")
	}
}
