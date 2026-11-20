package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

// Deleting a vault removes its file records. The files themselves have to go
// too, or they stay in the upload directory with nothing pointing at them.
func TestDeleteVault_RemovesItsUploadedFilesFromDisk(t *testing.T) {
	db := testutil.SetupTestDB(t)
	registered, err := NewAuthService(db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{
		FirstName: "Vault", LastName: "Owner", Email: "vault-files@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	uploadDir := t.TempDir()
	vaultSvc := NewVaultService(db)
	vaultSvc.SetUploadDir(uploadDir)
	deleted, err := vaultSvc.CreateVault(registered.User.AccountID, registered.User.ID, dto.CreateVaultRequest{Name: "Deleted"}, "en")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	kept, err := vaultSvc.CreateVault(registered.User.AccountID, registered.User.ID, dto.CreateVaultRequest{Name: "Kept"}, "en")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	storeFile := func(vaultID, fileUUID string) string {
		path := filepath.Join(uploadDir, fileUUID)
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatalf("write upload: %v", err)
		}
		if err := db.Create(&models.File{VaultID: vaultID, UUID: fileUUID, Name: "photo.png", MimeType: "image/png", Type: "photo", Size: 7}).Error; err != nil {
			t.Fatalf("create file record: %v", err)
		}
		return path
	}
	deletedVaultFile := storeFile(deleted.ID, "11111111-1111-4111-8111-111111111111")
	keptVaultFile := storeFile(kept.ID, "22222222-2222-4222-8222-222222222222")

	if err := vaultSvc.DeleteVault(deleted.ID); err != nil {
		t.Fatalf("DeleteVault failed: %v", err)
	}

	if _, err := os.Stat(deletedVaultFile); !os.IsNotExist(err) {
		t.Errorf("file of the deleted vault is still on disk (stat err = %v)", err)
	}
	if _, err := os.Stat(keptVaultFile); err != nil {
		t.Errorf("file of another vault was removed: %v", err)
	}
}
