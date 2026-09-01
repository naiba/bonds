package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type postTestContext struct {
	svc       *PostService
	journalID uint
	vaultID   string
	db        *gorm.DB
}

func setupPostTest(t *testing.T) (*PostService, uint, string) {
	t.Helper()
	ctx := setupPostTestFull(t)
	return ctx.svc, ctx.journalID, ctx.vaultID
}

func setupPostTestFull(t *testing.T) postTestContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "posts-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	journalSvc := NewJournalService(db)
	journal, err := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})
	if err != nil {
		t.Fatalf("CreateJournal failed: %v", err)
	}

	return postTestContext{
		svc:       NewPostService(db),
		journalID: journal.ID,
		vaultID:   vault.ID,
		db:        db,
	}
}

func TestCreatePost(t *testing.T) {
	svc, journalID, vaultID := setupPostTest(t)

	post, err := svc.Create(journalID, vaultID, dto.CreatePostRequest{
		Title:     "My First Post",
		Published: true,
		WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{
			{Position: 1, Label: "Intro", Content: "Hello world"},
			{Position: 2, Label: "Body", Content: "Main content"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.Title != "My First Post" {
		t.Errorf("Expected title 'My First Post', got '%s'", post.Title)
	}
	if !post.Published {
		t.Error("Expected published to be true")
	}
	if post.JournalID != journalID {
		t.Errorf("Expected journal_id %d, got %d", journalID, post.JournalID)
	}
	if len(post.Sections) != 2 {
		t.Errorf("Expected 2 sections, got %d", len(post.Sections))
	}
	if post.ID == 0 {
		t.Error("Expected post ID to be non-zero")
	}
}

func TestPostMarkdownRendersAndTracksUploadedFiles(t *testing.T) {
	ctx := setupPostTestFull(t)
	file := models.File{VaultID: ctx.vaultID, UUID: "markdown-file", Name: "photo.png", MimeType: "image/png", Type: "photo"}
	if err := ctx.db.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	post, err := ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title: "Markdown", WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{{
			Position: 0, Label: "Body", Content: "**bold**\n\n![photo](bonds-file:" + fmt.Sprint(file.ID) + ")", ContentFormat: "markdown",
		}},
	})
	if err != nil {
		t.Fatalf("Create Markdown post: %v", err)
	}
	section := post.Sections[0]
	if section.ContentFormat != "markdown" || !strings.Contains(section.RenderedContent, "<strong>bold</strong>") || !strings.Contains(section.RenderedContent, `data-bonds-file="`) {
		t.Fatalf("Markdown section response = %+v", section)
	}
	var referenceCount int64
	ctx.db.Model(&models.ContentFileReference{}).Where("file_id = ?", file.ID).Count(&referenceCount)
	if referenceCount != 1 {
		t.Fatalf("content file reference count = %d, want 1", referenceCount)
	}
	fileSvc := NewVaultFileService(ctx.db, t.TempDir())
	if err := fileSvc.Delete(file.ID, ctx.vaultID); !errors.Is(err, ErrFileInUse) {
		t.Fatalf("Delete referenced file error = %v, want ErrFileInUse", err)
	}

	legacyUpdated, err := ctx.svc.Update(post.ID, ctx.journalID, ctx.vaultID, dto.UpdatePostRequest{
		Title: "Legacy client update", WrittenAt: post.WrittenAt,
		Sections: []dto.PostSectionInput{{Position: 0, Label: "Body", Content: "![photo](bonds-file:" + fmt.Sprint(file.ID) + ")"}},
	})
	if err != nil {
		t.Fatalf("Update Markdown post without format: %v", err)
	}
	if legacyUpdated.Sections[0].ContentFormat != "markdown" {
		t.Fatalf("ContentFormat after legacy update = %q, want markdown", legacyUpdated.Sections[0].ContentFormat)
	}
	ctx.db.Model(&models.ContentFileReference{}).Where("file_id = ?", file.ID).Count(&referenceCount)
	if referenceCount != 1 {
		t.Fatalf("content file reference count after legacy update = %d, want 1", referenceCount)
	}

	_, err = ctx.svc.Update(post.ID, ctx.journalID, ctx.vaultID, dto.UpdatePostRequest{
		Title: "Markdown", WrittenAt: post.WrittenAt,
		Sections: []dto.PostSectionInput{{Position: 0, Label: "Body", Content: "reference removed", ContentFormat: "markdown"}},
	})
	if err != nil {
		t.Fatalf("Update Markdown post: %v", err)
	}
	ctx.db.Model(&models.ContentFileReference{}).Where("file_id = ?", file.ID).Count(&referenceCount)
	if referenceCount != 0 {
		t.Fatalf("content file reference count after update = %d, want 0", referenceCount)
	}
}

func TestPostRejectsUnknownMarkdownFile(t *testing.T) {
	ctx := setupPostTestFull(t)
	_, err := ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title: "Missing file", WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{{Content: "[missing](bonds-file:999999)", ContentFormat: "markdown"}},
	})
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("Create error = %v, want ErrFileNotFound", err)
	}
}

func TestListPosts(t *testing.T) {
	ctx := setupPostTestFull(t)
	contact := models.Contact{VaultID: ctx.vaultID, FirstName: strPtrOrNil("Alice")}
	if err := ctx.db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	firstPost, err := ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title:      "Post 1",
		WrittenAt:  time.Now(),
		ContactIDs: []string{contact.ID},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title:     "Post 2",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	posts, err := ctx.svc.List(ctx.journalID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("Expected 2 posts, got %d", len(posts))
	}
	for _, post := range posts {
		if post.ID != firstPost.ID {
			continue
		}
		if len(post.Contacts) != 1 || post.Contacts[0].ID != contact.ID {
			t.Errorf("Expected Post 1 to include contact %q, got %+v", contact.ID, post.Contacts)
		}
		return
	}
	t.Errorf("Expected to find Post 1 with ID %d", firstPost.ID)
}

func TestGetPost(t *testing.T) {
	svc, journalID, vaultID := setupPostTest(t)

	created, err := svc.Create(journalID, vaultID, dto.CreatePostRequest{
		Title:     "Get Me",
		WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{
			{Position: 1, Label: "Section", Content: "Content"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.Get(created.ID, journalID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Title != "Get Me" {
		t.Errorf("Expected title 'Get Me', got '%s'", got.Title)
	}
	if got.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, got.ID)
	}
	if len(got.Sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(got.Sections))
	}
}

func TestUpdatePost(t *testing.T) {
	svc, journalID, vaultID := setupPostTest(t)

	created, err := svc.Create(journalID, vaultID, dto.CreatePostRequest{
		Title:     "Original",
		WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{
			{Position: 1, Label: "Old", Content: "Old content"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, journalID, vaultID, dto.UpdatePostRequest{
		Title:     "Updated",
		Published: true,
		Sections: []dto.PostSectionInput{
			{Position: 1, Label: "New", Content: "New content"},
			{Position: 2, Label: "Extra", Content: "Extra content"},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("Expected title 'Updated', got '%s'", updated.Title)
	}
	if !updated.Published {
		t.Error("Expected published to be true")
	}
	if len(updated.Sections) != 2 {
		t.Errorf("Expected 2 sections after update, got %d", len(updated.Sections))
	}
}

func TestDeletePost(t *testing.T) {
	svc, journalID, vaultID := setupPostTest(t)

	created, err := svc.Create(journalID, vaultID, dto.CreatePostRequest{
		Title:     "To delete",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, journalID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	posts, err := svc.List(journalID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("Expected 0 posts after delete, got %d", len(posts))
	}
}

func TestPostNotFound(t *testing.T) {
	svc, journalID, vaultID := setupPostTest(t)

	_, err := svc.Get(9999, journalID, vaultID)
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}

	_, err = svc.Update(9999, journalID, vaultID, dto.UpdatePostRequest{Title: "nope"})
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}

	err = svc.Delete(9999, journalID, vaultID)
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}
}

func TestPostGetIncrementsViewCount(t *testing.T) {
	svc, journalID, vaultID := setupPostTest(t)

	created, err := svc.Create(journalID, vaultID, dto.CreatePostRequest{
		Title:     "View Count Test",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	baseCount := created.ViewCount

	got1, err := svc.Get(created.ID, journalID, vaultID)
	if err != nil {
		t.Fatalf("Get #1 failed: %v", err)
	}
	if got1.ViewCount != baseCount+1 {
		t.Errorf("Expected view_count %d after first Get, got %d", baseCount+1, got1.ViewCount)
	}

	got2, err := svc.Get(created.ID, journalID, vaultID)
	if err != nil {
		t.Fatalf("Get #2 failed: %v", err)
	}
	if got2.ViewCount != baseCount+2 {
		t.Errorf("Expected view_count %d after second Get, got %d", baseCount+2, got2.ViewCount)
	}
}

func TestPostUpdateWithContacts(t *testing.T) {
	ctx := setupPostTestFull(t)

	contact1 := models.Contact{VaultID: ctx.vaultID, FirstName: strPtrOrNil("Alice"), LastName: strPtrOrNil("Smith")}
	if err := ctx.db.Create(&contact1).Error; err != nil {
		t.Fatalf("Create contact1 failed: %v", err)
	}
	contact2 := models.Contact{VaultID: ctx.vaultID, FirstName: strPtrOrNil("Bob"), LastName: strPtrOrNil("Jones")}
	if err := ctx.db.Create(&contact2).Error; err != nil {
		t.Fatalf("Create contact2 failed: %v", err)
	}

	post, err := ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title:     "Post with contacts",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create post failed: %v", err)
	}

	updated, err := ctx.svc.Update(post.ID, ctx.journalID, ctx.vaultID, dto.UpdatePostRequest{
		Title:      "Post with contacts",
		ContactIDs: []string{contact1.ID, contact2.ID},
	})
	if err != nil {
		t.Fatalf("Update with contacts failed: %v", err)
	}
	if len(updated.Contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(updated.Contacts))
	}

	updated2, err := ctx.svc.Update(post.ID, ctx.journalID, ctx.vaultID, dto.UpdatePostRequest{
		Title:      "Post cleared contacts",
		ContactIDs: []string{},
	})
	if err != nil {
		t.Fatalf("Update with empty contacts failed: %v", err)
	}
	if len(updated2.Contacts) != 0 {
		t.Errorf("Expected 0 contacts after clearing, got %d", len(updated2.Contacts))
	}
}

func TestPostContactIDsFromSectionsPrefersInlineMentions(t *testing.T) {
	markerID := "550e8400-e29b-41d4-a716-446655440000"
	legacyID := "550e8400-e29b-41d4-a716-446655440001"
	sections := []dto.PostSectionInput{{Content: "Dinner with @[Alice](contact:" + markerID + ")"}}

	got := postContactIDsFromSections(sections, []string{legacyID})
	if len(got) != 1 || got[0] != markerID {
		t.Fatalf("inline marker must be authoritative, got %v", got)
	}
	got = postContactIDsFromSections([]dto.PostSectionInput{{Content: "Legacy text"}}, []string{legacyID})
	if len(got) != 1 || got[0] != legacyID {
		t.Fatalf("legacy explicit association must remain supported, got %v", got)
	}
}

func TestPostResponseIncludesContactHoverCardDetails(t *testing.T) {
	firstName := "Alice"
	nickname := "Ace"
	jobPosition := "Designer"
	lastTalkedTo := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	response := toPostResponse(&models.Post{Contacts: []models.Contact{{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		FirstName:    &firstName,
		Nickname:     &nickname,
		JobPosition:  &jobPosition,
		LastTalkedTo: &lastTalkedTo,
	}}})

	if len(response.Contacts) != 1 {
		t.Fatalf("contacts=%+v", response.Contacts)
	}
	got := response.Contacts[0]
	if got.Nickname != nickname || got.JobPosition != jobPosition {
		t.Fatalf("hover card details=%+v", got)
	}
	if got.LastTalkedTo == nil || !got.LastTalkedTo.Equal(lastTalkedTo) {
		t.Fatalf("last_talked_to=%v", got.LastTalkedTo)
	}
}
