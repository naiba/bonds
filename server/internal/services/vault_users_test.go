package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultUsersTest(t *testing.T) (*VaultUsersService, *AuthService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-users-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewVaultUsersService(db), authSvc, vault.ID, resp.User.ID, resp.User.AccountID
}

func TestVaultUsersListInitial(t *testing.T) {
	svc, _, vaultID, _, _ := setupVaultUsersTest(t)

	users, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user (creator), got %d", len(users))
	}
}

func TestVaultUsersAdd(t *testing.T) {
	svc, authSvc, vaultID, _, _ := setupVaultUsersTest(t)

	_, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Other",
		LastName:  "User",
		Email:     "other-user@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	user, err := svc.Add(vaultID, dto.AddVaultUserRequest{Email: "other-user@example.com", Permission: 200})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if user.Email != "other-user@example.com" {
		t.Errorf("Expected email 'other-user@example.com', got '%s'", user.Email)
	}
	if user.Permission != 200 {
		t.Errorf("Expected permission 200, got %d", user.Permission)
	}
}

func TestVaultUsersAddDuplicate(t *testing.T) {
	svc, _, vaultID, _, _ := setupVaultUsersTest(t)

	_, err := svc.Add(vaultID, dto.AddVaultUserRequest{Email: "vault-users-test@example.com", Permission: 200})
	if err != ErrUserAlreadyInVault {
		t.Errorf("Expected ErrUserAlreadyInVault, got %v", err)
	}
}

func TestVaultUsersAddEmailNotFound(t *testing.T) {
	svc, _, vaultID, _, _ := setupVaultUsersTest(t)

	_, err := svc.Add(vaultID, dto.AddVaultUserRequest{Email: "nonexistent@example.com", Permission: 200})
	if err != ErrUserEmailNotFound {
		t.Errorf("Expected ErrUserEmailNotFound, got %v", err)
	}
}

func TestVaultUsersUpdatePermission(t *testing.T) {
	svc, authSvc, vaultID, _, _ := setupVaultUsersTest(t)

	_, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Other", LastName: "User",
		Email: "other2@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	added, err := svc.Add(vaultID, dto.AddVaultUserRequest{Email: "other2@example.com", Permission: 300})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	updated, err := svc.UpdatePermission(added.ID, vaultID, dto.UpdateVaultUserPermRequest{Permission: 200})
	if err != nil {
		t.Fatalf("UpdatePermission failed: %v", err)
	}
	if updated.Permission != 200 {
		t.Errorf("Expected permission 200, got %d", updated.Permission)
	}
}

func TestVaultUsersRemoveCannotRemoveSelf(t *testing.T) {
	svc, _, vaultID, userID, _ := setupVaultUsersTest(t)

	users, _ := svc.List(vaultID)
	for _, u := range users {
		if u.UserID == userID {
			err := svc.Remove(u.ID, vaultID, userID)
			if err != ErrCannotRemoveSelf {
				t.Errorf("Expected ErrCannotRemoveSelf, got %v", err)
			}
			return
		}
	}
	t.Fatal("Creator not found in vault users")
}
