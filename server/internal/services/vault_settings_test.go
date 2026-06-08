package services

import (
	"errors"
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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault", Description: "desc"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewVaultSettingsService(db), vault.ID, resp.User.ID
}

func TestVaultSettingsGet(t *testing.T) {
	svc, vaultID, userID := setupVaultSettingsTest(t)

	settings, err := svc.Get(vaultID, userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if settings.Name != "Test Vault" {
		t.Errorf("Expected name 'Test Vault', got '%s'", settings.Name)
	}
	if !settings.ShowGroupTab {
		t.Error("Expected ShowGroupTab to be true by default")
	}
	if settings.NameOrder != nil {
		t.Errorf("Expected name_order override to be nil, got %q", *settings.NameOrder)
	}
	if settings.EffectiveNameOrder != "%first_name% %last_name%" {
		t.Errorf("Expected effective_name_order to fall back to user preference, got %q", settings.EffectiveNameOrder)
	}
}

func TestVaultSettingsUpdate(t *testing.T) {
	svc, vaultID, userID := setupVaultSettingsTest(t)

	settings, err := svc.Update(vaultID, userID, dto.UpdateVaultSettingsRequest{Name: "Updated", Description: "new desc"})
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
	svc, vaultID, userID := setupVaultSettingsTest(t)

	f := false
	settings, err := svc.UpdateVisibility(vaultID, userID, dto.UpdateTabVisibilityRequest{ShowGroupTab: &f})
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
	svc, vaultID, userID := setupVaultSettingsTest(t)

	tplID := uint(42)
	settings, err := svc.UpdateDefaultTemplate(vaultID, userID, dto.UpdateDefaultTemplateRequest{DefaultTemplateID: &tplID})
	if err != nil {
		t.Fatalf("UpdateDefaultTemplate failed: %v", err)
	}
	if settings.DefaultTemplateID == nil || *settings.DefaultTemplateID != 42 {
		t.Error("Expected DefaultTemplateID to be 42")
	}
}

func TestVaultSettingsNotFound(t *testing.T) {
	svc, _, _ := setupVaultSettingsTest(t)

	_, err := svc.Get("nonexistent", "missing-user")
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound, got %v", err)
	}
}

func TestVaultSettingsNameOrderFallsBackToUserPreference(t *testing.T) {
	svc, vaultID, userID := setupVaultSettingsTest(t)

	if err := NewPreferenceService(svc.db).UpdateNameOrder(userID, dto.UpdateNameOrderRequest{NameOrder: "%last_name%, %first_name%"}); err != nil {
		t.Fatalf("UpdateNameOrder failed: %v", err)
	}

	settings, err := svc.Get(vaultID, userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if settings.NameOrder != nil {
		t.Errorf("Expected nil vault override, got %q", *settings.NameOrder)
	}
	if settings.EffectiveNameOrder != "%last_name%, %first_name%" {
		t.Errorf("Expected effective_name_order from user preference, got %q", settings.EffectiveNameOrder)
	}
}

func TestVaultSettingsUpdateNameOrderOverride(t *testing.T) {
	svc, vaultID, userID := setupVaultSettingsTest(t)

	override := "%first_name% %last_name%{nickname? (%nickname%)}"
	settings, err := svc.UpdateNameOrder(vaultID, userID, dto.UpdateVaultNameOrderRequest{NameOrder: &override})
	if err != nil {
		t.Fatalf("UpdateNameOrder failed: %v", err)
	}
	if settings.NameOrder == nil || *settings.NameOrder != override {
		t.Fatalf("Expected name_order override %q, got %v", override, settings.NameOrder)
	}
	if settings.EffectiveNameOrder != override {
		t.Errorf("Expected effective_name_order override %q, got %q", override, settings.EffectiveNameOrder)
	}
}

func TestVaultSettingsClearNameOrderFallsBack(t *testing.T) {
	svc, vaultID, userID := setupVaultSettingsTest(t)

	fallback := "%last_name%, %first_name%"
	if err := NewPreferenceService(svc.db).UpdateNameOrder(userID, dto.UpdateNameOrderRequest{NameOrder: fallback}); err != nil {
		t.Fatalf("UpdateNameOrder failed: %v", err)
	}
	override := "%nickname%"
	if _, err := svc.UpdateNameOrder(vaultID, userID, dto.UpdateVaultNameOrderRequest{NameOrder: &override}); err != nil {
		t.Fatalf("UpdateNameOrder override failed: %v", err)
	}

	settings, err := svc.UpdateNameOrder(vaultID, userID, dto.UpdateVaultNameOrderRequest{NameOrder: nil})
	if err != nil {
		t.Fatalf("UpdateNameOrder clear failed: %v", err)
	}
	if settings.NameOrder != nil {
		t.Errorf("Expected cleared name_order override, got %q", *settings.NameOrder)
	}
	if settings.EffectiveNameOrder != fallback {
		t.Errorf("Expected fallback effective_name_order %q, got %q", fallback, settings.EffectiveNameOrder)
	}
}

func TestVaultSettingsRejectsInvalidNameOrderOverride(t *testing.T) {
	svc, vaultID, userID := setupVaultSettingsTest(t)

	invalid := "%first_name%{unknown? %nickname%}"
	_, err := svc.UpdateNameOrder(vaultID, userID, dto.UpdateVaultNameOrderRequest{NameOrder: &invalid})
	if !errors.Is(err, ErrInvalidNameOrder) {
		t.Errorf("Expected ErrInvalidNameOrder, got %v", err)
	}
}
