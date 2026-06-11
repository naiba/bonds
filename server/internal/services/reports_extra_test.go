package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
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
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	reportSvc := NewReportService(db)
	items, err := reportSvc.AddressesByCity(vault.ID, "Tokyo", resp.User.ID)
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
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	reportSvc := NewReportService(db)
	items, err := reportSvc.AddressesByCountry(vault.ID, "Japan", resp.User.ID)
	if err != nil {
		t.Fatalf("AddressesByCountry failed: %v", err)
	}
	if items == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items for non-existent country, got %d", len(items))
	}
}

func TestAddressesByCityUsesVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "reports-address-city-name-order@example.com")
	city := "Tokyo"
	province := "Tokyo"
	country := "JP"
	address := models.Address{VaultID: ctx.vaultID, City: &city, Province: &province, Country: &country}
	if err := ctx.db.Create(&address).Error; err != nil {
		t.Fatalf("Create address failed: %v", err)
	}
	if err := ctx.db.Model(&address).Association("Contacts").Append(&models.Contact{ID: ctx.contact.ID}); err != nil {
		t.Fatalf("Append contact to address failed: %v", err)
	}

	items, err := NewReportService(ctx.db).AddressesByCity(ctx.vaultID, city, ctx.userID)
	if err != nil {
		t.Fatalf("AddressesByCity failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].ContactName != "Zephyr, Alice (Ace)" {
		t.Fatalf("Expected contact_name 'Zephyr, Alice (Ace)', got '%s'", items[0].ContactName)
	}
}

func TestAddressesByCountryUsesVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "reports-address-country-name-order@example.com")
	country := "JP"
	city := "Tokyo"
	province := "Tokyo"
	address := models.Address{VaultID: ctx.vaultID, City: &city, Province: &province, Country: &country}
	if err := ctx.db.Create(&address).Error; err != nil {
		t.Fatalf("Create address failed: %v", err)
	}
	if err := ctx.db.Model(&address).Association("Contacts").Append(&models.Contact{ID: ctx.contact.ID}); err != nil {
		t.Fatalf("Append contact to address failed: %v", err)
	}

	items, err := NewReportService(ctx.db).AddressesByCountry(ctx.vaultID, country, ctx.userID)
	if err != nil {
		t.Fatalf("AddressesByCountry failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].ContactName != "Zephyr, Alice (Ace)" {
		t.Fatalf("Expected contact_name 'Zephyr, Alice (Ace)', got '%s'", items[0].ContactName)
	}
}
