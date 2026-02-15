package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultTagTest(t *testing.T) (*VaultTagService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-tag-test@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultTagService(db), vault.ID
}

func TestVaultTagCRUD(t *testing.T) {
	svc, vaultID := setupVaultTagTest(t)

	created, err := svc.Create(vaultID, dto.CreateTagRequest{Name: "Work"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Slug != "work" {
		t.Errorf("Expected slug 'work', got '%s'", created.Slug)
	}

	updated, err := svc.Update(created.ID, vaultID, dto.UpdateTagRequest{Name: "Personal"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Slug != "personal" {
		t.Errorf("Expected slug 'personal', got '%s'", updated.Slug)
	}

	tags, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestVaultTagNotFound(t *testing.T) {
	svc, vaultID := setupVaultTagTest(t)

	_, err := svc.Update(9999, vaultID, dto.UpdateTagRequest{Name: "x"})
	if err != ErrTagNotFound {
		t.Errorf("Expected ErrTagNotFound, got %v", err)
	}
}
