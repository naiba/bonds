package mcp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/testutil"
)

type allowVaultChecker struct {
	allowedVault string
}

func (c allowVaultChecker) CheckUserVaultAccess(_ string, vaultID string, _ int) error {
	if vaultID != c.allowedVault {
		return errors.New("forbidden")
	}
	return nil
}

type fakeSearchService struct{}

func (fakeSearchService) Search(vaultID, query string, page, perPage int) (*search.SearchResponse, error) {
	return &search.SearchResponse{
		Contacts: []search.SearchResult{{ID: "bleve-contact", Type: "contact", Name: "Bleve Contact", Score: 1}},
		Notes:    []search.SearchResult{},
		Total:    1,
	}, nil
}

func TestBondsSearcherScopesToVault(t *testing.T) {
	db := testutil.SetupTestDB(t)
	first := "Alice"
	last := "Smith"
	contact := models.Contact{VaultID: "vault-a", FirstName: &first, LastName: &last, Listed: true}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}
	otherFirst := "Alice"
	other := models.Contact{VaultID: "vault-b", FirstName: &otherFirst, Listed: true}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other contact: %v", err)
	}

	searcher := NewBondsSearcher(db, fakeSearchService{}, allowVaultChecker{allowedVault: "vault-a"})
	result, err := searcher.Search("user-1", SearchBondsArgs{VaultID: "vault-a", Query: "Alice"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	foundVaultA := false
	foundVaultB := false
	for _, item := range result.Results {
		if item.ID == contact.ID {
			foundVaultA = true
		}
		if item.ID == other.ID {
			foundVaultB = true
		}
	}
	if !foundVaultA {
		t.Fatal("expected vault-a contact in results")
	}
	if foundVaultB {
		t.Fatal("must not return contact from inaccessible vault")
	}
	if result.Capabilities.SemanticVectorSearch {
		t.Fatal("vector search must be disabled")
	}
}

func TestBondsSearcherDeniesInaccessibleVault(t *testing.T) {
	db := testutil.SetupTestDB(t)
	searcher := NewBondsSearcher(db, fakeSearchService{}, allowVaultChecker{allowedVault: "vault-a"})
	if _, err := searcher.Search("user-1", SearchBondsArgs{VaultID: "vault-b", Query: "Alice"}); err == nil {
		t.Fatal("expected access error")
	}
}

func TestBondsSearcherSkipsNotesForUnlistedContacts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	listedName := "Listed"
	listedContact := models.Contact{VaultID: "vault-a", FirstName: &listedName, Listed: true}
	if err := db.Create(&listedContact).Error; err != nil {
		t.Fatalf("failed to create listed contact: %v", err)
	}
	shadowName := "Shadow"
	shadowContact := models.Contact{VaultID: "vault-a", FirstName: &shadowName}
	if err := db.Create(&shadowContact).Error; err != nil {
		t.Fatalf("failed to create shadow contact: %v", err)
	}
	if err := db.Model(&shadowContact).Update("listed", false).Error; err != nil {
		t.Fatalf("failed to mark contact unlisted: %v", err)
	}

	listedTitle := "Secret Plan"
	listedNote := models.Note{VaultID: "vault-a", ContactID: listedContact.ID, Title: &listedTitle, Body: "visible secret"}
	if err := db.Create(&listedNote).Error; err != nil {
		t.Fatalf("failed to create listed note: %v", err)
	}
	shadowTitle := "Secret Shadow"
	shadowNote := models.Note{VaultID: "vault-a", ContactID: shadowContact.ID, Title: &shadowTitle, Body: "hidden secret"}
	if err := db.Create(&shadowNote).Error; err != nil {
		t.Fatalf("failed to create shadow note: %v", err)
	}

	searcher := NewBondsSearcher(db, nil, allowVaultChecker{allowedVault: "vault-a"})
	result, err := searcher.Search("user-1", SearchBondsArgs{VaultID: "vault-a", Query: "Secret"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	foundListed := false
	for _, item := range result.Results {
		if item.Type == "note" && item.ID == fmt.Sprint(listedNote.ID) {
			foundListed = true
		}
		if item.Type == "note" && item.ID == fmt.Sprint(shadowNote.ID) {
			t.Fatal("must not return note attached to unlisted contact")
		}
	}
	if !foundListed {
		t.Fatal("expected listed note in results")
	}
}

func TestBondsSearcherSkipsTasksOnlyAssignedToUnlistedContacts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	listedName := "Listed"
	listedContact := models.Contact{VaultID: "vault-a", FirstName: &listedName, Listed: true}
	if err := db.Create(&listedContact).Error; err != nil {
		t.Fatalf("failed to create listed contact: %v", err)
	}
	shadowName := "Shadow"
	shadowContact := models.Contact{VaultID: "vault-a", FirstName: &shadowName}
	if err := db.Create(&shadowContact).Error; err != nil {
		t.Fatalf("failed to create shadow contact: %v", err)
	}
	if err := db.Model(&shadowContact).Update("listed", false).Error; err != nil {
		t.Fatalf("failed to mark contact unlisted: %v", err)
	}

	listedTask := models.ContactTask{VaultID: "vault-a", Label: "Secret Listed Task", AuthorName: "tester"}
	if err := db.Create(&listedTask).Error; err != nil {
		t.Fatalf("failed to create listed task: %v", err)
	}
	if err := db.Create(&models.TaskContact{ContactTaskID: listedTask.ID, ContactID: listedContact.ID}).Error; err != nil {
		t.Fatalf("failed to assign listed task: %v", err)
	}
	shadowTask := models.ContactTask{VaultID: "vault-a", Label: "Secret Shadow Task", AuthorName: "tester"}
	if err := db.Create(&shadowTask).Error; err != nil {
		t.Fatalf("failed to create shadow task: %v", err)
	}
	if err := db.Create(&models.TaskContact{ContactTaskID: shadowTask.ID, ContactID: shadowContact.ID}).Error; err != nil {
		t.Fatalf("failed to assign shadow task: %v", err)
	}
	standaloneTask := models.ContactTask{VaultID: "vault-a", Label: "Secret Standalone Task", AuthorName: "tester"}
	if err := db.Create(&standaloneTask).Error; err != nil {
		t.Fatalf("failed to create standalone task: %v", err)
	}

	searcher := NewBondsSearcher(db, nil, allowVaultChecker{allowedVault: "vault-a"})
	result, err := searcher.Search("user-1", SearchBondsArgs{VaultID: "vault-a", Query: "Secret"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	foundListed := false
	foundStandalone := false
	for _, item := range result.Results {
		if item.Type == "task" && item.ID == fmt.Sprint(listedTask.ID) {
			foundListed = true
		}
		if item.Type == "task" && item.ID == fmt.Sprint(standaloneTask.ID) {
			foundStandalone = true
		}
		if item.Type == "task" && item.ID == fmt.Sprint(shadowTask.ID) {
			t.Fatal("must not return task assigned only to unlisted contacts")
		}
	}
	if !foundListed {
		t.Fatal("expected task assigned to listed contact in results")
	}
	if !foundStandalone {
		t.Fatal("expected standalone vault task in results")
	}
}
