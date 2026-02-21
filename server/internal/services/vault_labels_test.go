package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultLabelTest(t *testing.T) (*VaultLabelService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-label-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultLabelService(db), vault.ID
}

func TestVaultLabelCRUD(t *testing.T) {
	svc, vaultID := setupVaultLabelTest(t)

	created, err := svc.Create(vaultID, dto.CreateLabelRequest{Name: "Important", BgColor: "bg-red-200", TextColor: "text-red-700"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Name != "Important" {
		t.Errorf("Expected name 'Important', got '%s'", created.Name)
	}
	if created.Slug != "important" {
		t.Errorf("Expected slug 'important', got '%s'", created.Slug)
	}

	updated, err := svc.Update(created.ID, vaultID, dto.UpdateLabelRequest{Name: "Very Important", BgColor: "bg-blue-200", TextColor: "text-blue-700"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Very Important" {
		t.Errorf("Expected name 'Very Important', got '%s'", updated.Name)
	}
	if updated.Slug != "very-important" {
		t.Errorf("Expected slug 'very-important', got '%s'", updated.Slug)
	}

	labels, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	labels, _ = svc.List(vaultID)
	if len(labels) != 0 {
		t.Errorf("Expected 0 labels after delete, got %d", len(labels))
	}
}

func TestVaultLabelNotFound(t *testing.T) {
	svc, vaultID := setupVaultLabelTest(t)

	_, err := svc.Update(9999, vaultID, dto.UpdateLabelRequest{Name: "x"})
	if err != ErrLabelNotFound {
		t.Errorf("Expected ErrLabelNotFound, got %v", err)
	}

	err = svc.Delete(9999, vaultID)
	if err != ErrLabelNotFound {
		t.Errorf("Expected ErrLabelNotFound, got %v", err)
	}
}
