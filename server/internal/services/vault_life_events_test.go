package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultLifeEventTest(t *testing.T) (*VaultLifeEventService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-lifeevent-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultLifeEventService(db), vault.ID
}

func TestVaultLifeEventCategoryCRUD(t *testing.T) {
	svc, vaultID := setupVaultLifeEventTest(t)

	cats, err := svc.ListCategories(vaultID)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	seedCount := len(cats)

	pos := 10
	created, err := svc.CreateCategory(vaultID, dto.CreateLifeEventCategoryRequest{Label: "Travel", Position: &pos})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if created.Label != "Travel" {
		t.Errorf("Expected label 'Travel', got '%s'", created.Label)
	}

	pos2 := 20
	updated, err := svc.UpdateCategory(created.ID, vaultID, dto.UpdateLifeEventCategoryRequest{Label: "Adventures", Position: &pos2})
	if err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}
	if updated.Label != "Adventures" {
		t.Errorf("Expected label 'Adventures', got '%s'", updated.Label)
	}

	if err := svc.DeleteCategory(created.ID, vaultID); err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	cats, _ = svc.ListCategories(vaultID)
	if len(cats) != seedCount {
		t.Errorf("Expected %d categories after delete, got %d", seedCount, len(cats))
	}
}

func TestVaultLifeEventTypeCRUD(t *testing.T) {
	svc, vaultID := setupVaultLifeEventTest(t)

	cat, err := svc.CreateCategory(vaultID, dto.CreateLifeEventCategoryRequest{Label: "Cat1"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	lt, err := svc.CreateType(cat.ID, vaultID, dto.CreateLifeEventTypeRequest{Label: "Type1"})
	if err != nil {
		t.Fatalf("CreateType failed: %v", err)
	}
	if lt.Label != "Type1" {
		t.Errorf("Expected label 'Type1', got '%s'", lt.Label)
	}

	updated, err := svc.UpdateType(lt.ID, cat.ID, vaultID, dto.UpdateLifeEventTypeRequest{Label: "Type1 Updated"})
	if err != nil {
		t.Fatalf("UpdateType failed: %v", err)
	}
	if updated.Label != "Type1 Updated" {
		t.Errorf("Expected label 'Type1 Updated', got '%s'", updated.Label)
	}

	if err := svc.DeleteType(lt.ID, cat.ID, vaultID); err != nil {
		t.Fatalf("DeleteType failed: %v", err)
	}
}
