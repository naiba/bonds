package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactFeedTest(t *testing.T) (*FeedService, string, string) {
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
	feedRecorder := NewFeedRecorder(db)
	contactSvc.SetFeedRecorder(feedRecorder)

	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewFeedService(db), contact.ID, vault.ID
}

func TestContactFeedList(t *testing.T) {
	svc, contactID, _ := setupContactFeedTest(t)

	items, meta, err := svc.ListContactFeed(contactID, 1, 15)
	if err != nil {
		t.Fatalf("ListContactFeed failed: %v", err)
	}
	if len(items) < 1 {
		t.Errorf("Expected at least 1 feed item (from contact creation), got %d", len(items))
	}
	if meta.Page != 1 {
		t.Errorf("Expected page 1, got %d", meta.Page)
	}
}

func TestContactFeedListEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewFeedService(db)

	items, _, err := svc.ListContactFeed("nonexistent-contact", 1, 15)
	if err != nil {
		t.Fatalf("ListContactFeed failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 feed items, got %d", len(items))
	}
}
