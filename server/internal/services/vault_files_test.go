package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultFileTest(t *testing.T) (*VaultFileService, string, *models.File) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-files-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	file := &models.File{
		VaultID:  vault.ID,
		UUID:     "test-uuid-123",
		Name:     "test.pdf",
		MimeType: "application/pdf",
		Type:     "document",
		Size:     1024,
	}
	if err := db.Create(file).Error; err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	return NewVaultFileService(db, t.TempDir()), vault.ID, file
}

func TestVaultFileListEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-files-empty@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Empty Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewVaultFileService(db, t.TempDir())
	files, err := svc.List(vault.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}

func TestVaultFileListWithFiles(t *testing.T) {
	svc, vaultID, _ := setupVaultFileTest(t)

	files, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	if files[0].Name != "test.pdf" {
		t.Errorf("Expected name 'test.pdf', got '%s'", files[0].Name)
	}
	if files[0].MimeType != "application/pdf" {
		t.Errorf("Expected mime_type 'application/pdf', got '%s'", files[0].MimeType)
	}
	if files[0].Size != 1024 {
		t.Errorf("Expected size 1024, got %d", files[0].Size)
	}
}

func TestVaultFileDelete(t *testing.T) {
	svc, vaultID, file := setupVaultFileTest(t)

	if err := svc.Delete(file.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	files, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files after delete, got %d", len(files))
	}
}

func TestVaultFileDeleteNotFound(t *testing.T) {
	svc, vaultID, _ := setupVaultFileTest(t)

	err := svc.Delete(9999, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestUploadFile(t *testing.T) {
	svc, vaultID, _ := setupVaultFileTest(t)

	content := []byte("hello world test file content")
	reader := bytes.NewReader(content)

	result, err := svc.Upload(vaultID, "", "", "document", "test.txt", "text/plain", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if result.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", result.Name)
	}
	if result.MimeType != "text/plain" {
		t.Errorf("Expected mime 'text/plain', got '%s'", result.MimeType)
	}
	if result.Type != "document" {
		t.Errorf("Expected type 'document', got '%s'", result.Type)
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

func TestUploadFileWithContact(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "file-contact-test@example.com",
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

	svc := NewVaultFileService(db, t.TempDir())
	content := []byte("photo data")
	reader := bytes.NewReader(content)

	result, err := svc.Upload(vault.ID, contact.ID, resp.User.ID, "photo", "avatar.png", "image/png", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if result.Type != "photo" {
		t.Errorf("Expected type 'photo', got '%s'", result.Type)
	}
	if result.VaultID != vault.ID {
		t.Errorf("Expected vault_id '%s', got '%s'", vault.ID, result.VaultID)
	}
}

func TestGetFile(t *testing.T) {
	svc, vaultID, _ := setupVaultFileTest(t)

	content := []byte("get test content")
	reader := bytes.NewReader(content)

	uploaded, err := svc.Upload(vaultID, "", "", "document", "doc.pdf", "application/pdf", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	got, err := svc.Get(uploaded.ID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != uploaded.ID {
		t.Errorf("Expected ID %d, got %d", uploaded.ID, got.ID)
	}
	if got.Name != "doc.pdf" {
		t.Errorf("Expected name 'doc.pdf', got '%s'", got.Name)
	}
}

func TestGetFileNotFound(t *testing.T) {
	svc, vaultID, _ := setupVaultFileTest(t)

	_, err := svc.Get(9999, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestVaultFilesListByTypePagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "files-type-pagination@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewVaultFileService(db, t.TempDir())
	for i := 0; i < 3; i++ {
		file := models.File{
			VaultID:  vault.ID,
			UUID:     "type-page-uuid-" + string(rune('a'+i)),
			Name:     "doc.pdf",
			MimeType: "application/pdf",
			Type:     "document",
			Size:     1024,
		}
		if err := db.Create(&file).Error; err != nil {
			t.Fatalf("Create file failed: %v", err)
		}
	}

	files, meta, err := svc.ListByType(vault.ID, "document", 1, 2)
	if err != nil {
		t.Fatalf("ListByType page 1 failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files on page 1, got %d", len(files))
	}
	if meta.Total != 3 {
		t.Errorf("Expected total 3, got %d", meta.Total)
	}
	if meta.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", meta.TotalPages)
	}

	files2, meta2, err := svc.ListByType(vault.ID, "document", 2, 2)
	if err != nil {
		t.Fatalf("ListByType page 2 failed: %v", err)
	}
	if len(files2) != 1 {
		t.Errorf("Expected 1 file on page 2, got %d", len(files2))
	}
	if meta2.Total != 3 {
		t.Errorf("Expected total 3 on page 2, got %d", meta2.Total)
	}
}

func TestUploadFileWithContactCreatesFeedEntry(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "file-feed-test@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	svc := NewVaultFileService(db, t.TempDir())
	feedRecorder := NewFeedRecorder(db)
	svc.SetFeedRecorder(feedRecorder)

	content := []byte("photo data for feed test")
	reader := bytes.NewReader(content)

	result, err := svc.Upload(vault.ID, contact.ID, resp.User.ID, "photo", "vacation.jpg", "image/jpeg", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Verify feed entry was created
	var feedItem models.ContactFeedItem
	err = db.Where("contact_id = ? AND action = ?", contact.ID, ActionFileUploaded).First(&feedItem).Error
	if err != nil {
		t.Fatalf("Expected feed item for file upload, got error: %v", err)
	}
	if feedItem.ContactID != contact.ID {
		t.Errorf("Expected contact_id '%s', got '%s'", contact.ID, feedItem.ContactID)
	}
	if feedItem.AuthorID == nil || *feedItem.AuthorID != resp.User.ID {
		t.Errorf("Expected author_id '%s', got '%v'", resp.User.ID, feedItem.AuthorID)
	}
	if feedItem.FeedableID == nil || *feedItem.FeedableID != result.ID {
		t.Errorf("Expected feedable_id %d, got %v", result.ID, feedItem.FeedableID)
	}
	if feedItem.FeedableType == nil || *feedItem.FeedableType != "File" {
		t.Errorf("Expected feedable_type 'File', got %v", feedItem.FeedableType)
	}
	if feedItem.Description == nil || *feedItem.Description != "Uploaded photo: vacation.jpg" {
		t.Errorf("Expected description 'Uploaded photo: vacation.jpg', got %v", feedItem.Description)
	}
}

func TestUploadFileWithoutContactNoFeedEntry(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "file-nofeed-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewVaultFileService(db, t.TempDir())
	feedRecorder := NewFeedRecorder(db)
	svc.SetFeedRecorder(feedRecorder)

	content := []byte("document data")
	reader := bytes.NewReader(content)

	_, err = svc.Upload(vault.ID, "", resp.User.ID, "document", "report.pdf", "application/pdf", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// No feed entry should be created when contactID is empty
	var count int64
	db.Model(&models.ContactFeedItem{}).Where("action = ?", ActionFileUploaded).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 feed items for file upload without contact, got %d", count)
	}
}

func TestUploadFileWithoutFeedRecorder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "file-norecorder-test@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// No feedRecorder set — should not panic
	svc := NewVaultFileService(db, t.TempDir())

	content := []byte("safe upload")
	reader := bytes.NewReader(content)

	_, err = svc.Upload(vault.ID, contact.ID, resp.User.ID, "document", "safe.txt", "text/plain", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload should succeed without feedRecorder, got: %v", err)
	}
}

func TestDeleteFileRemovesDisk(t *testing.T) {
	svc, vaultID, _ := setupVaultFileTest(t)

	content := []byte("delete me")
	reader := bytes.NewReader(content)

	uploaded, err := svc.Upload(vaultID, "", "", "document", "temp.txt", "text/plain", int64(len(content)), reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	diskPath := filepath.Join(svc.UploadDir(), uploaded.UUID)
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatal("File should exist on disk before delete")
	}

	if err := svc.Delete(uploaded.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Error("File should be removed from disk after delete")
	}

	_, err = svc.Get(uploaded.ID, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound after delete, got %v", err)
	}
}
