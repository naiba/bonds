package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactReligionTest(t *testing.T) (*ContactReligionService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "religion-test@example.com",
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

	return NewContactReligionService(db), contact.ID, vault.ID
}

func TestContactReligionUpdate(t *testing.T) {
	svc, contactID, vaultID := setupContactReligionTest(t)

	religionID := uint(1)
	resp, err := svc.Update(contactID, vaultID, dto.UpdateContactReligionRequest{ReligionID: &religionID})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactReligionUpdateClear(t *testing.T) {
	svc, contactID, vaultID := setupContactReligionTest(t)

	religionID := uint(1)
	_, err := svc.Update(contactID, vaultID, dto.UpdateContactReligionRequest{ReligionID: &religionID})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	resp, err := svc.Update(contactID, vaultID, dto.UpdateContactReligionRequest{ReligionID: nil})
	if err != nil {
		t.Fatalf("Update to nil failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactReligionUpdateNotFound(t *testing.T) {
	svc, _, vaultID := setupContactReligionTest(t)

	religionID := uint(1)
	_, err := svc.Update("nonexistent-id", vaultID, dto.UpdateContactReligionRequest{ReligionID: &religionID})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}
