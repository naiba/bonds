package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultQuickFactTplTest(t *testing.T) (*VaultQuickFactTemplateService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-qft-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultQuickFactTemplateService(db), vault.ID
}

func TestVaultQuickFactTemplateCRUD(t *testing.T) {
	svc, vaultID := setupVaultQuickFactTplTest(t)

	tpls, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	seedCount := len(tpls)

	pos := 5
	created, err := svc.Create(vaultID, dto.CreateQuickFactTemplateRequest{Label: "Hobbies", Position: &pos})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Label != "Hobbies" {
		t.Errorf("Expected label 'Hobbies', got '%s'", created.Label)
	}

	pos2 := 10
	updated, err := svc.Update(created.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Sports", Position: &pos2})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Sports" {
		t.Errorf("Expected label 'Sports', got '%s'", updated.Label)
	}

	posUpdated, err := svc.UpdatePosition(created.ID, vaultID, 1)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}
	if posUpdated.Position != 1 {
		t.Errorf("Expected position 1, got %d", posUpdated.Position)
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	tpls, _ = svc.List(vaultID)
	if len(tpls) != seedCount {
		t.Errorf("Expected %d templates after delete, got %d", seedCount, len(tpls))
	}
}

func TestVaultQuickFactTemplateNotFound(t *testing.T) {
	svc, vaultID := setupVaultQuickFactTplTest(t)

	err := svc.Delete(9999, vaultID)
	if err != ErrQuickFactTplNotFound {
		t.Errorf("Expected ErrQuickFactTplNotFound, got %v", err)
	}
}
