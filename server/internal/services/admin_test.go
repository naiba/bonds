package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupAdminTest(t *testing.T) (*AdminService, *AuthService, *VaultService) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	adminSvc := NewAdminService(db, t.TempDir())
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	return adminSvc, authSvc, vaultSvc
}

func registerTestUser(t *testing.T, authSvc *AuthService, email string) *dto.AuthResponse {
	t.Helper()
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     email,
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	return resp
}

func TestAdminListUsers(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)

	registerTestUser(t, authSvc, "admin-list1@example.com")
	registerTestUser(t, authSvc, "admin-list2@example.com")

	users, err := adminSvc.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Email != "admin-list1@example.com" {
		t.Errorf("expected first user email admin-list1@example.com, got %s", users[0].Email)
	}
	if users[1].Email != "admin-list2@example.com" {
		t.Errorf("expected second user email admin-list2@example.com, got %s", users[1].Email)
	}
}

func TestAdminListUsers_WithStats(t *testing.T) {
	adminSvc, authSvc, vaultSvc := setupAdminTest(t)

	resp := registerTestUser(t, authSvc, "stats-user@example.com")

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(adminSvc.db)
	_, err = contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Contact1"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Contact2"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	users, err := adminSvc.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].ContactCount != 2 {
		t.Errorf("expected contact count 2, got %d", users[0].ContactCount)
	}
	if users[0].VaultCount != 1 {
		t.Errorf("expected vault count 1, got %d", users[0].VaultCount)
	}
}

func TestAdminToggleUser_Success(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)

	admin := registerTestUser(t, authSvc, "toggle-admin@example.com")
	target := registerTestUser(t, authSvc, "toggle-target@example.com")

	err := adminSvc.ToggleUser(admin.User.ID, target.User.ID, true)
	if err != nil {
		t.Fatalf("ToggleUser failed: %v", err)
	}

	var user models.User
	if err := adminSvc.db.First(&user, "id = ?", target.User.ID).Error; err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if !user.Disabled {
		t.Error("expected user to be disabled")
	}

	err = adminSvc.ToggleUser(admin.User.ID, target.User.ID, false)
	if err != nil {
		t.Fatalf("ToggleUser (enable) failed: %v", err)
	}

	if err := adminSvc.db.First(&user, "id = ?", target.User.ID).Error; err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if user.Disabled {
		t.Error("expected user to be enabled")
	}
}

func TestAdminToggleUser_CannotDisableSelf(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	admin := registerTestUser(t, authSvc, "self-disable@example.com")

	err := adminSvc.ToggleUser(admin.User.ID, admin.User.ID, true)
	if err != ErrCannotDisableSelf {
		t.Errorf("expected ErrCannotDisableSelf, got %v", err)
	}
}

func TestAdminToggleUser_NotFound(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	admin := registerTestUser(t, authSvc, "toggle-nf@example.com")

	err := adminSvc.ToggleUser(admin.User.ID, "nonexistent-id", true)
	if err != ErrAdminUserNotFound {
		t.Errorf("expected ErrAdminUserNotFound, got %v", err)
	}
}

func TestAdminSetAdmin_Success(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)

	registerTestUser(t, authSvc, "setadmin-actor@example.com")
	target := registerTestUser(t, authSvc, "setadmin-target@example.com")

	var userBefore models.User
	adminSvc.db.First(&userBefore, "id = ?", target.User.ID)
	if userBefore.IsInstanceAdministrator {
		t.Error("expected target to not be instance admin initially")
	}

	err := adminSvc.SetAdmin("some-actor-id", target.User.ID, true)
	if err != nil {
		t.Fatalf("SetAdmin failed: %v", err)
	}

	var userAfter models.User
	adminSvc.db.First(&userAfter, "id = ?", target.User.ID)
	if !userAfter.IsInstanceAdministrator {
		t.Error("expected target to be instance admin after SetAdmin(true)")
	}

	err = adminSvc.SetAdmin("some-actor-id", target.User.ID, false)
	if err != nil {
		t.Fatalf("SetAdmin(false) failed: %v", err)
	}

	adminSvc.db.First(&userAfter, "id = ?", target.User.ID)
	if userAfter.IsInstanceAdministrator {
		t.Error("expected target to not be instance admin after SetAdmin(false)")
	}
}

func TestAdminSetAdmin_CannotDemoteSelf(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	admin := registerTestUser(t, authSvc, "demote-self@example.com")

	err := adminSvc.SetAdmin(admin.User.ID, admin.User.ID, false)
	if err != ErrCannotDemoteSelf {
		t.Errorf("expected ErrCannotDemoteSelf, got %v", err)
	}
}

func TestAdminSetAdmin_CanPromoteSelf(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	admin := registerTestUser(t, authSvc, "promote-self@example.com")

	err := adminSvc.SetAdmin(admin.User.ID, admin.User.ID, true)
	if err != nil {
		t.Fatalf("SetAdmin(self, true) should succeed: %v", err)
	}
}

func TestAdminSetAdmin_NotFound(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	registerTestUser(t, authSvc, "setadmin-nf@example.com")

	err := adminSvc.SetAdmin("some-actor", "nonexistent-id", true)
	if err != ErrAdminUserNotFound {
		t.Errorf("expected ErrAdminUserNotFound, got %v", err)
	}
}

func TestAdminDeleteUser_Success(t *testing.T) {
	adminSvc, authSvc, vaultSvc := setupAdminTest(t)

	admin := registerTestUser(t, authSvc, "del-admin@example.com")
	target := registerTestUser(t, authSvc, "del-target@example.com")

	vault, err := vaultSvc.CreateVault(target.User.AccountID, target.User.ID, dto.CreateVaultRequest{Name: "Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(adminSvc.db)
	_, err = contactSvc.CreateContact(vault.ID, target.User.ID, dto.CreateContactRequest{FirstName: "ToDelete"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	err = adminSvc.DeleteUser(admin.User.ID, target.User.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	var userCount int64
	adminSvc.db.Model(&models.User{}).Where("id = ?", target.User.ID).Count(&userCount)
	if userCount != 0 {
		t.Error("expected target user to be deleted")
	}

	var accountCount int64
	adminSvc.db.Model(&models.Account{}).Where("id = ?", target.User.AccountID).Count(&accountCount)
	if accountCount != 0 {
		t.Error("expected target account to be deleted")
	}

	var vaultCount int64
	adminSvc.db.Model(&models.Vault{}).Where("id = ?", vault.ID).Count(&vaultCount)
	if vaultCount != 0 {
		t.Error("expected vault to be deleted")
	}

	var contactCount int64
	adminSvc.db.Model(&models.Contact{}).Where("vault_id = ?", vault.ID).Count(&contactCount)
	if contactCount != 0 {
		t.Error("expected contacts to be deleted")
	}
}

func TestAdminDeleteUser_CannotDeleteSelf(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	admin := registerTestUser(t, authSvc, "del-self@example.com")

	err := adminSvc.DeleteUser(admin.User.ID, admin.User.ID)
	if err != ErrCannotDeleteSelf {
		t.Errorf("expected ErrCannotDeleteSelf, got %v", err)
	}
}

func TestAdminDeleteUser_NotFound(t *testing.T) {
	adminSvc, authSvc, _ := setupAdminTest(t)
	admin := registerTestUser(t, authSvc, "del-nf@example.com")

	err := adminSvc.DeleteUser(admin.User.ID, "nonexistent-id")
	if err != ErrAdminUserNotFound {
		t.Errorf("expected ErrAdminUserNotFound, got %v", err)
	}
}

func TestAdminDeleteUser_SharedAccount_OnlyDeletesTargetUser(t *testing.T) {
	adminSvc, authSvc, vaultSvc := setupAdminTest(t)

	owner := registerTestUser(t, authSvc, "shared-owner@example.com")

	vault, err := vaultSvc.CreateVault(owner.User.AccountID, owner.User.ID, dto.CreateVaultRequest{Name: "Shared Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(adminSvc.db)
	_, err = contactSvc.CreateContact(vault.ID, owner.User.ID, dto.CreateContactRequest{FirstName: "SharedContact"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	invitedUser := models.User{
		AccountID:              owner.User.AccountID,
		Email:                  "shared-invited@example.com",
		IsAccountAdministrator: false,
	}
	if err := adminSvc.db.Create(&invitedUser).Error; err != nil {
		t.Fatalf("Create invited user failed: %v", err)
	}
	invitedUserVault := models.UserVault{
		UserID:     invitedUser.ID,
		VaultID:    vault.ID,
		Permission: 200,
	}
	if err := adminSvc.db.Create(&invitedUserVault).Error; err != nil {
		t.Fatalf("Create UserVault failed: %v", err)
	}

	admin := registerTestUser(t, authSvc, "shared-admin@example.com")
	err = adminSvc.DeleteUser(admin.User.ID, invitedUser.ID)
	if err != nil {
		t.Fatalf("DeleteUser (shared account) failed: %v", err)
	}

	var deletedCount int64
	adminSvc.db.Model(&models.User{}).Where("id = ?", invitedUser.ID).Count(&deletedCount)
	if deletedCount != 0 {
		t.Error("expected invited user to be deleted")
	}

	var ownerCount int64
	adminSvc.db.Model(&models.User{}).Where("id = ?", owner.User.ID).Count(&ownerCount)
	if ownerCount != 1 {
		t.Error("expected owner user to still exist")
	}

	var accountCount int64
	adminSvc.db.Model(&models.Account{}).Where("id = ?", owner.User.AccountID).Count(&accountCount)
	if accountCount != 1 {
		t.Error("expected shared account to still exist")
	}

	var vaultCount int64
	adminSvc.db.Model(&models.Vault{}).Where("id = ?", vault.ID).Count(&vaultCount)
	if vaultCount != 1 {
		t.Error("expected vault to still exist")
	}

	var contactCount int64
	adminSvc.db.Model(&models.Contact{}).Where("vault_id = ?", vault.ID).Count(&contactCount)
	if contactCount != 1 {
		t.Error("expected contacts to still exist")
	}

	var uvCount int64
	adminSvc.db.Model(&models.UserVault{}).Where("user_id = ?", invitedUser.ID).Count(&uvCount)
	if uvCount != 0 {
		t.Error("expected deleted user's UserVault to be removed")
	}
}
