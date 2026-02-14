package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupFeedRecorderTest(t *testing.T) (*FeedRecorder, *ContactService, *NoteService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "feed-recorder-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	fr := NewFeedRecorder(db)

	contactSvc := NewContactService(db)
	contactSvc.SetFeedRecorder(fr)

	noteSvc := NewNoteService(db)
	noteSvc.SetFeedRecorder(fr)

	return fr, contactSvc, noteSvc, vault.ID, resp.User.ID, resp.User.AccountID
}

func TestRecordFeedItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	fr := NewFeedRecorder(db)

	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "feed-record-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	entityID := uint(42)
	entityType := "Note"
	err = fr.Record(contact.ID, resp.User.ID, ActionNoteCreated, "Created a note", &entityID, &entityType)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	var items []models.ContactFeedItem
	if err := db.Where("contact_id = ?", contact.ID).Find(&items).Error; err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 feed item, got %d", len(items))
	}
	if items[0].Action != ActionNoteCreated {
		t.Errorf("Expected action '%s', got '%s'", ActionNoteCreated, items[0].Action)
	}
	if items[0].FeedableID == nil || *items[0].FeedableID != 42 {
		t.Errorf("Expected feedable_id 42, got %v", items[0].FeedableID)
	}
	if items[0].FeedableType == nil || *items[0].FeedableType != "Note" {
		t.Errorf("Expected feedable_type 'Note', got %v", items[0].FeedableType)
	}
}

func TestCreateContactRecordsFeed(t *testing.T) {
	_, contactSvc, _, vaultID, userID, _ := setupFeedRecorderTest(t)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	var items []models.ContactFeedItem
	if err := contactSvc.db.Where("contact_id = ? AND action = ?", contact.ID, ActionContactCreated).Find(&items).Error; err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 feed item for contact_created, got %d", len(items))
	}
	if items[0].AuthorID == nil || *items[0].AuthorID != userID {
		t.Errorf("Expected author_id '%s', got %v", userID, items[0].AuthorID)
	}
}

func TestCreateNoteRecordsFeed(t *testing.T) {
	_, contactSvc, noteSvc, vaultID, userID, _ := setupFeedRecorderTest(t)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	note, err := noteSvc.Create(contact.ID, vaultID, userID, dto.CreateNoteRequest{
		Title: "Test Note",
		Body:  "Test body",
	})
	if err != nil {
		t.Fatalf("Create note failed: %v", err)
	}

	var items []models.ContactFeedItem
	if err := noteSvc.db.Where("contact_id = ? AND action = ?", contact.ID, ActionNoteCreated).Find(&items).Error; err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 feed item for note_created, got %d", len(items))
	}
	if items[0].FeedableID == nil || *items[0].FeedableID != note.ID {
		t.Errorf("Expected feedable_id %d, got %v", note.ID, items[0].FeedableID)
	}
	if items[0].FeedableType == nil || *items[0].FeedableType != "Note" {
		t.Errorf("Expected feedable_type 'Note', got %v", items[0].FeedableType)
	}
}
