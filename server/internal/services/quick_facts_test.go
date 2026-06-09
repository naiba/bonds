package services

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupQuickFactTest(t *testing.T) (*QuickFactService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "quick-facts-test@example.com",
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

	return NewQuickFactService(db), contact.ID, vault.ID
}

func createQuickFactTemplateForTest(t *testing.T, svc *QuickFactService, vaultID string, req dto.CreateQuickFactTemplateRequest) *dto.QuickFactTemplateResponse {
	t.Helper()
	tplSvc := NewVaultQuickFactTemplateService(svc.db)
	tpl, err := tplSvc.Create(vaultID, req)
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	return tpl
}

func TestCreateQuickFact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	fact, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{
		Content: "Loves coffee",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if fact.Content != "Loves coffee" {
		t.Errorf("Expected content 'Loves coffee', got '%s'", fact.Content)
	}
	if fact.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, fact.ContactID)
	}
	if fact.VaultQuickFactsTemplateID != 1 {
		t.Errorf("Expected vault_quick_facts_template_id 1, got %d", fact.VaultQuickFactsTemplateID)
	}
	if fact.ID == 0 {
		t.Error("Expected quick fact ID to be non-zero")
	}
}

func TestQuickFactTemplateMetadataAndGroupResponse(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	defaultValue := "Yes"
	helpText := "Choose a known value"
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{
		Label:         "Likes tea",
		FieldType:     QuickFactFieldSelect,
		SelectOptions: []string{"Yes", "No"},
		Required:      true,
		HelpText:      &helpText,
		DefaultValue:  &defaultValue,
	})

	created, err := svc.Create(contactID, vaultID, tpl.ID, dto.CreateQuickFactRequest{})
	if err != nil {
		t.Fatalf("Create select default quick fact failed: %v", err)
	}
	if created.FieldType != QuickFactFieldSelect {
		t.Fatalf("Expected select field type, got %s", created.FieldType)
	}
	if created.ValueOption == nil || *created.ValueOption != "Yes" || created.Content != "Yes" {
		t.Fatalf("Expected default option to be reflected in value and content, got %+v", created)
	}

	groups, err := svc.ListAll(contactID, vaultID)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	var found *dto.QuickFactGroupResponse
	for i := range groups {
		if groups[i].TemplateID == tpl.ID {
			found = &groups[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("Expected group for template %d", tpl.ID)
	}
	if found.FieldType != QuickFactFieldSelect || !found.Required || found.HelpText == nil || *found.HelpText != helpText || found.DefaultValue == nil || *found.DefaultValue != defaultValue {
		t.Fatalf("Unexpected template metadata in group: %+v", found)
	}
	if len(found.SelectOptions) != 2 || found.SelectOptions[0] != "Yes" || found.SelectOptions[1] != "No" {
		t.Fatalf("Unexpected select options: %+v", found.SelectOptions)
	}
}

func TestQuickFactScalarValidation(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	numberTpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Lucky number", FieldType: QuickFactFieldNumber, Required: true})
	if _, err := svc.Create(contactID, vaultID, numberTpl.ID, dto.CreateQuickFactRequest{}); !errors.Is(err, ErrQuickFactRequiredValue) {
		t.Fatalf("Expected required number error, got %v", err)
	}
	if _, err := svc.Create(contactID, vaultID, numberTpl.ID, dto.CreateQuickFactRequest{Content: "abc"}); !errors.Is(err, ErrQuickFactInvalidValue) {
		t.Fatalf("Expected invalid number error, got %v", err)
	}
	numberValue := 12.5
	created, err := svc.Create(contactID, vaultID, numberTpl.ID, dto.CreateQuickFactRequest{ValueNumber: &numberValue})
	if err != nil {
		t.Fatalf("Create number quick fact failed: %v", err)
	}
	if created.ValueNumber == nil || *created.ValueNumber != numberValue || created.Content != "12.5" {
		t.Fatalf("Unexpected number response: %+v", created)
	}

	dateTpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "First met", FieldType: QuickFactFieldDate})
	badDate := "2026-1-2"
	if _, err := svc.Create(contactID, vaultID, dateTpl.ID, dto.CreateQuickFactRequest{ValueDate: &badDate}); !errors.Is(err, ErrQuickFactInvalidValue) {
		t.Fatalf("Expected invalid date error, got %v", err)
	}
	goodDate := "2026-01-02"
	dateFact, err := svc.Create(contactID, vaultID, dateTpl.ID, dto.CreateQuickFactRequest{ValueDate: &goodDate})
	if err != nil {
		t.Fatalf("Create date quick fact failed: %v", err)
	}
	if dateFact.ValueDate == nil || *dateFact.ValueDate != goodDate || dateFact.Content != goodDate {
		t.Fatalf("Unexpected date response: %+v", dateFact)
	}
}

func TestQuickFactSelectValidation(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Favorite", FieldType: QuickFactFieldSelect, SelectOptions: []string{"A", "B"}})
	badOption := "C"
	if _, err := svc.Create(contactID, vaultID, tpl.ID, dto.CreateQuickFactRequest{ValueOption: &badOption}); !errors.Is(err, ErrQuickFactInvalidValue) {
		t.Fatalf("Expected invalid select option error, got %v", err)
	}
	goodOption := "B"
	created, err := svc.Create(contactID, vaultID, tpl.ID, dto.CreateQuickFactRequest{ValueOption: &goodOption})
	if err != nil {
		t.Fatalf("Create select quick fact failed: %v", err)
	}
	if created.ValueOption == nil || *created.ValueOption != goodOption || created.Content != goodOption {
		t.Fatalf("Unexpected select response: %+v", created)
	}
}

func TestQuickFactFileBackedCreateReplaceDelete(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	uploadDir := t.TempDir()
	fileSvc := NewVaultFileService(svc.db, uploadDir)
	svc.SetFileService(fileSvc)
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Memory", FieldType: QuickFactFieldPhoto})

	created, err := svc.UploadFile(contactID, vaultID, tpl.ID, "", "first.png", "image/png", int64(len("first")), bytes.NewReader([]byte("first")))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if created.FileID == nil || created.File == nil || created.File.Type != "photo" || created.File.Name != "first.png" {
		t.Fatalf("Expected photo file metadata, got %+v", created)
	}
	if created.Content != "first.png" {
		t.Fatalf("Expected content to be display filename, got %q", created.Content)
	}
	oldFileID := *created.FileID
	oldFile, err := fileSvc.Get(oldFileID, vaultID)
	if err != nil {
		t.Fatalf("Expected old file record to exist: %v", err)
	}
	oldPath := filepath.Join(uploadDir, oldFile.UUID)
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("Expected old file on disk before replace: %v", err)
	}

	replaced, err := svc.ReplaceFile(created.ID, contactID, vaultID, tpl.ID, "", "second.png", "image/png", int64(len("second")), bytes.NewReader([]byte("second")))
	if err != nil {
		t.Fatalf("ReplaceFile failed: %v", err)
	}
	if replaced.FileID == nil {
		t.Fatalf("Expected replacement file id, got %+v", replaced)
	}
	if *replaced.FileID == oldFileID {
		t.Fatalf("Expected replacement file id to differ from old id %d", oldFileID)
	}
	if replaced.File == nil {
		t.Fatalf("Expected replacement file metadata, got %+v", replaced)
	}
	if replaced.File.Name != "second.png" {
		t.Fatalf("Expected replacement file name second.png, got %q", replaced.File.Name)
	}
	if _, err := fileSvc.Get(oldFileID, vaultID); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("Expected old file record deleted, got %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("Expected old UUID file removed from disk, got %v", err)
	}
	newFileID := *replaced.FileID
	if err := svc.Delete(replaced.ID, contactID, vaultID, tpl.ID); err != nil {
		t.Fatalf("Delete quick fact failed: %v", err)
	}
	if _, err := fileSvc.Get(newFileID, vaultID); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("Expected replacement file deleted with quick fact, got %v", err)
	}
}

func TestQuickFactReplaceFileRejectsDeletedContact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	uploadDir := t.TempDir()
	fileSvc := NewVaultFileService(svc.db, uploadDir)
	svc.SetFileService(fileSvc)
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Memory", FieldType: QuickFactFieldPhoto})
	created, err := svc.UploadFile(contactID, vaultID, tpl.ID, "", "first.png", "image/png", int64(len("first")), bytes.NewReader([]byte("first")))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	contactSvc := NewContactService(svc.db)
	if err := contactSvc.DeleteContact(contactID, vaultID); err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	_, err = svc.ReplaceFile(created.ID, contactID, vaultID, tpl.ID, "", "second.png", "image/png", int64(len("second")), bytes.NewReader([]byte("second")))
	if !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("Expected ErrContactNotFound when replacing file for deleted contact, got %v", err)
	}
}

func TestQuickFactFileOwnershipExcludesContactModules(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	uploadDir := t.TempDir()
	fileSvc := NewVaultFileService(svc.db, uploadDir)
	svc.SetFileService(fileSvc)
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Memory", FieldType: QuickFactFieldPhoto})
	created, err := svc.UploadFile(contactID, vaultID, tpl.ID, "", "first.png", "image/png", int64(len("first")), bytes.NewReader([]byte("first")))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	photos, _, err := fileSvc.ListContactPhotos(contactID, vaultID, 1, 30)
	if err != nil {
		t.Fatalf("ListContactPhotos failed: %v", err)
	}
	if len(photos) != 0 {
		t.Fatalf("Expected QuickFact-owned files excluded from contact photos, got %+v", photos)
	}
	if created.FileID == nil {
		t.Fatal("Expected file id")
	}
	if err := fileSvc.DeleteContactPhoto(*created.FileID, contactID, vaultID); !errors.Is(err, ErrFileInUse) {
		t.Fatalf("Expected ErrFileInUse from contact photo delete, got %v", err)
	}
	var file models.File
	if err := svc.db.Where("id = ?", *created.FileID).First(&file).Error; err != nil {
		t.Fatalf("Expected file record: %v", err)
	}
	if file.FileableType == nil || *file.FileableType != "QuickFact" || file.FileableID == nil || *file.FileableID != created.ID {
		t.Fatalf("Expected QuickFact file ownership, got %+v", file)
	}
}

func TestQuickFactTemplateIDMismatchRejected(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	uploadDir := t.TempDir()
	fileSvc := NewVaultFileService(svc.db, uploadDir)
	svc.SetFileService(fileSvc)
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Memory", FieldType: QuickFactFieldPhoto})
	otherTpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Other", FieldType: QuickFactFieldPhoto})
	created, err := svc.UploadFile(contactID, vaultID, tpl.ID, "", "first.png", "image/png", int64(len("first")), bytes.NewReader([]byte("first")))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if _, err := svc.ReplaceFile(created.ID, contactID, vaultID, otherTpl.ID, "", "second.png", "image/png", int64(len("second")), bytes.NewReader([]byte("second"))); !errors.Is(err, ErrQuickFactTemplateMismatch) {
		t.Fatalf("Expected replace template mismatch, got %v", err)
	}
	textTpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Text", FieldType: QuickFactFieldText})
	textFact, err := svc.Create(contactID, vaultID, textTpl.ID, dto.CreateQuickFactRequest{Content: "hello"})
	if err != nil {
		t.Fatalf("Create text fact failed: %v", err)
	}
	if _, err := svc.Update(textFact.ID, contactID, vaultID, otherTpl.ID, dto.UpdateQuickFactRequest{Content: "bye"}); !errors.Is(err, ErrQuickFactTemplateMismatch) {
		t.Fatalf("Expected update template mismatch, got %v", err)
	}
	if err := svc.Delete(created.ID, contactID, vaultID, otherTpl.ID); !errors.Is(err, ErrQuickFactTemplateMismatch) {
		t.Fatalf("Expected delete template mismatch, got %v", err)
	}
}

func TestQuickFactTemplateDeleteCleansFactsAndOwnedFiles(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	uploadDir := t.TempDir()
	fileSvc := NewVaultFileService(svc.db, uploadDir)
	svc.SetFileService(fileSvc)
	tplSvc := NewVaultQuickFactTemplateService(svc.db)
	tplSvc.SetUploadDir(uploadDir)
	tpl, err := tplSvc.Create(vaultID, dto.CreateQuickFactTemplateRequest{Label: "Attachment", FieldType: QuickFactFieldDocument})
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	created, err := svc.UploadFile(contactID, vaultID, tpl.ID, "", "note.pdf", "application/pdf", int64(len("pdf")), bytes.NewReader([]byte("pdf")))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if created.FileID == nil {
		t.Fatal("Expected file id")
	}
	fileID := *created.FileID
	file, err := fileSvc.Get(fileID, vaultID)
	if err != nil {
		t.Fatalf("Get file failed: %v", err)
	}
	diskPath := filepath.Join(uploadDir, file.UUID)
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("Expected uploaded file on disk: %v", err)
	}
	if err := tplSvc.Delete(tpl.ID, vaultID); err != nil {
		t.Fatalf("Delete template failed: %v", err)
	}
	if _, err := fileSvc.Get(fileID, vaultID); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("Expected file record deleted, got %v", err)
	}
	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Fatalf("Expected file removed from disk, got %v", err)
	}
	facts, err := svc.List(contactID, vaultID, tpl.ID)
	if !errors.Is(err, ErrQuickFactTplNotFound) && len(facts) != 0 {
		t.Fatalf("Expected facts gone after template delete, got facts=%+v err=%v", facts, err)
	}
}

func TestVaultFileDeleteProtectsQuickFactReferences(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)
	uploadDir := t.TempDir()
	fileSvc := NewVaultFileService(svc.db, uploadDir)
	svc.SetFileService(fileSvc)
	tpl := createQuickFactTemplateForTest(t, svc, vaultID, dto.CreateQuickFactTemplateRequest{Label: "Attachment", FieldType: QuickFactFieldDocument})
	created, err := svc.UploadFile(contactID, vaultID, tpl.ID, "", "note.pdf", "application/pdf", int64(len("pdf")), bytes.NewReader([]byte("pdf")))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if created.FileID == nil {
		t.Fatal("Expected file id")
	}
	if err := fileSvc.Delete(*created.FileID, vaultID); !errors.Is(err, ErrFileInUse) {
		t.Fatalf("Expected ErrFileInUse, got %v", err)
	}
}

func TestListQuickFacts(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	_, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "Fact 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "Fact 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	facts, err := svc.List(contactID, vaultID, 1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(facts) != 2 {
		t.Errorf("Expected 2 quick facts, got %d", len(facts))
	}
}

func TestListAllQuickFactsGroupsFactsByVaultTemplate(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	var templates []models.VaultQuickFactsTemplate
	if err := svc.db.Where("vault_id = ?", vaultID).Order("position ASC").Find(&templates).Error; err != nil {
		t.Fatalf("List templates failed: %v", err)
	}
	if len(templates) < 2 {
		t.Fatalf("Expected at least 2 quick fact templates, got %d", len(templates))
	}

	_, err := svc.Create(contactID, vaultID, templates[0].ID, dto.CreateQuickFactRequest{Content: "Enjoys hiking"})
	if err != nil {
		t.Fatalf("Create first template fact failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, templates[1].ID, dto.CreateQuickFactRequest{Content: "Avoids caffeine"})
	if err != nil {
		t.Fatalf("Create second template fact failed: %v", err)
	}

	groups, err := svc.ListAll(contactID, vaultID)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(groups) != len(templates) {
		t.Fatalf("Expected %d template groups, got %d", len(templates), len(groups))
	}
	if groups[0].TemplateID != templates[0].ID {
		t.Errorf("Expected first template ID %d, got %d", templates[0].ID, groups[0].TemplateID)
	}
	if groups[0].TemplateLabel == "" {
		t.Error("Expected first template label to be populated")
	}
	if len(groups[0].Facts) != 1 || groups[0].Facts[0].Content != "Enjoys hiking" {
		t.Fatalf("Expected first group to contain hiking fact, got %+v", groups[0].Facts)
	}
	if len(groups[1].Facts) != 1 || groups[1].Facts[0].Content != "Avoids caffeine" {
		t.Fatalf("Expected second group to contain caffeine fact, got %+v", groups[1].Facts)
	}
}

func TestUpdateQuickFact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	created, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, vaultID, 1, dto.UpdateQuickFactRequest{
		Content: "Updated content",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", updated.Content)
	}
}

func TestDeleteQuickFact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	created, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID, 1); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	facts, err := svc.List(contactID, vaultID, 1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 quick facts after delete, got %d", len(facts))
	}
}

func TestDeleteQuickFactNotFound(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	err := svc.Delete(9999, contactID, vaultID, 1)
	if err != ErrQuickFactNotFound {
		t.Errorf("Expected ErrQuickFactNotFound, got %v", err)
	}
}
