package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultMoodParamTest(t *testing.T) (*VaultMoodParamService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-mood-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultMoodParamService(db), vault.ID
}

func TestVaultMoodParamCRUD(t *testing.T) {
	svc, vaultID := setupVaultMoodParamTest(t)

	params, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	seedCount := len(params)

	pos := 10
	created, err := svc.Create(vaultID, dto.CreateMoodTrackingParameterRequest{Label: "Happy", HexColor: "#00FF00", Position: &pos})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Label != "Happy" {
		t.Errorf("Expected label 'Happy', got '%s'", created.Label)
	}

	pos2 := 20
	updated, err := svc.Update(created.ID, vaultID, dto.UpdateMoodTrackingParameterRequest{Label: "Very Happy", HexColor: "#00FF00", Position: &pos2})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Very Happy" {
		t.Errorf("Expected label 'Very Happy', got '%s'", updated.Label)
	}

	posUpdated, err := svc.UpdatePosition(created.ID, vaultID, 5)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}
	if posUpdated.Position == nil || *posUpdated.Position != 5 {
		t.Error("Expected position to be 5")
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	params, _ = svc.List(vaultID)
	if len(params) != seedCount {
		t.Errorf("Expected %d params after delete, got %d", seedCount, len(params))
	}
}

func TestVaultMoodParamNotFound(t *testing.T) {
	svc, vaultID := setupVaultMoodParamTest(t)

	_, err := svc.UpdatePosition(9999, vaultID, 1)
	if err != ErrMoodParamNotFound {
		t.Errorf("Expected ErrMoodParamNotFound, got %v", err)
	}
}
