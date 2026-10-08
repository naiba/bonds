package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
)

func TestEncodedContactDestinationSurvivesMerge(t *testing.T) {
	for _, kind := range []string{"escaped_colon", "entity_colon", "escaped_uuid", "encoded_address", "reference_definition"} {
		t.Run(kind, func(t *testing.T) {
			f := setupContactMergeContent(t)
			destination := "contact:" + f.source
			switch kind {
			case "escaped_colon":
				destination = `contact\:` + f.source
			case "entity_colon":
				destination = "contact&#58;" + f.source
			case "encoded_address":
				destination = encodeContactAddress(destination)
			case "escaped_uuid":
				destination = "contact:" + strings.ReplaceAll(f.source, "-", `\-`)
			}
			live := "Met [Duplicate](" + destination + ") after lunch."
			if kind == "reference_definition" {
				live = "Met [Duplicate][friend].\n\n[friend]: " + encodeContactAddress(destination)
			}
			literal := "`[Example](" + destination + ")`\n\\[Example](" + destination + ")\n\n"
			body := literal + live
			notes := NewNoteService(f.svc.db)
			note, err := notes.Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: body, BodyFormat: "markdown"})
			if err != nil {
				t.Fatal(err)
			}
			activities := NewActivityService(f.svc.db)
			activity, err := activities.UpdateForUser(f.vault, f.user, f.activity.ID, f.activityRequest(body))
			if err != nil {
				t.Fatal(err)
			}
			journal := models.Journal{VaultID: f.vault, Name: "Synthetic encoded destination"}
			if err := f.svc.db.Create(&journal).Error; err != nil {
				t.Fatal(err)
			}
			posts := NewPostService(f.svc.db)
			post, err := posts.Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Remember", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
			if err != nil {
				t.Fatal(err)
			}
			for name, rendered := range map[string]string{"note": note.RenderedBody, "activity": activity.RenderedDescription, "journal": post.Sections[0].RenderedContent} {
				if !strings.Contains(rendered, `data-bonds-contact="`+f.source+`"`) {
					t.Fatalf("%s fixture not live: %s", name, rendered)
				}
			}
			// Legacy posts may have no pivot. Discovery must use the decoded
			// destination even when the raw body contains neither contact nor UUID.
			if err := f.svc.db.Where("post_id = ?", post.ID).Delete(&models.ContactPost{}).Error; err != nil {
				t.Fatal(err)
			}
			preview, err := f.svc.PreviewContactMerge(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source))
			if err != nil {
				t.Fatal(err)
			}
			if preview.Effects["note_mentions"] != 1 || preview.Effects["activity_mentions"] != 1 || preview.Effects["posts"] != 1 {
				t.Fatalf("effects=%+v", preview.Effects)
			}
			if _, err := f.svc.GetContact(f.source, f.user, f.vault); err != nil {
				t.Fatal(err)
			}
			if _, err := f.svc.MergeContacts(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)); err != nil {
				t.Fatal(err)
			}
			if _, err := f.svc.GetContact(f.source, f.user, f.vault); err != ErrContactNotFound {
				t.Fatalf("source not removed: %v", err)
			}
			var stored models.Note
			if err := f.svc.db.First(&stored, f.note.ID).Error; err != nil {
				t.Fatal(err)
			}
			note.RenderedBody = markdown.Render(stored.Body, stored.BodyFormat)
			activity, err = activities.Get(f.vault, f.user, f.activity.ID)
			if err != nil {
				t.Fatal(err)
			}
			post, err = posts.Get(post.ID, journal.ID, f.vault)
			if err != nil {
				t.Fatal(err)
			}
			scheme := "contact:"
			switch kind {
			case "escaped_colon":
				scheme = `contact\:`
			case "entity_colon":
				scheme = "contact&#58;"
			case "encoded_address", "reference_definition":
				scheme = encodeContactAddress("contact:")
			}
			want := literal + "Met [Duplicate](" + scheme + f.target + ") after lunch."
			if kind == "reference_definition" {
				want = literal + "Met [Duplicate][friend].\n\n[friend]: " + scheme + f.target
			}
			for name, body := range map[string]string{"note": stored.Body, "activity": activity.Description, "journal": post.Sections[0].Content} {
				if body != want {
					t.Errorf("%s: body=%q want=%q", name, body, want)
				}
			}
			if len(post.Contacts) != 1 || post.Contacts[0].ID != f.target {
				t.Fatalf("journal association=%+v", post.Contacts)
			}
			for name, rendered := range map[string]string{"note": note.RenderedBody, "activity": activity.RenderedDescription, "journal": post.Sections[0].RenderedContent} {
				if strings.Contains(rendered, `data-bonds-contact="`+f.source+`"`) || !strings.Contains(rendered, `data-bonds-contact="`+f.target+`"`) {
					t.Errorf("%s live link points to deleted source after merge: %s", name, rendered)
				}
			}
		})
	}
}

func encodeContactAddress(address string) string {
	var encoded strings.Builder
	for _, character := range address {
		fmt.Fprintf(&encoded, "&#x%x;", character)
	}
	return encoded.String()
}

func TestEncodedContactReferencesRespectWriteValidation(t *testing.T) {
	f := setupContactMergeContent(t)
	missing := "550e8400-e29b-41d4-a716-446655440000"
	link := "[Unknown](" + encodeContactAddress("contact:"+missing) + ")"
	notes := NewNoteService(f.svc.db)
	activities := NewActivityService(f.svc.db)
	journal := models.Journal{VaultID: f.vault, Name: "Synthetic encoded validation"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	posts := NewPostService(f.svc.db)
	for _, literal := range []bool{false, true} {
		body := link
		if literal {
			body = "`" + link + "`"
		}
		_, noteErr := notes.Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: body, BodyFormat: "markdown"})
		_, activityErr := activities.CreateForUser(f.vault, f.user, f.activityRequest(body))
		_, postErr := posts.Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Example", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
		for kind, err := range map[string]error{"note": noteErr, "activity": activityErr, "journal": postErr} {
			if literal && err != nil {
				t.Errorf("%s literal rejected: %v", kind, err)
			}
			if !literal && err != ErrContactNotFound {
				t.Errorf("%s missing live reference: %v", kind, err)
			}
		}
	}
}

func TestNonRenderedContactDestinationsDoNotRequireLiveContacts(t *testing.T) {
	f := setupContactMergeContent(t)
	destination := "contact:550e8400-e29b-41d4-a716-446655440000"
	journal := models.Journal{VaultID: f.vault, Name: "Synthetic noninteractive destinations"}
	if err := f.svc.db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"[outer <" + destination + ">](https://example.test)", "Text[^friend]\n\n[^friend]: [Unknown](" + destination + ")"} {
		if strings.Contains(markdown.Render(body, "markdown"), "data-bonds-contact=") {
			t.Fatal("fixture is a contact link")
		}
		note, err := NewNoteService(f.svc.db).Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: body, BodyFormat: "markdown"})
		if err != nil {
			t.Fatal(err)
		}
		if note.Body != body {
			t.Fatal("literal note changed")
		}
		activity, err := NewActivityService(f.svc.db).CreateForUser(f.vault, f.user, f.activityRequest(body))
		if err != nil {
			t.Fatal(err)
		}
		if activity.Description != body || len(activity.MentionedContacts) != 0 {
			t.Fatal("noninteractive activity destination interpreted as mention")
		}
		post, err := NewPostService(f.svc.db).Create(journal.ID, f.vault, dto.CreatePostRequest{Title: "Example", Sections: []dto.PostSectionInput{{Content: body, ContentFormat: "markdown"}}})
		if err != nil {
			t.Fatal(err)
		}
		if post.Sections[0].Content != body || len(post.Contacts) != 0 {
			t.Fatal("noninteractive post destination interpreted as mention")
		}
	}
}
