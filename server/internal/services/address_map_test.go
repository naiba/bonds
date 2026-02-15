package services

import (
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupAddressMapTest(t *testing.T) (*AddressService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "address-map-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
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

func TestGetMapImageURL(t *testing.T) {
	svc, contactID, vaultID := setupAddressMapTest(t)

	addr, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1:   "123 Main St",
		City:    "Springfield",
		Country: "US",
	})
	if err != nil {
		t.Fatalf("Create address failed: %v", err)
	}

	lat := 40.7128
	lng := -74.0060
	if err := svc.db.Model(&models.Address{}).Where("id = ?", addr.ID).
		Updates(map[string]interface{}{"latitude": lat, "longitude": lng}).Error; err != nil {
		t.Fatalf("Update coordinates failed: %v", err)
	}

	url, err := svc.GetMapImageURL(addr.ID, contactID, vaultID, 600, 400)
	if err != nil {
		t.Fatalf("GetMapImageURL failed: %v", err)
	}

	if !strings.Contains(url, "openstreetmap.org") {
		t.Errorf("Expected URL to contain 'openstreetmap.org', got '%s'", url)
	}
	if !strings.Contains(url, "40.7") {
		t.Errorf("Expected URL to contain latitude, got '%s'", url)
	}
	if !strings.Contains(url, "-74.0") {
		t.Errorf("Expected URL to contain longitude, got '%s'", url)
	}
}

func TestGetMapImageURLNoCoordinates(t *testing.T) {
	svc, contactID, vaultID := setupAddressMapTest(t)

	addr, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "No Geocode Address",
	})
	if err != nil {
		t.Fatalf("Create address failed: %v", err)
	}

	_, err = svc.GetMapImageURL(addr.ID, contactID, vaultID, 600, 400)
	if err == nil {
		t.Error("Expected error for address with no coordinates")
	}
	if !strings.Contains(err.Error(), "no coordinates") {
		t.Errorf("Expected 'no coordinates' error, got '%s'", err.Error())
	}
}

func TestGetMapImageURLAddressNotFound(t *testing.T) {
	svc, contactID, vaultID := setupAddressMapTest(t)

	_, err := svc.GetMapImageURL(9999, contactID, vaultID, 600, 400)
	if err != ErrAddressNotFound {
		t.Errorf("Expected ErrAddressNotFound, got %v", err)
	}
}

func TestGetMapImageURLWrongVault(t *testing.T) {
	svc, contactID, vaultID := setupAddressMapTest(t)

	addr, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "Wrong Vault Address",
	})
	if err != nil {
		t.Fatalf("Create address failed: %v", err)
	}

	_, err = svc.GetMapImageURL(addr.ID, contactID, "non-existent-vault-id", 600, 400)
	if err == nil {
		t.Error("Expected error for wrong vault ID")
	}
}
