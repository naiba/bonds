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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
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

	notes, _, err := svc.List(contactID, vaultID, 1, 15)
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

	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateNoteRequest{
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

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	notes, _, err := svc.List(contactID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("Expected 0 notes after delete, got %d", len(notes))
	}
}

func TestCreateNoteWithEmotion(t *testing.T) {
	svc, contactID, vaultID, userID := setupNoteTest(t)

	var eid uint
	svc.db.Raw("SELECT id FROM emotions LIMIT 1").Scan(&eid)
	if eid == 0 {
		t.Fatal("Expected at least one seeded emotion")
	}

	note, err := svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{
		Body:      "Note with emotion",
		EmotionID: &eid,
	})
	if err != nil {
		t.Fatalf("Create with emotion failed: %v", err)
	}
	if note.EmotionID == nil || *note.EmotionID != eid {
		t.Errorf("Expected emotion_id %d, got %v", eid, note.EmotionID)
	}

	updated, err := svc.Update(note.ID, contactID, vaultID, dto.UpdateNoteRequest{
		Body:      "Updated note",
		EmotionID: nil,
	})
	if err != nil {
		t.Fatalf("Update emotion to nil failed: %v", err)
	}
	if updated.EmotionID != nil {
		t.Errorf("Expected emotion_id nil after update, got %v", updated.EmotionID)
	}
}

func TestNoteListPagination(t *testing.T) {
	svc, contactID, vaultID, userID := setupNoteTest(t)

	for i := 0; i < 3; i++ {
		_, err := svc.Create(contactID, vaultID, userID, dto.CreateNoteRequest{Body: "Note body"})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	notes, meta, err := svc.List(contactID, vaultID, 1, 2)
	if err != nil {
		t.Fatalf("List page 1 failed: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("Expected 2 notes on page 1, got %d", len(notes))
	}
	if meta.Total != 3 {
		t.Errorf("Expected total 3, got %d", meta.Total)
	}
	if meta.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", meta.TotalPages)
	}
	if meta.Page != 1 {
		t.Errorf("Expected page 1, got %d", meta.Page)
	}
	if meta.PerPage != 2 {
		t.Errorf("Expected per_page 2, got %d", meta.PerPage)
	}

	notes2, meta2, err := svc.List(contactID, vaultID, 2, 2)
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}
	if len(notes2) != 1 {
		t.Errorf("Expected 1 note on page 2, got %d", len(notes2))
	}
	if meta2.Total != 3 {
		t.Errorf("Expected total 3 on page 2, got %d", meta2.Total)
	}
}

func TestNoteNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupNoteTest(t)

	_, err := svc.Update(9999, contactID, vaultID, dto.UpdateNoteRequest{Body: "nope"})
	if err != ErrNoteNotFound {
		t.Errorf("Update: expected ErrNoteNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID, vaultID)
	if err != ErrNoteNotFound {
		t.Errorf("Delete: expected ErrNoteNotFound, got %v", err)
	}
}
