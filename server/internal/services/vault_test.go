package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultTest(t *testing.T) (*VaultService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	regReq := dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-test@example.com",
		Password:  "password123",
	}
	resp, err := authSvc.Register(regReq, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewVaultService(db), resp.User.AccountID, resp.User.ID
}

func TestCreateVault(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	req := dto.CreateVaultRequest{
		Name:        "My Vault",
		Description: "Test vault",
	}

	vault, err := svc.CreateVault(accountID, userID, req, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	if vault.Name != "My Vault" {
		t.Errorf("Expected name 'My Vault', got '%s'", vault.Name)
	}
	if vault.Description != "Test vault" {
		t.Errorf("Expected description 'Test vault', got '%s'", vault.Description)
	}
	if vault.ID == "" {
		t.Error("Expected vault ID to be non-empty")
	}

	db := svc.db

	var dateTypeCount int64
	db.Model(&models.ContactImportantDateType{}).Where("vault_id = ?", vault.ID).Count(&dateTypeCount)
	if dateTypeCount != 5 {
		t.Errorf("expected 5 ContactImportantDateTypes, got %d", dateTypeCount)
	}

	var moodCount int64
	db.Model(&models.MoodTrackingParameter{}).Where("vault_id = ?", vault.ID).Count(&moodCount)
	if moodCount != 5 {
		t.Errorf("expected 5 MoodTrackingParameters, got %d", moodCount)
	}

	var catCount int64
	db.Model(&models.LifeEventCategory{}).Where("vault_id = ?", vault.ID).Count(&catCount)
	if catCount != 4 {
		t.Errorf("expected 4 LifeEventCategories, got %d", catCount)
	}

	var qfCount int64
	db.Model(&models.VaultQuickFactsTemplate{}).Where("vault_id = ?", vault.ID).Count(&qfCount)
	if qfCount != 2 {
		t.Errorf("expected 2 VaultQuickFactsTemplates, got %d", qfCount)
	}
}

func TestCreateVault_UserContactAutoCreated(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{
		Name: "Contact Auto Test",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	if vault.UserContactID == "" {
		t.Fatal("Expected UserContactID to be populated after vault creation")
	}

	var uv models.UserVault
	if err := svc.db.Where("user_id = ? AND vault_id = ?", userID, vault.ID).First(&uv).Error; err != nil {
		t.Fatalf("UserVault lookup failed: %v", err)
	}
	if uv.ContactID == "" {
		t.Fatal("Expected UserVault.ContactID to be set")
	}
	if uv.ContactID != vault.UserContactID {
		t.Errorf("UserVault.ContactID (%s) != VaultResponse.UserContactID (%s)", uv.ContactID, vault.UserContactID)
	}

	var contact models.Contact
	if err := svc.db.First(&contact, "id = ?", uv.ContactID).Error; err != nil {
		t.Fatalf("Self-contact lookup failed: %v", err)
	}
	if contact.CanBeDeleted {
		t.Error("Self-contact should have CanBeDeleted=false")
	}
	if contact.Listed {
		t.Error("Self-contact should have Listed=false")
	}
	if contact.VaultID != vault.ID {
		t.Errorf("Self-contact VaultID = %s, want %s", contact.VaultID, vault.ID)
	}
}

func TestListVaults(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	_, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault 1"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	_, err = svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault 2"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	vaults, err := svc.ListVaults(userID)
	if err != nil {
		t.Fatalf("ListVaults failed: %v", err)
	}
	if len(vaults) != 2 {
		t.Errorf("Expected 2 vaults, got %d", len(vaults))
	}
}

func TestUpdateVault(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Before"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	updated, err := svc.UpdateVault(created.ID, userID, dto.UpdateVaultRequest{Name: "After", Description: "Updated"})
	if err != nil {
		t.Fatalf("UpdateVault failed: %v", err)
	}
	if updated.Name != "After" {
		t.Errorf("Expected name 'After', got '%s'", updated.Name)
	}
	if updated.Description != "Updated" {
		t.Errorf("Expected description 'Updated', got '%s'", updated.Description)
	}
}

func TestDeleteVault(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "ToDelete"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	if err := svc.DeleteVault(created.ID); err != nil {
		t.Fatalf("DeleteVault failed: %v", err)
	}

	_, err = svc.GetVault(created.ID, userID)
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound, got %v", err)
	}
}

func TestCheckUserVaultAccess(t *testing.T) {
	t.Run("creator_manager_permission", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)

		created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Access Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(userID, created.ID, models.PermissionManager); err != nil {
			t.Errorf("Expected access, got: %v", err)
		}
	})

	t.Run("nonexistent_user", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)

		created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Access Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess("nonexistent", created.ID, models.PermissionViewer); err != ErrVaultForbidden {
			t.Errorf("Expected ErrVaultForbidden, got %v", err)
		}
	})

	t.Run("editor_access_manager_only_fails", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)
		db := svc.db

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		editorUser := models.User{
			ID:        "test-editor-1",
			AccountID: accountID,
			Email:     "editor-user@example.com",
			FirstName: strPtrOrNil("Editor"),
		}
		if err := db.Create(&editorUser).Error; err != nil {
			t.Fatalf("Create editor user failed: %v", err)
		}

		uv := models.UserVault{
			UserID:     editorUser.ID,
			VaultID:    vault.ID,
			Permission: models.PermissionEditor,
		}
		if err := db.Create(&uv).Error; err != nil {
			t.Fatalf("Create UserVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(editorUser.ID, vault.ID, models.PermissionManager); err != ErrInsufficientPerm {
			t.Errorf("Expected ErrInsufficientPerm, got %v", err)
		}
	})

	t.Run("viewer_access_editor_only_fails", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)
		db := svc.db

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		viewerUser := models.User{
			ID:        "test-viewer-1",
			AccountID: accountID,
			Email:     "viewer-user@example.com",
			FirstName: strPtrOrNil("Viewer"),
		}
		if err := db.Create(&viewerUser).Error; err != nil {
			t.Fatalf("Create viewer user failed: %v", err)
		}

		uv := models.UserVault{
			UserID:     viewerUser.ID,
			VaultID:    vault.ID,
			Permission: models.PermissionViewer,
		}
		if err := db.Create(&uv).Error; err != nil {
			t.Fatalf("Create UserVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(viewerUser.ID, vault.ID, models.PermissionEditor); err != ErrInsufficientPerm {
			t.Errorf("Expected ErrInsufficientPerm, got %v", err)
		}
	})

	t.Run("editor_access_editor_required_succeeds", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)
		db := svc.db

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		editorUser := models.User{
			ID:        "test-editor-2",
			AccountID: accountID,
			Email:     "editor-user2@example.com",
			FirstName: strPtrOrNil("Editor"),
		}
		if err := db.Create(&editorUser).Error; err != nil {
			t.Fatalf("Create editor user failed: %v", err)
		}

		uv := models.UserVault{
			UserID:     editorUser.ID,
			VaultID:    vault.ID,
			Permission: models.PermissionEditor,
		}
		if err := db.Create(&uv).Error; err != nil {
			t.Fatalf("Create UserVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(editorUser.ID, vault.ID, models.PermissionEditor); err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})

	t.Run("manager_access_viewer_required_succeeds", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(userID, vault.ID, models.PermissionViewer); err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})
}
