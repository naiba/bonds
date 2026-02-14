package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupNoteTest(t *testing.T) (*NoteService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "note-test@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewNoteService(db), contact.ID, vault.ID, resp.User.ID
}

func TestCreateNote(t *testing.T) {
	svc, contactID, vaultID, userID := setupNoteTest(t)

	note, err := svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{
		Title: "Test Note",
		Body:  "This is a test note body.",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if note.Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got '%s'", note.Title)
	}
	if note.Body != "This is a test note body." {
		t.Errorf("Expected body 'This is a test note body.', got '%s'", note.Body)
	}
	if note.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, note.ContactID)
	}
	if note.ID == 0 {
		t.Error("Expected note ID to be non-zero")
	}
}

func TestListNotes(t *testing.T) {
	svc, contactID, vaultID, userID := setupNoteTest(t)

	_, err := svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{Body: "Note 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{Body: "Note 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	notes, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(notes))
	}
}

func TestUpdateNote(t *testing.T) {
	svc, contactID, vaultID, userID := setupNoteTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{Body: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, dto.UpdateNoteRequest{
		Title: "Updated Title",
		Body:  "Updated body",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", updated.Title)
	}
	if updated.Body != "Updated body" {
		t.Errorf("Expected body 'Updated body', got '%s'", updated.Body)
	}
}

func TestDeleteNote(t *testing.T) {
	svc, contactID, vaultID, userID := setupNoteTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{Body: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	notes, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("Expected 0 notes after delete, got %d", len(notes))
	}
}

func TestDeleteNoteNotFound(t *testing.T) {
	svc, contactID, _, _ := setupNoteTest(t)

	err := svc.Delete(9999, contactID)
	if err != ErrNoteNotFound {
		t.Errorf("Expected ErrNoteNotFound, got %v", err)
	}
}

func TestUpdateNoteNotFound(t *testing.T) {
	svc, contactID, _, _ := setupNoteTest(t)

	_, err := svc.Update(9999, contactID, dto.UpdateNoteRequest{Body: "nope"})
	if err != ErrNoteNotFound {
		t.Errorf("Expected ErrNoteNotFound, got %v", err)
	}
}
