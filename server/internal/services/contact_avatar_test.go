package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupContactAvatarTest(t *testing.T) (*ContactAvatarService, string, string, *gorm.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "avatar-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewContactAvatarService(db), contact.ID, vault.ID, db
}

func TestContactAvatarUpdate(t *testing.T) {
	svc, contactID, vaultID, db := setupContactAvatarTest(t)

	file := models.File{
		VaultID:  vaultID,
		UUID:     "test-uuid",
		Name:     "avatar.png",
		MimeType: "image/png",
		Type:     "avatar",
		Size:     1024,
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	resp, err := svc.UpdateAvatar(contactID, vaultID, file.ID)
	if err != nil {
		t.Fatalf("UpdateAvatar failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactAvatarDelete(t *testing.T) {
	svc, contactID, vaultID, db := setupContactAvatarTest(t)

	file := models.File{
		VaultID:  vaultID,
		UUID:     "test-uuid",
		Name:     "avatar.png",
		MimeType: "image/png",
		Type:     "avatar",
		Size:     1024,
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	_, err := svc.UpdateAvatar(contactID, vaultID, file.ID)
	if err != nil {
		t.Fatalf("UpdateAvatar failed: %v", err)
	}

	resp, err := svc.DeleteAvatar(contactID, vaultID)
	if err != nil {
		t.Fatalf("DeleteAvatar failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactAvatarUpdateNotFound(t *testing.T) {
	svc, _, vaultID, _ := setupContactAvatarTest(t)

	_, err := svc.UpdateAvatar("nonexistent-id", vaultID, 1)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactAvatarDeleteNotFound(t *testing.T) {
	svc, _, vaultID, _ := setupContactAvatarTest(t)

	_, err := svc.DeleteAvatar("nonexistent-id", vaultID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}
