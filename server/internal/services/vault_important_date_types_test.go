package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultDateTypeTest(t *testing.T) (*VaultImportantDateTypeService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-datetype-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultImportantDateTypeService(db), vault.ID
}

func TestVaultDateTypeCRUD(t *testing.T) {
	svc, vaultID := setupVaultDateTypeTest(t)

	types, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	seedCount := len(types)

	created, err := svc.Create(vaultID, dto.CreateImportantDateTypeRequest{Label: "Anniversary"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Label != "Anniversary" {
		t.Errorf("Expected label 'Anniversary', got '%s'", created.Label)
	}
	if !created.CanBeDeleted {
		t.Error("Expected CanBeDeleted to be true for user-created type")
	}

	updated, err := svc.Update(created.ID, vaultID, dto.UpdateImportantDateTypeRequest{Label: "Wedding Anniversary"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Wedding Anniversary" {
		t.Errorf("Expected label 'Wedding Anniversary', got '%s'", updated.Label)
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	types, _ = svc.List(vaultID)
	if len(types) != seedCount {
		t.Errorf("Expected %d types after delete, got %d", seedCount, len(types))
	}
}

func TestVaultDateTypeCannotDeleteDefault(t *testing.T) {
	svc, vaultID := setupVaultDateTypeTest(t)

	types, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	for _, dt := range types {
		if !dt.CanBeDeleted {
			err := svc.Delete(dt.ID, vaultID)
			if err != ErrCannotDeleteDefault {
				t.Errorf("Expected ErrCannotDeleteDefault for seeded type, got %v", err)
			}
			return
		}
	}
}
