package services

import (
	"bytes"
	"errors"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPostPhotoTest(t *testing.T) (*VaultFileService, uint, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "post-photos-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	journalSvc := NewJournalService(db)
	journal, err := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})
	if err != nil {
		t.Fatalf("CreateJournal failed: %v", err)
	}

	postSvc := NewPostService(db)
	post, err := postSvc.Create(journal.ID, vault.ID, dto.CreatePostRequest{
		Title:     "Test Post",
		WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}

	return NewVaultFileService(db, t.TempDir()), post.ID, vault.ID
}

func TestListPostPhotosEmpty(t *testing.T) {
	svc, postID, vaultID := setupPostPhotoTest(t)

	photos, err := svc.ListPostPhotos(postID, vaultID)
	if err != nil {
		t.Fatalf("ListPostPhotos failed: %v", err)
	}
	if len(photos) != 0 {
		t.Errorf("Expected 0 photos, got %d", len(photos))
	}
}

func TestUploadPostPhoto(t *testing.T) {
	svc, postID, vaultID := setupPostPhotoTest(t)

	content := []byte("fake image data for post photo")
	reader := bytes.NewReader(content)

	result, err := svc.UploadPostPhoto(postID, vaultID, "photo.jpg", "image/jpeg", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("UploadPostPhoto failed: %v", err)
	}

	if result.Name != "photo.jpg" {
		t.Errorf("Expected name 'photo.jpg', got '%s'", result.Name)
	}
	if result.MimeType != "image/jpeg" {
		t.Errorf("Expected mime_type 'image/jpeg', got '%s'", result.MimeType)
	}
	if result.Type != "photo" {
		t.Errorf("Expected type 'photo', got '%s'", result.Type)
	}
	if result.Size != len(content) {
		t.Errorf("Expected size %d, got %d", len(content), result.Size)
	}
	if result.UUID == "" {
		t.Error("Expected non-empty UUID")
	}
	if result.ID == 0 {
		t.Error("Expected non-zero ID")
	}

	diskPath := filepath.Join(svc.UploadDir(), result.UUID)
	data, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("File not found on disk: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Error("File content on disk does not match uploaded content")
	}
}

func TestListPostPhotosAfterUpload(t *testing.T) {
	svc, postID, vaultID := setupPostPhotoTest(t)

	content1 := []byte("photo 1")
	_, err := svc.UploadPostPhoto(postID, vaultID, "photo1.jpg", "image/jpeg", int64(len(content1)), bytes.NewReader(content1))
	if err != nil {
		t.Fatalf("UploadPostPhoto failed: %v", err)
	}

	content2 := []byte("photo 2")
	_, err = svc.UploadPostPhoto(postID, vaultID, "photo2.png", "image/png", int64(len(content2)), bytes.NewReader(content2))
	if err != nil {
		t.Fatalf("UploadPostPhoto failed: %v", err)
	}

	photos, err := svc.ListPostPhotos(postID, vaultID)
	if err != nil {
		t.Fatalf("ListPostPhotos failed: %v", err)
	}
	if len(photos) != 2 {
		t.Errorf("Expected 2 photos, got %d", len(photos))
	}
}

func TestDeletePostPhoto(t *testing.T) {
	svc, postID, vaultID := setupPostPhotoTest(t)

	content := []byte("to delete")
	uploaded, err := svc.UploadPostPhoto(postID, vaultID, "delete-me.jpg", "image/jpeg", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("UploadPostPhoto failed: %v", err)
	}

	diskPath := filepath.Join(svc.UploadDir(), uploaded.UUID)
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatal("File should exist on disk before delete")
	}

	if err := svc.DeletePostPhoto(uploaded.ID, postID, vaultID); err != nil {
		t.Fatalf("DeletePostPhoto failed: %v", err)
	}

	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Error("File should be removed from disk after delete")
	}

	photos, err := svc.ListPostPhotos(postID, vaultID)
	if err != nil {
		t.Fatalf("ListPostPhotos failed: %v", err)
	}
	if len(photos) != 0 {
		t.Errorf("Expected 0 photos after delete, got %d", len(photos))
	}
}

func TestDeletePostPhotoNotFound(t *testing.T) {
	svc, postID, vaultID := setupPostPhotoTest(t)

	err := svc.DeletePostPhoto(9999, postID, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

// The photo's record can be undeletable (it is a cover, referenced by a foreign
// key). The file on disk was removed before the record, leaving a record that
// points at nothing.
func TestDeletePostPhoto_KeepsTheFileOnDiskWhenTheRecordCannotBeDeleted(t *testing.T) {
	svc, postID, vaultID := setupPostPhotoTest(t)
	content := []byte("photo that must survive")
	uploaded, err := svc.UploadPostPhoto(postID, vaultID, "cover.jpg", "image/jpeg", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("UploadPostPhoto: %v", err)
	}
	storedPath := filepath.Join(svc.UploadDir(), uploaded.UUID)
	recordDeletionFailure := errors.New("record deletion refused")
	const hook = "bonds:test_refuse_post_photo_record_deletion"
	if err := svc.db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "files" {
			_ = tx.AddError(recordDeletionFailure)
		}
	}); err != nil {
		t.Fatalf("register hook: %v", err)
	}
	t.Cleanup(func() { _ = svc.db.Callback().Delete().Remove(hook) })

	if err := svc.DeletePostPhoto(uploaded.ID, postID, vaultID); !errors.Is(err, recordDeletionFailure) {
		t.Fatalf("DeletePostPhoto error = %v, want the record deletion failure", err)
	}

	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("the file was removed although its record is still there: %v", err)
	}
	if !bytes.Equal(stored, content) {
		t.Fatalf("file content = %q, want %q", stored, content)
	}
}
