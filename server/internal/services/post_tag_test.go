package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPostTagTest(t *testing.T) (*PostTagService, uint, uint, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "pt-test@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	journalSvc := NewJournalService(db)
	journal, _ := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})

	postSvc := NewPostService(db)
	post, _ := postSvc.Create(journal.ID, dto.CreatePostRequest{Title: "Test Post", WrittenAt: time.Now()})

	return NewPostTagService(db), post.ID, journal.ID, vault.ID
}

func TestAddPostTagWithNewName(t *testing.T) {
	svc, postID, journalID, vaultID := setupPostTagTest(t)

	tag, err := svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{Name: "travel"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if tag.Name != "travel" {
		t.Errorf("Expected name 'travel', got '%s'", tag.Name)
	}
	if tag.Slug != "travel" {
		t.Errorf("Expected slug 'travel', got '%s'", tag.Slug)
	}
}

func TestAddPostTagWithExistingID(t *testing.T) {
	svc, postID, journalID, vaultID := setupPostTagTest(t)

	tag1, _ := svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{Name: "food"})

	tag2, err := svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{TagID: tag1.ID})
	if err != nil {
		t.Fatalf("Add with existing TagID failed: %v", err)
	}
	if tag2.ID != tag1.ID {
		t.Errorf("Expected same tag ID %d, got %d", tag1.ID, tag2.ID)
	}
}

func TestRemovePostTag(t *testing.T) {
	svc, postID, journalID, vaultID := setupPostTagTest(t)

	tag, _ := svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{Name: "temp"})
	if err := svc.Remove(tag.ID, postID, journalID); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestListPostTags(t *testing.T) {
	svc, postID, journalID, vaultID := setupPostTagTest(t)

	svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{Name: "alpha"})
	svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{Name: "beta"})

	tags, err := svc.List(postID, journalID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestListPostTagsNotFound(t *testing.T) {
	svc, _, journalID, _ := setupPostTagTest(t)

	_, err := svc.List(99999, journalID)
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}
}

func TestUpdatePostTag(t *testing.T) {
	svc, postID, journalID, vaultID := setupPostTagTest(t)

	tag, _ := svc.Add(postID, journalID, vaultID, dto.AddPostTagRequest{Name: "old"})

	updated, err := svc.Update(tag.ID, postID, journalID, vaultID, dto.UpdatePostTagRequest{Name: "new"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "new" {
		t.Errorf("Expected name 'new', got '%s'", updated.Name)
	}
}
