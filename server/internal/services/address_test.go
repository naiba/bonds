package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupAddressTest(t *testing.T) (*AddressService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "address-test@example.com",
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

	return NewAddressService(db), contact.ID, vault.ID
}

func TestCreateAddress(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)

	address, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1:   "123 Main St",
		City:    "Springfield",
		Country: "US",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if address.Line1 != "123 Main St" {
		t.Errorf("Expected line1 '123 Main St', got '%s'", address.Line1)
	}
	if address.City != "Springfield" {
		t.Errorf("Expected city 'Springfield', got '%s'", address.City)
	}
	if address.Country != "US" {
		t.Errorf("Expected country 'US', got '%s'", address.Country)
	}
	if address.VaultID != vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", vaultID, address.VaultID)
	}
	if address.ID == 0 {
		t.Error("Expected address ID to be non-zero")
	}
}

func TestListAddresses(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{Line1: "Address 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreateAddressRequest{Line1: "Address 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	addresses, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(addresses))
	}
}

func TestUpdateAddress(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "Original",
		City:  "OldCity",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "Updated",
		City:  "NewCity",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Line1 != "Updated" {
		t.Errorf("Expected line1 'Updated', got '%s'", updated.Line1)
	}
	if updated.City != "NewCity" {
		t.Errorf("Expected city 'NewCity', got '%s'", updated.City)
	}
}

func TestDeleteAddress(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{Line1: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	addresses, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(addresses) != 0 {
		t.Errorf("Expected 0 addresses after delete, got %d", len(addresses))
	}
}

func TestAddressNotFound(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)

	_, err := svc.Update(9999, contactID, vaultID, dto.UpdateAddressRequest{Line1: "nope"})
	if err != ErrAddressNotFound {
		t.Errorf("Expected ErrAddressNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID, vaultID)
	if err != ErrAddressNotFound {
		t.Errorf("Expected ErrAddressNotFound, got %v", err)
	}
}
