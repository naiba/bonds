package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func TestAddressesByCity(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "rpt-city-test@example.com", Password: "password123",
	})
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})

	reportSvc := NewReportService(db)
	items, err := reportSvc.AddressesByCity(vault.ID, "Tokyo")
	if err != nil {
		t.Fatalf("AddressesByCity failed: %v", err)
	}
	if items == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items for non-existent city, got %d", len(items))
	}
}

func TestAddressesByCountry(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "rpt-country-test@example.com", Password: "password123",
	})
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})

	reportSvc := NewReportService(db)
	items, err := reportSvc.AddressesByCountry(vault.ID, "Japan")
	if err != nil {
		t.Fatalf("AddressesByCountry failed: %v", err)
	}
	if items == nil {
		t.Fatal("Expected non-nil result")
	}
}
