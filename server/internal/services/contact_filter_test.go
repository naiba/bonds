package services

import (
	"testing"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupContactFilterTest(t *testing.T) (*ContactService, *gorm.DB, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "filter-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewContactService(db), db, vault.ID, resp.User.ID
}

func TestListContacts_FilterActive(t *testing.T) {
	svc, _, vaultID, userID := setupContactFilterTest(t)

	// Create 2 active contacts
	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Create 1 archived contact
	c3, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.ToggleArchive(c3.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}

	// Default filter (empty) should return only active
	contacts, meta, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 active contacts with default filter, got %d", len(contacts))
	}
	if meta.Total != 2 {
		t.Errorf("Expected total 2, got %d", meta.Total)
	}

	// Explicit "active" filter should also return only active
	contacts2, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "active")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts2) != 2 {
		t.Errorf("Expected 2 active contacts with 'active' filter, got %d", len(contacts2))
	}
}

func TestListContacts_FilterArchived(t *testing.T) {
	svc, _, vaultID, userID := setupContactFilterTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	c3, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.ToggleArchive(c3.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}

	contacts, meta, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "archived")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("Expected 1 archived contact, got %d", len(contacts))
	}
	if meta.Total != 1 {
		t.Errorf("Expected total 1, got %d", meta.Total)
	}
	if len(contacts) > 0 && contacts[0].FirstName != "Charlie" {
		t.Errorf("Expected archived contact 'Charlie', got '%s'", contacts[0].FirstName)
	}
}

func TestListContacts_FilterAll(t *testing.T) {
	svc, _, vaultID, userID := setupContactFilterTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	c3, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.ToggleArchive(c3.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}

	contacts, meta, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "all")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 3 {
		t.Errorf("Expected 3 contacts with 'all' filter, got %d", len(contacts))
	}
	if meta.Total != 3 {
		t.Errorf("Expected total 3, got %d", meta.Total)
	}
}

func TestListContacts_FilterFavorites(t *testing.T) {
	svc, _, vaultID, userID := setupContactFilterTest(t)

	c1, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Favorite Alice
	_, err = svc.ToggleFavorite(c1.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}

	contacts, meta, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "favorites")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("Expected 1 favorite contact, got %d", len(contacts))
	}
	if meta.Total != 1 {
		t.Errorf("Expected total 1, got %d", meta.Total)
	}
	if len(contacts) > 0 && contacts[0].FirstName != "Alice" {
		t.Errorf("Expected favorite contact 'Alice', got '%s'", contacts[0].FirstName)
	}
}

func TestListContacts_FavoritesFirst(t *testing.T) {
	svc, _, vaultID, userID := setupContactFilterTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	c2, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Favorite Bob
	_, err = svc.ToggleFavorite(c2.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}

	// Sort by first_name — Bob should still be first because favorites come first
	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 3 {
		t.Fatalf("Expected 3 contacts, got %d", len(contacts))
	}
	if contacts[0].FirstName != "Bob" {
		t.Errorf("Expected first contact 'Bob' (favorite), got '%s'", contacts[0].FirstName)
	}
	if !contacts[0].IsFavorite {
		t.Error("Expected first contact to be marked as favorite")
	}
	// Non-favorites should be sorted alphabetically
	if contacts[1].FirstName != "Alice" {
		t.Errorf("Expected second contact 'Alice', got '%s'", contacts[1].FirstName)
	}
	if contacts[2].FirstName != "Charlie" {
		t.Errorf("Expected third contact 'Charlie', got '%s'", contacts[2].FirstName)
	}
}

func TestListContacts_FilterArchivedExcludesFavorites(t *testing.T) {
	svc, _, vaultID, userID := setupContactFilterTest(t)

	c1, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	// Favorite and archive Alice
	_, err = svc.ToggleFavorite(c1.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	_, err = svc.ToggleArchive(c1.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}

	// favorites filter should not return archived contacts
	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "favorites")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("Expected 0 favorite contacts (archived ones excluded), got %d", len(contacts))
	}

	// archived filter should return the archived contact
	contactsArchived, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "archived")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contactsArchived) != 1 {
		t.Errorf("Expected 1 archived contact, got %d", len(contactsArchived))
	}
}

func TestListContactsByLabel_FilterActive(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	contactSvc := NewContactService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "clf-filter@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	label := models.Label{VaultID: vault.ID, Name: "Test", Slug: "test"}
	if err := db.Create(&label).Error; err != nil {
		t.Fatalf("Create label failed: %v", err)
	}

	// Create active contact with label
	c1, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Active"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	db.Create(&models.ContactLabel{LabelID: label.ID, ContactID: c1.ID})

	// Create archived contact with label
	c2, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Archived"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	db.Create(&models.ContactLabel{LabelID: label.ID, ContactID: c2.ID})
	_, err = contactSvc.ToggleArchive(c2.ID, vault.ID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}

	// Default filter → only active
	contacts, meta, err := contactSvc.ListContactsByLabel(vault.ID, resp.User.ID, label.ID, 1, 15, "", "")
	if err != nil {
		t.Fatalf("ListContactsByLabel failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("Expected 1 active contact, got %d", len(contacts))
	}
	if meta.Total != 1 {
		t.Errorf("Expected total 1, got %d", meta.Total)
	}

	// All filter → both
	contacts2, meta2, err := contactSvc.ListContactsByLabel(vault.ID, resp.User.ID, label.ID, 1, 15, "", "all")
	if err != nil {
		t.Fatalf("ListContactsByLabel failed: %v", err)
	}
	if len(contacts2) != 2 {
		t.Errorf("Expected 2 contacts with 'all' filter, got %d", len(contacts2))
	}
	if meta2.Total != 2 {
		t.Errorf("Expected total 2, got %d", meta2.Total)
	}
}