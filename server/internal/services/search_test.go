package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/testutil"
)

func TestSearchService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "search-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	dir := t.TempDir()
	bleveEngine, err := search.NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer bleveEngine.Close()

	searchSvc := NewSearchService(bleveEngine)
	contactSvc := NewContactService(db)
	contactSvc.SetSearchService(searchSvc)

	_, err = contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice", LastName: "Smith"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Bob", LastName: "Jones"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	result, err := searchSvc.Search(vault.ID, "Alice", 1, 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total == 0 {
		t.Fatal("Expected at least 1 result for 'Alice'")
	}

	found := false
	for _, r := range result.Results {
		if r.Type == "contact" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find a contact in search results")
	}
}

func TestSearchService_DeleteContact(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "search-delete@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	dir := t.TempDir()
	bleveEngine, err := search.NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer bleveEngine.Close()

	searchSvc := NewSearchService(bleveEngine)
	contactSvc := NewContactService(db)
	contactSvc.SetSearchService(searchSvc)

	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Charlie", LastName: "Brown"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	result, err := searchSvc.Search(vault.ID, "Charlie", 1, 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total == 0 {
		t.Fatal("Expected at least 1 result for 'Charlie' before deletion")
	}

	if err := contactSvc.DeleteContact(contact.ID, vault.ID); err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	result, err = searchSvc.Search(vault.ID, "Charlie", 1, 20)
	if err != nil {
		t.Fatalf("Search after delete failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Expected 0 results after deletion, got %d", result.Total)
	}
}

func TestSearchService_VaultIsolation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "search-isolation@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault1, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault One"})
	if err != nil {
		t.Fatalf("CreateVault 1 failed: %v", err)
	}
	vault2, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault Two"})
	if err != nil {
		t.Fatalf("CreateVault 2 failed: %v", err)
	}

	dir := t.TempDir()
	bleveEngine, err := search.NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer bleveEngine.Close()

	searchSvc := NewSearchService(bleveEngine)
	contactSvc := NewContactService(db)
	contactSvc.SetSearchService(searchSvc)

	_, err = contactSvc.CreateContact(vault1.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice", LastName: "InVaultOne"})
	if err != nil {
		t.Fatalf("CreateContact in vault1 failed: %v", err)
	}
	contact2, err := contactSvc.CreateContact(vault2.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice", LastName: "InVaultTwo"})
	if err != nil {
		t.Fatalf("CreateContact in vault2 failed: %v", err)
	}

	result, err := searchSvc.Search(vault1.ID, "Alice", 1, 20)
	if err != nil {
		t.Fatalf("Search vault1 failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected exactly 1 result in vault1, got %d", result.Total)
	}
	for _, r := range result.Results {
		if r.ID == contact2.ID {
			t.Error("Vault1 search returned vault2's contact — isolation violated")
		}
	}

	result, err = searchSvc.Search(vault2.ID, "Alice", 1, 20)
	if err != nil {
		t.Fatalf("Search vault2 failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected exactly 1 result in vault2, got %d", result.Total)
	}
}

func TestSearchService_NoteIndexing(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "search-note@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	dir := t.TempDir()
	bleveEngine, err := search.NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer bleveEngine.Close()

	searchSvc := NewSearchService(bleveEngine)
	contactSvc := NewContactService(db)
	contactSvc.SetSearchService(searchSvc)
	noteSvc := NewNoteService(db)
	noteSvc.SetSearchService(searchSvc)

	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Dave", LastName: "Notes"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	_, err = noteSvc.Create(contact.ID, vault.ID, resp.User.ID, dto.CreateNoteRequest{
		Title: "Meeting Summary",
		Body:  "Discussed project milestones and quarterly goals",
	})
	if err != nil {
		t.Fatalf("Create note failed: %v", err)
	}

	result, err := searchSvc.Search(vault.ID, "milestones", 1, 20)
	if err != nil {
		t.Fatalf("Search for note content failed: %v", err)
	}
	if result.Total == 0 {
		t.Fatal("Expected at least 1 result for note body content 'milestones'")
	}

	found := false
	for _, r := range result.Results {
		if r.Type == "note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find a note in search results")
	}

	result, err = searchSvc.Search(vault.ID, "Meeting Summary", 1, 20)
	if err != nil {
		t.Fatalf("Search for note title failed: %v", err)
	}
	if result.Total == 0 {
		t.Fatal("Expected at least 1 result for note title 'Meeting Summary'")
	}
}

func TestSearchService_EmptyQuery(t *testing.T) {
	dir := t.TempDir()
	bleveEngine, err := search.NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer bleveEngine.Close()

	searchSvc := NewSearchService(bleveEngine)

	result, err := searchSvc.Search("some-vault-id", "", 1, 20)
	if err != nil {
		return
	}
	if result.Total != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", result.Total)
	}
}
