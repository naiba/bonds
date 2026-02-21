package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupQuickFactToggleTest(t *testing.T) (*QuickFactService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "quick-facts-toggle-test@example.com",
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

	return NewQuickFactService(db), contact.ID, vault.ID
}

func TestToggleShowQuickFacts(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactToggleTest(t)

	newVal, err := svc.ToggleShowQuickFacts(contactID, vaultID)
	if err != nil {
		t.Fatalf("ToggleShowQuickFacts failed: %v", err)
	}
	if !newVal {
		t.Error("Expected ShowQuickFacts to be true after first toggle")
	}

	newVal, err = svc.ToggleShowQuickFacts(contactID, vaultID)
	if err != nil {
		t.Fatalf("ToggleShowQuickFacts failed: %v", err)
	}
	if newVal {
		t.Error("Expected ShowQuickFacts to be false after second toggle")
	}
}

func TestToggleShowQuickFactsVerifyPersistence(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "quick-facts-persist@example.com",
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

	svc := NewQuickFactService(db)

	_, err = svc.ToggleShowQuickFacts(contact.ID, vault.ID)
	if err != nil {
		t.Fatalf("ToggleShowQuickFacts failed: %v", err)
	}

	var c models.Contact
	if err := db.Where("id = ?", contact.ID).First(&c).Error; err != nil {
		t.Fatalf("Query contact failed: %v", err)
	}
	if !c.ShowQuickFacts {
		t.Error("Expected ShowQuickFacts to be true in DB")
	}
}

func TestToggleShowQuickFactsContactNotFound(t *testing.T) {
	svc, _, vaultID := setupQuickFactToggleTest(t)

	_, err := svc.ToggleShowQuickFacts("non-existent-contact-id", vaultID)
	if err == nil {
		t.Error("Expected error for non-existent contact")
	}
}

func TestToggleShowQuickFactsWrongVault(t *testing.T) {
	svc, contactID, _ := setupQuickFactToggleTest(t)

	_, err := svc.ToggleShowQuickFacts(contactID, "non-existent-vault-id")
	if err == nil {
		t.Error("Expected error for wrong vault ID")
	}
}
