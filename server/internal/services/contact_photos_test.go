package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactPhotoTest(t *testing.T) (*VaultFileService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-photos-test@example.com",
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

	contactID := contact.ID
	for i, name := range []string{"photo1.jpg", "photo2.png"} {
		file := models.File{
			VaultID:     vault.ID,
			UUID:        "contact-photo-uuid-" + name,
			Name:        name,
			MimeType:    "image/jpeg",
			Type:        "photo",
			Size:        1024 * (i + 1),
			UfileableID: &contactID,
		}
		if err := db.Create(&file).Error; err != nil {
			t.Fatalf("Create file failed: %v", err)
		}
	}

	return svc, contact.ID, vault.ID
}

func TestListContactPhotos(t *testing.T) {
	svc, contactID, vaultID := setupContactPhotoTest(t)

	photos, _, err := svc.ListContactPhotos(contactID, vaultID, 1, 30)
	if err != nil {
		t.Fatalf("ListContactPhotos failed: %v", err)
	}
	if len(photos) != 2 {
		t.Errorf("Expected 2 photos, got %d", len(photos))
	}
}

func TestListContactPhotosEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-photos-empty@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Empty Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Nobody"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	svc := NewVaultFileService(db, t.TempDir())
	photos, _, err := svc.ListContactPhotos(contact.ID, vault.ID, 1, 30)
	if err != nil {
		t.Fatalf("ListContactPhotos failed: %v", err)
	}
	if len(photos) != 0 {
		t.Errorf("Expected 0 photos, got %d", len(photos))
	}
}

func TestGetContactPhoto(t *testing.T) {
	svc, contactID, vaultID := setupContactPhotoTest(t)

	photos, _, err := svc.ListContactPhotos(contactID, vaultID, 1, 30)
	if err != nil {
		t.Fatalf("ListContactPhotos failed: %v", err)
	}
	if len(photos) == 0 {
		t.Fatal("Expected at least 1 photo")
	}

	photo, err := svc.GetContactPhoto(photos[0].ID, contactID, vaultID)
	if err != nil {
		t.Fatalf("GetContactPhoto failed: %v", err)
	}
	if photo.ID != photos[0].ID {
		t.Errorf("Expected ID %d, got %d", photos[0].ID, photo.ID)
	}
	if photo.Type != "photo" {
		t.Errorf("Expected type 'photo', got '%s'", photo.Type)
	}
}

func TestGetContactPhotoNotFound(t *testing.T) {
	svc, contactID, vaultID := setupContactPhotoTest(t)

	_, err := svc.GetContactPhoto(9999, contactID, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestDeleteContactPhoto(t *testing.T) {
	svc, contactID, vaultID := setupContactPhotoTest(t)

	photos, _, err := svc.ListContactPhotos(contactID, vaultID, 1, 30)
	if err != nil {
		t.Fatalf("ListContactPhotos failed: %v", err)
	}
	if len(photos) == 0 {
		t.Fatal("Expected at least 1 photo")
	}

	if err := svc.DeleteContactPhoto(photos[0].ID, contactID, vaultID); err != nil {
		t.Fatalf("DeleteContactPhoto failed: %v", err)
	}

	remaining, _, err := svc.ListContactPhotos(contactID, vaultID, 1, 30)
	if err != nil {
		t.Fatalf("ListContactPhotos failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("Expected 1 photo after delete, got %d", len(remaining))
	}
}

func TestDeleteContactPhotoNotFound(t *testing.T) {
	svc, contactID, vaultID := setupContactPhotoTest(t)

	err := svc.DeleteContactPhoto(9999, contactID, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestListContactDocuments(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-docs-test@example.com",
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
	contactID := contact.ID
	for _, name := range []string{"doc1.pdf", "doc2.pdf"} {
		file := models.File{
			VaultID:     vault.ID,
			UUID:        "contact-doc-uuid-" + name,
			Name:        name,
			MimeType:    "application/pdf",
			Type:        "document",
			Size:        2048,
			UfileableID: &contactID,
		}
		if err := db.Create(&file).Error; err != nil {
			t.Fatalf("Create file failed: %v", err)
		}
	}

	docs, _, err := svc.ListContactDocuments(contactID, vault.ID, 1, 15)
	if err != nil {
		t.Fatalf("ListContactDocuments failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

func TestDeleteContactDocument(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-docs-del@example.com",
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
	contactID := contact.ID
	file := models.File{
		VaultID:     vault.ID,
		UUID:        "contact-doc-uuid-del",
		Name:        "to-delete.pdf",
		MimeType:    "application/pdf",
		Type:        "document",
		Size:        2048,
		UfileableID: &contactID,
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	if err := svc.DeleteContactDocument(file.ID, contactID, vault.ID); err != nil {
		t.Fatalf("DeleteContactDocument failed: %v", err)
	}

	docs, _, err := svc.ListContactDocuments(contactID, vault.ID, 1, 15)
	if err != nil {
		t.Fatalf("ListContactDocuments failed: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents after delete, got %d", len(docs))
	}
}

func TestDeleteContactDocumentNotFound(t *testing.T) {
	svc, contactID, vaultID := setupContactPhotoTest(t)

	err := svc.DeleteContactDocument(9999, contactID, vaultID)
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

func TestContactPhotosListPagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "photos-pagination@example.com",
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
	contactID := contact.ID
	for i := 0; i < 3; i++ {
		file := models.File{
			VaultID:     vault.ID,
			UUID:        "photo-page-uuid-" + string(rune('a'+i)),
			Name:        "photo.jpg",
			MimeType:    "image/jpeg",
			Type:        "photo",
			Size:        1024,
			UfileableID: &contactID,
		}
		if err := db.Create(&file).Error; err != nil {
			t.Fatalf("Create file failed: %v", err)
		}
	}

	photos, meta, err := svc.ListContactPhotos(contactID, vault.ID, 1, 2)
	if err != nil {
		t.Fatalf("ListContactPhotos page 1 failed: %v", err)
	}
	if len(photos) != 2 {
		t.Errorf("Expected 2 photos on page 1, got %d", len(photos))
	}
	if meta.Total != 3 {
		t.Errorf("Expected total 3, got %d", meta.Total)
	}
	if meta.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", meta.TotalPages)
	}

	photos2, meta2, err := svc.ListContactPhotos(contactID, vault.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListContactPhotos page 2 failed: %v", err)
	}
	if len(photos2) != 1 {
		t.Errorf("Expected 1 photo on page 2, got %d", len(photos2))
	}
	if meta2.Total != 3 {
		t.Errorf("Expected total 3 on page 2, got %d", meta2.Total)
	}
}

func TestContactDocumentsListPagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "docs-pagination@example.com",
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
	contactID := contact.ID
	for i := 0; i < 3; i++ {
		file := models.File{
			VaultID:     vault.ID,
			UUID:        "doc-page-uuid-" + string(rune('a'+i)),
			Name:        "doc.pdf",
			MimeType:    "application/pdf",
			Type:        "document",
			Size:        2048,
			UfileableID: &contactID,
		}
		if err := db.Create(&file).Error; err != nil {
			t.Fatalf("Create file failed: %v", err)
		}
	}

	docs, meta, err := svc.ListContactDocuments(contactID, vault.ID, 1, 2)
	if err != nil {
		t.Fatalf("ListContactDocuments page 1 failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents on page 1, got %d", len(docs))
	}
	if meta.Total != 3 {
		t.Errorf("Expected total 3, got %d", meta.Total)
	}
	if meta.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", meta.TotalPages)
	}

	docs2, meta2, err := svc.ListContactDocuments(contactID, vault.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListContactDocuments page 2 failed: %v", err)
	}
	if len(docs2) != 1 {
		t.Errorf("Expected 1 document on page 2, got %d", len(docs2))
	}
	if meta2.Total != 3 {
		t.Errorf("Expected total 3 on page 2, got %d", meta2.Total)
	}
}
