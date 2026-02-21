package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type companyTestContext struct {
	svc     *CompanyService
	db      *gorm.DB
	vaultID string
	company *models.Company
}

func setupCompanyTest(t *testing.T) *companyTestContext {
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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
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

	return &companyTestContext{
		svc:     NewCompanyService(db),
		db:      db,
		vaultID: vault.ID,
		company: company,
	}
}

func TestCompanyList(t *testing.T) {
	ctx := setupCompanyTest(t)

	companies, err := ctx.svc.List(ctx.vaultID)
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
	ctx := setupCompanyTest(t)

	got, err := ctx.svc.Get(ctx.company.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("Expected name 'Acme Corp', got '%s'", got.Name)
	}
	if got.ID != ctx.company.ID {
		t.Errorf("Expected ID %d, got %d", ctx.company.ID, got.ID)
	}
}

func TestCompanyGetNotFound(t *testing.T) {
	ctx := setupCompanyTest(t)

	_, err := ctx.svc.Get(9999)
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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Empty Vault"}, "en")
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

func TestCompanyCreate(t *testing.T) {
	ctx := setupCompanyTest(t)

	company, err := ctx.svc.Create(ctx.vaultID, dto.CreateCompanyRequest{
		Name: "New Corp",
		Type: "startup",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if company.Name != "New Corp" {
		t.Errorf("Expected name 'New Corp', got '%s'", company.Name)
	}
	if company.Type != "startup" {
		t.Errorf("Expected type 'startup', got '%s'", company.Type)
	}
	if company.ID == 0 {
		t.Error("Expected company ID to be non-zero")
	}
	if company.VaultID != ctx.vaultID {
		t.Errorf("Expected vaultID '%s', got '%s'", ctx.vaultID, company.VaultID)
	}

	companies, err := ctx.svc.List(ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(companies) != 2 {
		t.Errorf("Expected 2 companies after create, got %d", len(companies))
	}
}

func TestCompanyCreateWithoutType(t *testing.T) {
	ctx := setupCompanyTest(t)

	company, err := ctx.svc.Create(ctx.vaultID, dto.CreateCompanyRequest{
		Name: "No Type Corp",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if company.Name != "No Type Corp" {
		t.Errorf("Expected name 'No Type Corp', got '%s'", company.Name)
	}
	if company.Type != "" {
		t.Errorf("Expected empty type, got '%s'", company.Type)
	}
}

func TestCompanyUpdate(t *testing.T) {
	ctx := setupCompanyTest(t)

	updated, err := ctx.svc.Update(ctx.company.ID, ctx.vaultID, dto.UpdateCompanyRequest{
		Name: "Updated Corp",
		Type: "enterprise",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated Corp" {
		t.Errorf("Expected name 'Updated Corp', got '%s'", updated.Name)
	}
	if updated.Type != "enterprise" {
		t.Errorf("Expected type 'enterprise', got '%s'", updated.Type)
	}
}

func TestCompanyUpdateNotFound(t *testing.T) {
	ctx := setupCompanyTest(t)

	_, err := ctx.svc.Update(9999, ctx.vaultID, dto.UpdateCompanyRequest{
		Name: "Ghost",
	})
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestCompanyUpdateWrongVault(t *testing.T) {
	ctx := setupCompanyTest(t)

	_, err := ctx.svc.Update(ctx.company.ID, "wrong-vault-id", dto.UpdateCompanyRequest{
		Name: "Wrong Vault",
	})
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestCompanyDelete(t *testing.T) {
	ctx := setupCompanyTest(t)

	err := ctx.svc.Delete(ctx.company.ID, ctx.vaultID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	companies, err := ctx.svc.List(ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(companies) != 0 {
		t.Errorf("Expected 0 companies after delete, got %d", len(companies))
	}
}

func TestCompanyDeleteNotFound(t *testing.T) {
	ctx := setupCompanyTest(t)

	err := ctx.svc.Delete(9999, ctx.vaultID)
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestCompanyDeleteWrongVault(t *testing.T) {
	ctx := setupCompanyTest(t)

	err := ctx.svc.Delete(ctx.company.ID, "wrong-vault-id")
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestCompanyDeleteUnlinksContacts(t *testing.T) {
	ctx := setupCompanyTest(t)

	contact := models.Contact{
		VaultID:   ctx.vaultID,
		FirstName: strPtrOrNil("John"),
		LastName:  strPtrOrNil("Doe"),
		CompanyID: &ctx.company.ID,
	}
	if err := ctx.db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	var before models.Contact
	ctx.db.First(&before, "id = ?", contact.ID)
	if before.CompanyID == nil {
		t.Fatal("Expected contact to have CompanyID before delete")
	}

	err := ctx.svc.Delete(ctx.company.ID, ctx.vaultID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	var after models.Contact
	ctx.db.First(&after, "id = ?", contact.ID)
	if after.CompanyID != nil {
		t.Errorf("Expected contact CompanyID to be nil after company delete, got %v", *after.CompanyID)
	}
}
