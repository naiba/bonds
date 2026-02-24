package services

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func strPtr(s string) *string {
	return &s
}

func TestSearchService_RebuildIndex_EmptyDatabase(t *testing.T) {
	db := testutil.SetupTestDB(t)
	searchSvc := NewSearchService(&search.NoopEngine{})

	contactCount, noteCount, err := searchSvc.RebuildIndex(db)
	if err != nil {
		t.Fatalf("RebuildIndex failed: %v", err)
	}

	if contactCount != 0 {
		t.Errorf("Expected 0 contacts, got %d", contactCount)
	}
	if noteCount != 0 {
		t.Errorf("Expected 0 notes, got %d", noteCount)
	}
}

func TestSearchService_RebuildIndex_WithContacts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	searchSvc := NewSearchService(&search.NoopEngine{})

	// Create test vault and contacts
	vault := models.Vault{
		ID:        "vault-test-1",
		AccountID: "account-test-1",
		Type:      "personal",
		Name:      "Test Vault",
	}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	// Create multiple contacts
	contacts := []models.Contact{
		{
			ID:        "contact-1",
			VaultID:   "vault-test-1",
			FirstName: strPtr("John"),
			LastName:  strPtr("Doe"),
			Listed:    true,
		},
		{
			ID:        "contact-2",
			VaultID:   "vault-test-1",
			FirstName: strPtr("Jane"),
			LastName:  strPtr("Smith"),
			Listed:    true,
		},
		{
			ID:        "contact-3",
			VaultID:   "vault-test-1",
			FirstName: strPtr("Bob"),
			LastName:  strPtr("Johnson"),
			Listed:    true,
		},
	}
	for _, contact := range contacts {
		if err := db.Create(&contact).Error; err != nil {
			t.Fatalf("Failed to create contact: %v", err)
		}
	}

	contactCount, noteCount, err := searchSvc.RebuildIndex(db)
	if err != nil {
		t.Fatalf("RebuildIndex failed: %v", err)
	}

	if contactCount != 3 {
		t.Errorf("Expected 3 contacts, got %d", contactCount)
	}
	if noteCount != 0 {
		t.Errorf("Expected 0 notes, got %d", noteCount)
	}
}

func TestSearchService_RebuildIndex_WithNotes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	searchSvc := NewSearchService(&search.NoopEngine{})

	// Create test vault and contact
	vault := models.Vault{
		ID:        "vault-test-2",
		AccountID: "account-test-2",
		Type:      "personal",
		Name:      "Test Vault",
	}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	contact := models.Contact{
		ID:        "contact-notes-1",
		VaultID:   "vault-test-2",
		FirstName: strPtr("Alice"),
		LastName:  strPtr("Wonder"),
		Listed:    true,
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("Failed to create contact: %v", err)
	}

	// Create multiple notes
	notes := []models.Note{
		{
			ContactID: "contact-notes-1",
			VaultID:   "vault-test-2",
			Title:     strPtr("First Note"),
			Body:      "This is the first note content",
		},
		{
			ContactID: "contact-notes-1",
			VaultID:   "vault-test-2",
			Title:     strPtr("Second Note"),
			Body:      "This is the second note content",
		},
		{
			ContactID: "contact-notes-1",
			VaultID:   "vault-test-2",
			Title:     strPtr("Third Note"),
			Body:      "This is the third note content",
		},
	}
	for _, note := range notes {
		if err := db.Create(&note).Error; err != nil {
			t.Fatalf("Failed to create note: %v", err)
		}
	}

	contactCount, noteCount, err := searchSvc.RebuildIndex(db)
	if err != nil {
		t.Fatalf("RebuildIndex failed: %v", err)
	}

	if contactCount != 1 {
		t.Errorf("Expected 1 contact, got %d", contactCount)
	}
	if noteCount != 3 {
		t.Errorf("Expected 3 notes, got %d", noteCount)
	}
}

func TestSearchService_RebuildIndex_WithContactsAndNotes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	searchSvc := NewSearchService(&search.NoopEngine{})

	// Create test vault
	vault := models.Vault{
		ID:        "vault-test-3",
		AccountID: "account-test-3",
		Type:      "personal",
		Name:      "Test Vault",
	}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	// Create multiple contacts
	for i := 1; i <= 2; i++ {
		contact := models.Contact{
			ID:        "contact-mixed-" + string(rune(i+48)),
			VaultID:   "vault-test-3",
			FirstName: strPtr("Contact"),
			LastName:  strPtr("Test"),
			Listed:    true,
		}
		if err := db.Create(&contact).Error; err != nil {
			t.Fatalf("Failed to create contact: %v", err)
		}

		// Create notes for each contact
		for j := 1; j <= 2; j++ {
			note := models.Note{
				ContactID: contact.ID,
				VaultID:   "vault-test-3",
				Title:     strPtr("Note for contact"),
				Body:      "Note content",
			}
			if err := db.Create(&note).Error; err != nil {
				t.Fatalf("Failed to create note: %v", err)
			}
		}
	}

	contactCount, noteCount, err := searchSvc.RebuildIndex(db)
	if err != nil {
		t.Fatalf("RebuildIndex failed: %v", err)
	}

	if contactCount != 2 {
		t.Errorf("Expected 2 contacts, got %d", contactCount)
	}
	if noteCount != 4 {
		t.Errorf("Expected 4 notes, got %d", noteCount)
	}
}

func TestSearchService_RebuildIndex_EngineFailure(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create a mock engine that fails on Rebuild
	failingEngine := &failingSearchEngine{}
	searchSvc := NewSearchService(failingEngine)

	_, _, err := searchSvc.RebuildIndex(db)
	if err == nil {
		t.Fatal("Expected RebuildIndex to fail when engine.Rebuild() fails")
	}
}

// failingSearchEngine is a test double that fails on Rebuild
type failingSearchEngine struct{}

func (e *failingSearchEngine) IndexContact(id, vaultID, firstName, lastName, nickname, jobPosition string) error {
	return nil
}

func (e *failingSearchEngine) IndexNote(id string, vaultID, contactID, title, body string) error {
	return nil
}

func (e *failingSearchEngine) DeleteDocument(id string) error {
	return nil
}

func (e *failingSearchEngine) Search(vaultID, query string, limit, offset int) (*search.SearchResponse, error) {
	return &search.SearchResponse{}, nil
}

func (e *failingSearchEngine) Rebuild() error {
	return gorm.ErrInvalidDB
}

func (e *failingSearchEngine) Close() error {
	return nil
}
