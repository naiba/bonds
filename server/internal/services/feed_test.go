package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupFeedTest(t *testing.T) (*FeedService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "feed-test@example.com",
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

	return NewFeedService(db), vault.ID, contact.ID
}

func TestFeedEmpty(t *testing.T) {
	svc, vaultID, _ := setupFeedTest(t)

	items, meta, err := svc.GetFeed(vaultID, 1, 15)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 feed items, got %d", len(items))
	}
	if meta.Total != 0 {
		t.Errorf("Expected total 0, got %d", meta.Total)
	}
}

func TestFeedWithItems(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "feed-items@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	desc := "Created contact"
	item := &models.ContactFeedItem{
		ContactID:   contact.ID,
		AuthorID:    &resp.User.ID,
		Action:      "contact_created",
		Description: &desc,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("Create feed item failed: %v", err)
	}

	svc := NewFeedService(db)
	items, meta, err := svc.GetFeed(vault.ID, 1, 15)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 feed item, got %d", len(items))
	}
	if meta.Total != 1 {
		t.Errorf("Expected total 1, got %d", meta.Total)
	}
	if len(items) > 0 && items[0].Action != "contact_created" {
		t.Errorf("Expected action 'contact_created', got '%s'", items[0].Action)
	}
}

func TestFeedPagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "feed-pagination@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		item := &models.ContactFeedItem{
			ContactID: contact.ID,
			Action:    "note_created",
		}
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("Create feed item failed: %v", err)
		}
	}

	svc := NewFeedService(db)
	items, meta, err := svc.GetFeed(vault.ID, 1, 2)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 feed items on page 1, got %d", len(items))
	}
	if meta.Total != 5 {
		t.Errorf("Expected total 5, got %d", meta.Total)
	}
	if meta.TotalPages != 3 {
		t.Errorf("Expected 3 total pages, got %d", meta.TotalPages)
	}

	items2, _, err := svc.GetFeed(vault.ID, 3, 2)
	if err != nil {
		t.Fatalf("GetFeed page 3 failed: %v", err)
	}
	if len(items2) != 1 {
		t.Errorf("Expected 1 feed item on page 3, got %d", len(items2))
	}
}

func TestFeedContactName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "feed-contactname@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{
		FirstName: "Alice",
		LastName:  "Wang",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	desc := "Test feed contact name"
	item := &models.ContactFeedItem{
		ContactID:   contact.ID,
		AuthorID:    &resp.User.ID,
		Action:      "note_created",
		Description: &desc,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("Create feed item failed: %v", err)
	}

	svc := NewFeedService(db)
	items, _, err := svc.GetFeed(vault.ID, 1, 15)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 feed item, got %d", len(items))
	}
	expectedName := "Alice Wang"
	if items[0].ContactName != expectedName {
		t.Errorf("Expected contact_name=%q, got %q", expectedName, items[0].ContactName)
	}

	// Also verify ListContactFeed returns the same contact_name
	items2, _, err := svc.ListContactFeed(contact.ID, 1, 15)
	if err != nil {
		t.Fatalf("ListContactFeed failed: %v", err)
	}
	if len(items2) != 1 {
		t.Fatalf("Expected 1 feed item from ListContactFeed, got %d", len(items2))
	}
	if items2[0].ContactName != expectedName {
		t.Errorf("ListContactFeed: expected contact_name=%q, got %q", expectedName, items2[0].ContactName)
	}
}
