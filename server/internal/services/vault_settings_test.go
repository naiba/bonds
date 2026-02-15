package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultSettingsTest(t *testing.T) (*VaultSettingsService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-settings-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault", Description: "desc"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewVaultSettingsService(db), vault.ID, resp.User.ID
}

func TestVaultSettingsGet(t *testing.T) {
	svc, vaultID, _ := setupVaultSettingsTest(t)

	settings, err := svc.Get(vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if settings.Name != "Test Vault" {
		t.Errorf("Expected name 'Test Vault', got '%s'", settings.Name)
	}
	if !settings.ShowGroupTab {
		t.Error("Expected ShowGroupTab to be true by default")
	}
}

func TestVaultSettingsUpdate(t *testing.T) {
	svc, vaultID, _ := setupVaultSettingsTest(t)

	settings, err := svc.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Updated", Description: "new desc"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if settings.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", settings.Name)
	}
	if settings.Description != "new desc" {
		t.Errorf("Expected description 'new desc', got '%s'", settings.Description)
	}
}

func TestVaultSettingsUpdateVisibility(t *testing.T) {
	svc, vaultID, _ := setupVaultSettingsTest(t)

	f := false
	settings, err := svc.UpdateVisibility(vaultID, dto.UpdateTabVisibilityRequest{ShowGroupTab: &f})
	if err != nil {
		t.Fatalf("UpdateVisibility failed: %v", err)
	}
	if settings.ShowGroupTab {
		t.Error("Expected ShowGroupTab to be false after update")
	}
	if !settings.ShowTasksTab {
		t.Error("Expected ShowTasksTab to remain true")
	}
}

func TestVaultSettingsUpdateDefaultTemplate(t *testing.T) {
	svc, vaultID, _ := setupVaultSettingsTest(t)

	tplID := uint(42)
	settings, err := svc.UpdateDefaultTemplate(vaultID, dto.UpdateDefaultTemplateRequest{DefaultTemplateID: &tplID})
	if err != nil {
		t.Fatalf("UpdateDefaultTemplate failed: %v", err)
	}
	if settings.DefaultTemplateID == nil || *settings.DefaultTemplateID != 42 {
		t.Error("Expected DefaultTemplateID to be 42")
	}
}

func TestVaultSettingsNotFound(t *testing.T) {
	svc, _, _ := setupVaultSettingsTest(t)

	_, err := svc.Get("nonexistent")
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound, got %v", err)
	}
}
