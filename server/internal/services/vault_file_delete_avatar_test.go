package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// A file used as a contact's avatar is referenced by contacts.file_id. With
// foreign keys enforced, deleting the file record failed on that reference
// after the file had already been removed from disk: the record survived and
// pointed at nothing.
func TestVaultFileDelete_ReleasesContactAvatarBeforeRemovingTheFile(t *testing.T) {
	svc, vaultID, file := setupVaultFileTest(t)
	storedPath := filepath.Join(svc.uploadDir, file.UUID)
	if err := os.WriteFile(storedPath, []byte("avatar"), 0o600); err != nil {
		t.Fatalf("write stored file: %v", err)
	}
	firstName := "Avatar"
	contact := models.Contact{VaultID: vaultID, FirstName: &firstName, FileID: &file.ID}
	if err := svc.db.Create(&contact).Error; err != nil {
		t.Fatalf("create contact using the file as avatar: %v", err)
	}

	if err := svc.Delete(file.ID, vaultID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var reloaded models.Contact
	if err := svc.db.First(&reloaded, "id = ?", contact.ID).Error; err != nil {
		t.Fatalf("reload contact: %v", err)
	}
	if reloaded.FileID != nil {
		t.Errorf("contact still references deleted file %d as its avatar", *reloaded.FileID)
	}
	var remaining int64
	if err := svc.db.Model(&models.File{}).Where("id = ?", file.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count file records: %v", err)
	}
	if remaining != 0 {
		t.Errorf("file record still exists")
	}
	if _, err := os.Stat(storedPath); !os.IsNotExist(err) {
		t.Errorf("stored file still on disk (stat err = %v)", err)
	}
}

// The file on disk cannot be brought back by a rollback, so it must only be
// removed once the database work has succeeded.
func TestVaultFileDelete_KeepsTheFileOnDiskWhenTheRecordCannotBeDeleted(t *testing.T) {
	svc, vaultID, file := setupVaultFileTest(t)
	storedPath := filepath.Join(svc.uploadDir, file.UUID)
	if err := os.WriteFile(storedPath, []byte("document"), 0o600); err != nil {
		t.Fatalf("write stored file: %v", err)
	}
	recordDeletionFailure := errors.New("record deletion refused")
	const hook = "bonds:test_refuse_file_record_deletion"
	if err := svc.db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "files" {
			_ = tx.AddError(recordDeletionFailure)
		}
	}); err != nil {
		t.Fatalf("register hook: %v", err)
	}
	t.Cleanup(func() { _ = svc.db.Callback().Delete().Remove(hook) })

	if err := svc.Delete(file.ID, vaultID); !errors.Is(err, recordDeletionFailure) {
		t.Fatalf("Delete error = %v, want the record deletion failure", err)
	}

	if _, err := os.Stat(storedPath); err != nil {
		t.Errorf("stored file was removed although its record is still there: %v", err)
	}
}
