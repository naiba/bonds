package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func TestUpdateDefaultTab(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "dt-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	if err := vaultSvc.UpdateDefaultTab(vault.ID, "notes"); err != nil {
		t.Fatalf("UpdateDefaultTab failed: %v", err)
	}

	got, err := vaultSvc.GetVault(vault.ID)
	if err != nil {
		t.Fatalf("GetVault failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetVault returned nil")
	}
}

func TestUpdateDefaultTabNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	vaultSvc := NewVaultService(db)

	err := vaultSvc.UpdateDefaultTab("nonexistent-id", "notes")
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound, got %v", err)
	}
}
