package services

import (
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

func setupPostTest(t *testing.T) (*PostService, uint) {
	t.Helper()
	ctx := setupPostTestFull(t)
	return ctx.svc, ctx.journalID
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
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
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
	svc, journalID := setupPostTest(t)

	post, err := svc.Create(journalID, dto.CreatePostRequest{
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

func TestListPosts(t *testing.T) {
	svc, journalID := setupPostTest(t)

	_, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:     "Post 1",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(journalID, dto.CreatePostRequest{
		Title:     "Post 2",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	posts, err := svc.List(journalID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("Expected 2 posts, got %d", len(posts))
	}
}

func TestGetPost(t *testing.T) {
	svc, journalID := setupPostTest(t)

	created, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:     "Get Me",
		WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{
			{Position: 1, Label: "Section", Content: "Content"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.Get(created.ID, journalID)
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
	svc, journalID := setupPostTest(t)

	created, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:     "Original",
		WrittenAt: time.Now(),
		Sections: []dto.PostSectionInput{
			{Position: 1, Label: "Old", Content: "Old content"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, journalID, dto.UpdatePostRequest{
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
	svc, journalID := setupPostTest(t)

	created, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:     "To delete",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, journalID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	posts, err := svc.List(journalID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("Expected 0 posts after delete, got %d", len(posts))
	}
}

func TestPostNotFound(t *testing.T) {
	svc, journalID := setupPostTest(t)

	_, err := svc.Get(9999, journalID)
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}

	_, err = svc.Update(9999, journalID, dto.UpdatePostRequest{Title: "nope"})
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}

	err = svc.Delete(9999, journalID)
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}
}

func TestPostGetIncrementsViewCount(t *testing.T) {
	svc, journalID := setupPostTest(t)

	created, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:     "View Count Test",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	baseCount := created.ViewCount

	got1, err := svc.Get(created.ID, journalID)
	if err != nil {
		t.Fatalf("Get #1 failed: %v", err)
	}
	if got1.ViewCount != baseCount+1 {
		t.Errorf("Expected view_count %d after first Get, got %d", baseCount+1, got1.ViewCount)
	}

	got2, err := svc.Get(created.ID, journalID)
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

	post, err := ctx.svc.Create(ctx.journalID, dto.CreatePostRequest{
		Title:     "Post with contacts",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create post failed: %v", err)
	}

	updated, err := ctx.svc.Update(post.ID, ctx.journalID, dto.UpdatePostRequest{
		Title:      "Post with contacts",
		ContactIDs: []string{contact1.ID, contact2.ID},
	})
	if err != nil {
		t.Fatalf("Update with contacts failed: %v", err)
	}
	if len(updated.Contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(updated.Contacts))
	}

	updated2, err := ctx.svc.Update(post.ID, ctx.journalID, dto.UpdatePostRequest{
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
