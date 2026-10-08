package services

import (
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
)

func TestContactMergePreservesMarkdownLiteralExamples(t *testing.T) {
	for _, kind := range []string{"inline_code", "fenced_code", "escaped_link", "image_label"} {
		t.Run(kind, func(t *testing.T) {
			f := setupContactMergeContent(t)
			literal := "[Duplicate](contact:" + f.source + ")"
			var body string
			switch kind {
			case "inline_code":
				body = "Example: `" + literal + "`"
			case "fenced_code":
				body = "Example:\n```text\n" + literal + "\n```"
			case "image_label":
				body = "![example " + literal + "](https://example.test/photo.png)"
			case "escaped_link":
				body = "Example: \\" + literal
			}
			rendered := markdown.Render(body, markdown.FormatMarkdown)
			if strings.Contains(rendered, `data-bonds-contact=`) {
				t.Fatalf("fixture is a live reference, not literal text: %s", rendered)
			}
			_, err := NewNoteService(f.svc.db).Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body, BodyFormat: "markdown"})
			if err != nil {
				t.Fatal(err)
			}
			req := f.activityRequest(body)
			req.DescriptionFormat = "markdown"
			if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, req); err != nil {
				t.Fatal(err)
			}
			if _, err := f.svc.MergeContacts(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)); err != nil {
				t.Fatal(err)
			}
			var stored models.Note
			if err := f.svc.db.First(&stored, f.note.ID).Error; err != nil {
				t.Fatal(err)
			}
			activity, err := NewActivityService(f.svc.db).Get(f.vault, f.user, f.activity.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Body != body {
				t.Errorf("merge rewrote literal note text: got %q want %q", stored.Body, body)
			}
			if activity.Description != body {
				t.Errorf("merge rewrote literal activity text: got %q want %q", activity.Description, body)
			}
		})
	}
}

func TestMarkdownLiteralDoesNotRequireLiveContact(t *testing.T) {
	f := setupContactMergeContent(t)
	body := "A literal example: `[Unknown](contact:550e8400-e29b-41d4-a716-446655440000)`"
	if strings.Contains(markdown.Render(body, "markdown"), "data-bonds-contact=") {
		t.Fatal("fixture is not literal")
	}
	if _, err := NewNoteService(f.svc.db).Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: body, BodyFormat: "markdown"}); err != nil {
		t.Errorf("ordinary literal note text rejected: %v", err)
	}
	req := f.activityRequest(body)
	req.DescriptionFormat = "markdown"
	if _, err := NewActivityService(f.svc.db).CreateForUser(f.vault, f.user, req); err != nil {
		t.Errorf("ordinary literal activity text rejected: %v", err)
	}
}

func TestContactMergeRedirectsOnlyRenderedLinksInMixedMarkdown(t *testing.T) {
	f := setupContactMergeContent(t)
	literal := "`[Duplicate](contact:" + f.source + ")`\n\\[Duplicate](contact:" + f.source + ")"
	live := "[Duplicate](contact:" + f.source + ") @[Again](contact:" + f.source + ")"
	body := literal + "\n\n" + live
	want := literal + "\n\n" + strings.ReplaceAll(live, f.source, f.target)
	if _, err := NewNoteService(f.svc.db).Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body, BodyFormat: "markdown"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, f.activityRequest(body)); err != nil {
		t.Fatal(err)
	}
	journal := models.Journal{VaultID: f.vault, Name: "Synthetic examples"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	posts := NewPostService(f.svc.db)
	post, err := posts.Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Examples", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
	if err != nil {
		t.Fatal(err)
	}
	review, err := f.svc.PreviewContactMerge(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source))
	if err != nil {
		t.Fatal(err)
	}
	if review.Effects["note_mentions"] != 1 || review.Effects["activity_mentions"] != 1 || review.Effects["posts"] != 1 {
		t.Fatalf("effects=%+v", review.Effects)
	}
	if _, err := f.svc.MergeContacts(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)); err != nil {
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
	gotPost, err := posts.Get(post.ID, journal.ID, f.vault)
	if err != nil {
		t.Fatal(err)
	}
	if note.Body != want || activity.Description != want || gotPost.Sections[0].Content != want {
		t.Fatalf("unexpected body: note=%q activity=%q post=%q", note.Body, activity.Description, gotPost.Sections[0].Content)
	}
	if len(gotPost.Contacts) != 1 || gotPost.Contacts[0].ID != f.target || len(activity.MentionedContacts) != 1 || activity.MentionedContacts[0].ID != f.target {
		t.Fatal("associations not redirected")
	}
}

func TestMarkdownLiteralEditsRetainOmittedFormats(t *testing.T) {
	f := setupContactMergeContent(t)
	body := "`[Unknown](contact:550e8400-e29b-41d4-a716-446655440000)`"
	notes := NewNoteService(f.svc.db)
	note, err := notes.Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body})
	if err != nil {
		t.Fatal(err)
	}
	if note.BodyFormat != "markdown" || note.Body != body {
		t.Fatalf("note format/body changed: %+v", note)
	}
	req := f.activityRequest(body)
	req.DescriptionFormat = ""
	activity, err := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if activity.DescriptionFormat != "markdown" || len(activity.MentionedContacts) != 0 {
		t.Fatal("activity interpreted literal as a mention")
	}
	journal := models.Journal{VaultID: f.vault, Name: "Synthetic examples"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	posts := NewPostService(f.svc.db)
	post, err := posts.Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Examples", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Contacts) != 0 {
		t.Fatal("literal created an association")
	}
	updated, err := posts.Update(post.ID, journal.ID, f.vault, dto.UpdatePostRequest{Title: "Retitled", Sections: []dto.PostSectionInput{{Content: body}}, ContactIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Sections[0].ContentFormat != "markdown" || updated.Sections[0].Content != body || len(updated.Contacts) != 0 {
		t.Fatalf("post format/body/associations changed: %+v", updated)
	}
	// Switching to plain makes the marker interactive, so a new dead link must
	// still fail validation rather than inheriting literal text as history.
	if _, err := notes.Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body, BodyFormat: "plain"}); err != ErrContactNotFound {
		t.Fatalf("new plain dead reference accepted: %v", err)
	}
}
