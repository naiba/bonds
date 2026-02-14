package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupGroupTest(t *testing.T) (*GroupService, string, *gorm.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "groups-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewGroupService(db), vault.ID, db
}

func TestListGroups(t *testing.T) {
	svc, vaultID, db := setupGroupTest(t)

	group1 := models.Group{VaultID: vaultID, Name: "Family"}
	if err := db.Create(&group1).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}
	group2 := models.Group{VaultID: vaultID, Name: "Friends"}
	if err := db.Create(&group2).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	groups, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestGetGroup(t *testing.T) {
	svc, vaultID, db := setupGroupTest(t)

	group := models.Group{VaultID: vaultID, Name: "Work Team"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	got, err := svc.Get(group.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Work Team" {
		t.Errorf("Expected name 'Work Team', got '%s'", got.Name)
	}
	if got.VaultID != vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", vaultID, got.VaultID)
	}
	if got.ID != group.ID {
		t.Errorf("Expected ID %d, got %d", group.ID, got.ID)
	}
	if len(got.Contacts) != 0 {
		t.Errorf("Expected 0 contacts, got %d", len(got.Contacts))
	}
}

func TestUpdateGroup(t *testing.T) {
	svc, vaultID, db := setupGroupTest(t)

	group := models.Group{VaultID: vaultID, Name: "Original"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	updated, err := svc.Update(group.ID, dto.UpdateGroupRequest{
		Name: "Updated Group",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated Group" {
		t.Errorf("Expected name 'Updated Group', got '%s'", updated.Name)
	}
}

func TestDeleteGroup(t *testing.T) {
	svc, vaultID, db := setupGroupTest(t)

	group := models.Group{VaultID: vaultID, Name: "To delete"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	if err := svc.Delete(group.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	groups, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups after delete, got %d", len(groups))
	}
}

func TestDeleteGroupNotFound(t *testing.T) {
	svc, _, _ := setupGroupTest(t)

	err := svc.Delete(9999)
	if err != ErrGroupNotFound {
		t.Errorf("Expected ErrGroupNotFound, got %v", err)
	}
}

func TestGetGroupNotFound(t *testing.T) {
	svc, _, _ := setupGroupTest(t)

	_, err := svc.Get(9999)
	if err != ErrGroupNotFound {
		t.Errorf("Expected ErrGroupNotFound, got %v", err)
	}
}
