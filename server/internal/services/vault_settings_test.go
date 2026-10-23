package services

import (
	"errors"
	"github.com/naiba/bonds/internal/models"
	"reflect"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultSettingsTest(t *testing.T) (*VaultSettingsService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(db)

	registered, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-settings-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(registered.User.AccountID, registered.User.ID, dto.CreateVaultRequest{
		Name: "Test Vault", Description: "desc",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultSettingsService(db), vault.ID
}

func TestVaultSettingsGetContainsOnlyVaultConfiguration(t *testing.T) {
	svc, vaultID := setupVaultSettingsTest(t)
	settings, err := svc.Get(vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if settings.Name != "Test Vault" || !settings.ShowGroupTab {
		t.Fatalf("unexpected vault settings: %+v", settings)
	}
	if settings.DefaultTemplateID == nil {
		t.Fatal("vault must have a default contact layout")
	}
}

func TestVaultSettingsUpdate(t *testing.T) {
	svc, vaultID := setupVaultSettingsTest(t)
	settings, err := svc.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Updated", Description: "new desc"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if settings.Name != "Updated" || settings.Description != "new desc" {
		t.Fatalf("unexpected updated settings: %+v", settings)
	}
}

func TestVaultSettingsUpdateVisibility(t *testing.T) {
	svc, vaultID := setupVaultSettingsTest(t)
	hidden := false
	settings, err := svc.UpdateVisibility(vaultID, dto.UpdateTabVisibilityRequest{ShowGroupTab: &hidden})
	if err != nil {
		t.Fatalf("UpdateVisibility failed: %v", err)
	}
	if settings.ShowGroupTab || !settings.ShowTasksTab {
		t.Fatalf("unexpected visibility settings: %+v", settings)
	}
}

func TestVaultSettingsNotFound(t *testing.T) {
	svc, _ := setupVaultSettingsTest(t)
	if _, err := svc.Get("nonexistent"); err != ErrVaultNotFound {
		t.Fatalf("expected ErrVaultNotFound, got %v", err)
	}
}

func TestVaultSettingsCardDAVFilterPersistence(t *testing.T) {
	svc, vaultID := setupVaultSettingsTest(t)
	initial, err := svc.Get(vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if initial.CardDAVFilter != nil {
		t.Fatal("new vault should not filter contacts")
	}
	filter := &models.CardDAVFilter{Match: "any", Fields: []string{"phone", "email"}}
	if _, err = svc.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Book", CardDAVFilter: filter}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.CardDAVFilter, filter) {
		t.Fatalf("filter not persisted: %+v", got.CardDAVFilter)
	}
	if _, err = svc.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	got, err = svc.Get(vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.CardDAVFilter, filter) {
		t.Fatal("legacy update erased filter")
	}
	for _, invalid := range []models.CardDAVFilter{{Match: "unknown", Fields: []string{"phone"}}, {Match: "all", Fields: []string{"secret"}}, {Match: "all", Fields: []string{"phone", "phone"}}, {Match: "any", Fields: make([]string, 9)}} {
		if _, err = svc.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Invalid", CardDAVFilter: &invalid}); !errors.Is(err, ErrInvalidCardDAVFilter) {
			t.Fatalf("invalid filter accepted: %v", err)
		}
	}
	got, err = svc.Get(vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Renamed" || !reflect.DeepEqual(got.CardDAVFilter, filter) {
		t.Fatal("invalid update changed settings")
	}
	if _, err = svc.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Book", CardDAVFilter: &models.CardDAVFilter{Match: "all"}}); err != nil {
		t.Fatal(err)
	}
	got, err = svc.Get(vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CardDAVFilter != nil {
		t.Fatal("empty fields did not clear filter")
	}
}
