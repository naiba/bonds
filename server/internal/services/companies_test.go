package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupCompanyTest(t *testing.T) (*CompanyService, string, *models.Company) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "companies-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	compType := "tech"
	company := &models.Company{
		VaultID: vault.ID,
		Name:    "Acme Corp",
		Type:    &compType,
	}
	if err := db.Create(company).Error; err != nil {
		t.Fatalf("Create company failed: %v", err)
	}

	return NewCompanyService(db), vault.ID, company
}

func TestCompanyList(t *testing.T) {
	svc, vaultID, _ := setupCompanyTest(t)

	companies, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(companies) != 1 {
		t.Errorf("Expected 1 company, got %d", len(companies))
	}
	if companies[0].Name != "Acme Corp" {
		t.Errorf("Expected name 'Acme Corp', got '%s'", companies[0].Name)
	}
	if companies[0].Type != "tech" {
		t.Errorf("Expected type 'tech', got '%s'", companies[0].Type)
	}
}

func TestCompanyGet(t *testing.T) {
	svc, _, company := setupCompanyTest(t)

	got, err := svc.Get(company.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("Expected name 'Acme Corp', got '%s'", got.Name)
	}
	if got.ID != company.ID {
		t.Errorf("Expected ID %d, got %d", company.ID, got.ID)
	}
}

func TestCompanyGetNotFound(t *testing.T) {
	svc, _, _ := setupCompanyTest(t)

	_, err := svc.Get(9999)
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestCompanyListEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "companies-empty@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Empty Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewCompanyService(db)
	companies, err := svc.List(vault.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(companies) != 0 {
		t.Errorf("Expected 0 companies, got %d", len(companies))
	}
}
