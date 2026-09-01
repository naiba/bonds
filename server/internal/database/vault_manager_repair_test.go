package database

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRepairManagerlessVaultsPromotesBestActiveMember(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.Account{}, &models.User{}, &models.Vault{}, &models.UserVault{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	account := models.Account{}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	vault := models.Vault{AccountID: account.ID, Name: "Managerless", Type: "personal"}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("create vault: %v", err)
	}
	disabledAdmin := models.User{AccountID: account.ID, Email: "disabled-admin@example.com"}
	activeUser := models.User{AccountID: account.ID, Email: "active-user@example.com"}
	activeAdmin := models.User{AccountID: account.ID, Email: "active-admin@example.com"}
	for _, user := range []*models.User{&disabledAdmin, &activeUser, &activeAdmin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Email, err)
		}
	}
	if err := db.Model(&disabledAdmin).Updates(map[string]interface{}{"disabled": true, "is_account_administrator": true}).Error; err != nil {
		t.Fatalf("configure disabled admin: %v", err)
	}
	if err := db.Model(&activeAdmin).Update("is_account_administrator", true).Error; err != nil {
		t.Fatalf("configure active admin: %v", err)
	}
	for _, user := range []models.User{disabledAdmin, activeUser, activeAdmin} {
		permission := models.PermissionEditor
		if user.ID == disabledAdmin.ID {
			permission = models.PermissionManager
		}
		if err := db.Create(&models.UserVault{VaultID: vault.ID, UserID: user.ID, Permission: permission}).Error; err != nil {
			t.Fatalf("create membership for %s: %v", user.Email, err)
		}
	}

	if err := repairManagerlessVaults(db); err != nil {
		t.Fatalf("repair managerless vault: %v", err)
	}
	if err := repairManagerlessVaults(db); err != nil {
		t.Fatalf("second repair managerless vault: %v", err)
	}
	var memberships []models.UserVault
	if err := db.Where("vault_id = ?", vault.ID).Find(&memberships).Error; err != nil {
		t.Fatalf("list memberships: %v", err)
	}
	activeManagerCount := 0
	for _, membership := range memberships {
		if membership.Permission == models.PermissionManager && membership.UserID != disabledAdmin.ID {
			activeManagerCount++
			if membership.UserID != activeAdmin.ID {
				t.Errorf("promoted user = %s, want active admin %s", membership.UserID, activeAdmin.ID)
			}
		}
	}
	if activeManagerCount != 1 {
		t.Fatalf("active manager count = %d, want 1", activeManagerCount)
	}
}
