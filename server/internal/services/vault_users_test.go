package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewVaultUsersService(db), authSvc, vault.ID, resp.User.ID, resp.User.AccountID
}

func registerSameAccountUser(t *testing.T, authSvc *AuthService, accountID, email string) {
	t.Helper()
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Other", LastName: "User",
		Email: email, Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register user failed: %v", err)
	}
	if err := authSvc.db.Model(&models.User{}).Where("id = ?", resp.User.ID).
		Update("account_id", accountID).Error; err != nil {
		t.Fatalf("Failed to reassign account: %v", err)
	}
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
	svc, authSvc, vaultID, _, accountID := setupVaultUsersTest(t)

	registerSameAccountUser(t, authSvc, accountID, "other-user@example.com")

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
	svc, authSvc, vaultID, _, accountID := setupVaultUsersTest(t)

	registerSameAccountUser(t, authSvc, accountID, "other2@example.com")

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

func TestVaultUsersUpdatePermissionRejectsSoleManagerDemotion(t *testing.T) {
	svc, _, vaultID, userID, _ := setupVaultUsersTest(t)

	users, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	manager := users[0]
	if manager.UserID != userID || manager.Permission != models.PermissionManager {
		t.Fatalf("initial membership = %+v, want creator manager", manager)
	}

	_, err = svc.UpdatePermission(manager.ID, vaultID, dto.UpdateVaultUserPermRequest{Permission: models.PermissionEditor})
	if !errors.Is(err, ErrLastVaultManager) {
		t.Fatalf("UpdatePermission error = %v, want ErrLastVaultManager", err)
	}

	users, err = svc.List(vaultID)
	if err != nil {
		t.Fatalf("List after rejected update failed: %v", err)
	}
	if users[0].Permission != models.PermissionManager {
		t.Fatalf("permission after rejected update = %d, want manager", users[0].Permission)
	}
}

func TestVaultUsersUpdatePermissionAlwaysRetainsOneManager(t *testing.T) {
	svc, authSvc, vaultID, ownerID, accountID := setupVaultUsersTest(t)
	registerSameAccountUser(t, authSvc, accountID, "second-manager@example.com")
	second, err := svc.Add(vaultID, dto.AddVaultUserRequest{
		Email: "second-manager@example.com", Permission: models.PermissionManager,
	})
	if err != nil {
		t.Fatalf("Add second manager failed: %v", err)
	}
	users, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	var ownerMembershipID uint
	for _, user := range users {
		if user.UserID == ownerID {
			ownerMembershipID = user.ID
		}
	}

	if _, err := svc.UpdatePermission(ownerMembershipID, vaultID, dto.UpdateVaultUserPermRequest{Permission: models.PermissionEditor}); err != nil {
		t.Fatalf("demote one of two managers: %v", err)
	}
	_, err = svc.UpdatePermission(second.ID, vaultID, dto.UpdateVaultUserPermRequest{Permission: models.PermissionViewer})
	if !errors.Is(err, ErrLastVaultManager) {
		t.Fatalf("demote final manager error = %v, want ErrLastVaultManager", err)
	}
}

func TestVaultUsersDisabledManagerDoesNotSatisfyManagerGuard(t *testing.T) {
	svc, authSvc, vaultID, ownerID, accountID := setupVaultUsersTest(t)
	registerSameAccountUser(t, authSvc, accountID, "disabled-manager@example.com")
	second, err := svc.Add(vaultID, dto.AddVaultUserRequest{
		Email: "disabled-manager@example.com", Permission: models.PermissionManager,
	})
	if err != nil {
		t.Fatalf("Add second manager failed: %v", err)
	}
	if err := svc.db.Model(&models.User{}).Where("id = ?", second.UserID).Update("disabled", true).Error; err != nil {
		t.Fatalf("disable second manager: %v", err)
	}
	users, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	var ownerMembershipID uint
	for _, user := range users {
		if user.UserID == ownerID {
			ownerMembershipID = user.ID
		}
		if user.UserID == second.UserID && !user.Disabled {
			t.Fatal("disabled manager was not marked disabled in the response")
		}
	}

	_, err = svc.UpdatePermission(ownerMembershipID, vaultID, dto.UpdateVaultUserPermRequest{Permission: models.PermissionEditor})
	if !errors.Is(err, ErrLastVaultManager) {
		t.Fatalf("demote only active manager error = %v, want ErrLastVaultManager", err)
	}
}

func TestVaultUsersRemoveRejectsSoleManager(t *testing.T) {
	svc, _, vaultID, ownerID, _ := setupVaultUsersTest(t)
	users, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	err = svc.Remove(users[0].ID, vaultID, "different-current-user")
	if !errors.Is(err, ErrLastVaultManager) {
		t.Fatalf("Remove sole manager error = %v, want ErrLastVaultManager", err)
	}
	remaining, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List after rejected removal failed: %v", err)
	}
	if len(remaining) != 1 || remaining[0].UserID != ownerID {
		t.Fatalf("remaining memberships = %+v, want owner", remaining)
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
