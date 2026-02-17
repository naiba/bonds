package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func setupTemplatePageTest(t *testing.T) (*TemplatePageService, string, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)

	account := models.Account{}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("Create account failed: %v", err)
	}

	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	hStr := string(hashedPwd)
	user := models.User{
		AccountID:              account.ID,
		Email:                  "tp-test@example.com",
		Password:               &hStr,
		IsAccountAdministrator: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	if err := models.SeedAccountDefaults(db, account.ID, user.ID, user.Email); err != nil {
		t.Fatalf("SeedAccountDefaults failed: %v", err)
	}

	var tmpl models.Template
	if err := db.Where("account_id = ?", account.ID).First(&tmpl).Error; err != nil {
		t.Fatalf("Find template failed: %v", err)
	}

	svc := NewTemplatePageService(db)
	return svc, account.ID, tmpl.ID
}

func TestTemplatePageList(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	pages, err := svc.List(templateID, accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(pages) != 5 {
		t.Errorf("Expected 5 default pages, got %d", len(pages))
	}
}

func TestTemplatePageCreate(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	page, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name:     "Custom Page",
		Slug:     "custom",
		Position: 10,
		Type:     "custom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if page.Name != "Custom Page" {
		t.Errorf("Expected name 'Custom Page', got '%s'", page.Name)
	}
	if page.Slug != "custom" {
		t.Errorf("Expected slug 'custom', got '%s'", page.Slug)
	}
	if page.Position != 10 {
		t.Errorf("Expected position 10, got %d", page.Position)
	}
	if page.Type != "custom" {
		t.Errorf("Expected type 'custom', got '%s'", page.Type)
	}
	if page.ID == 0 {
		t.Error("Expected page ID to be non-zero")
	}
	if !page.CanBeDeleted {
		t.Error("Expected new page to be deletable")
	}
}

func TestTemplatePageGet(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	pages, err := svc.List(templateID, accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("Expected at least one page")
	}

	page, err := svc.Get(pages[0].ID, accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if page.ID != pages[0].ID {
		t.Errorf("Expected page ID %d, got %d", pages[0].ID, page.ID)
	}
}

func TestTemplatePageUpdate(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	created, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name:     "Original",
		Slug:     "original",
		Position: 1,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, accountID, dto.UpdateTemplatePageRequest{
		Name:     "Updated",
		Slug:     "updated",
		Position: 5,
		Type:     "new-type",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", updated.Name)
	}
	if updated.Slug != "updated" {
		t.Errorf("Expected slug 'updated', got '%s'", updated.Slug)
	}
	if updated.Position != 5 {
		t.Errorf("Expected position 5, got %d", updated.Position)
	}
	if updated.Type != "new-type" {
		t.Errorf("Expected type 'new-type', got '%s'", updated.Type)
	}
}

func TestTemplatePageDelete(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	created, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name: "To Delete", Slug: "to-delete", Position: 99,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, accountID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.Get(created.ID, accountID)
	if err != ErrTemplatePageNotFound {
		t.Errorf("Expected ErrTemplatePageNotFound after delete, got %v", err)
	}
}

func TestTemplatePageDeleteCannotBeDeleted(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	pages, err := svc.List(templateID, accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var undeletablePage *dto.TemplatePageResponse
	for i, p := range pages {
		if !p.CanBeDeleted {
			undeletablePage = &pages[i]
			break
		}
	}
	if undeletablePage == nil {
		t.Fatal("Expected at least one undeletable page in seed data")
	}

	err = svc.Delete(undeletablePage.ID, accountID)
	if err != ErrTemplatePageCannotBeDeleted {
		t.Errorf("Expected ErrTemplatePageCannotBeDeleted, got %v", err)
	}
}

func TestTemplatePageUpdatePosition(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	created, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name: "Reorder", Slug: "reorder", Position: 1,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.UpdatePosition(created.ID, accountID, 42); err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	page, err := svc.Get(created.ID, accountID)
	if err != nil {
		t.Fatalf("Get after UpdatePosition failed: %v", err)
	}
	if page.Position != 42 {
		t.Errorf("Expected position 42, got %d", page.Position)
	}
}

func TestTemplatePageModuleAdd(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	db := svc.db
	modName := "Test Module"
	mod := models.Module{AccountID: accountID, Name: &modName}
	if err := db.Create(&mod).Error; err != nil {
		t.Fatalf("Create module failed: %v", err)
	}

	// Create a fresh page to avoid seed module interference
	freshPage, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name: "Module Test Page",
	})
	if err != nil {
		t.Fatalf("Create page failed: %v", err)
	}
	pageID := freshPage.ID

	if err := svc.AddModule(pageID, accountID, dto.AddModuleToPageRequest{
		ModuleID: mod.ID, Position: 1,
	}); err != nil {
		t.Fatalf("AddModule failed: %v", err)
	}

	modules, err := svc.ListModules(pageID, accountID)
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("Expected 1 module, got %d", len(modules))
	}
	if modules[0].ModuleID != mod.ID {
		t.Errorf("Expected module ID %d, got %d", mod.ID, modules[0].ModuleID)
	}
	if modules[0].ModuleName != "Test Module" {
		t.Errorf("Expected module name 'Test Module', got '%s'", modules[0].ModuleName)
	}
}

func TestTemplatePageModuleRemove(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	db := svc.db
	modName := "Remove Me"
	mod := models.Module{AccountID: accountID, Name: &modName}
	if err := db.Create(&mod).Error; err != nil {
		t.Fatalf("Create module failed: %v", err)
	}

	freshPage, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name: "Remove Test Page",
	})
	if err != nil {
		t.Fatalf("Create page failed: %v", err)
	}
	pageID := freshPage.ID

	if err := svc.AddModule(pageID, accountID, dto.AddModuleToPageRequest{
		ModuleID: mod.ID, Position: 1,
	}); err != nil {
		t.Fatalf("AddModule failed: %v", err)
	}

	if err := svc.RemoveModule(pageID, mod.ID, accountID); err != nil {
		t.Fatalf("RemoveModule failed: %v", err)
	}

	modules, err := svc.ListModules(pageID, accountID)
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	if len(modules) != 0 {
		t.Errorf("Expected 0 modules after remove, got %d", len(modules))
	}
}

func TestTemplatePageModuleUpdatePosition(t *testing.T) {
	svc, accountID, templateID := setupTemplatePageTest(t)

	db := svc.db
	modName := "Reorder Module"
	mod := models.Module{AccountID: accountID, Name: &modName}
	if err := db.Create(&mod).Error; err != nil {
		t.Fatalf("Create module failed: %v", err)
	}

	freshPage, err := svc.Create(templateID, accountID, dto.CreateTemplatePageRequest{
		Name: "Position Test Page",
	})
	if err != nil {
		t.Fatalf("Create page failed: %v", err)
	}
	pageID := freshPage.ID

	if err := svc.AddModule(pageID, accountID, dto.AddModuleToPageRequest{
		ModuleID: mod.ID, Position: 1,
	}); err != nil {
		t.Fatalf("AddModule failed: %v", err)
	}

	if err := svc.UpdateModulePosition(pageID, mod.ID, accountID, 99); err != nil {
		t.Fatalf("UpdateModulePosition failed: %v", err)
	}

	modules, err := svc.ListModules(pageID, accountID)
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("Expected 1 module, got %d", len(modules))
	}
	if modules[0].Position != 99 {
		t.Errorf("Expected position 99, got %d", modules[0].Position)
	}
}

func TestTemplatePageNotFound(t *testing.T) {
	svc, accountID, _ := setupTemplatePageTest(t)

	_, err := svc.Get(99999, accountID)
	if err != ErrTemplatePageNotFound {
		t.Errorf("Expected ErrTemplatePageNotFound, got %v", err)
	}
}

func TestTemplatePageWrongAccount(t *testing.T) {
	svc, _, templateID := setupTemplatePageTest(t)

	_, err := svc.List(templateID, "wrong-account-id")
	if err != ErrTemplateNotFound {
		t.Errorf("Expected ErrTemplateNotFound for wrong account, got %v", err)
	}
}
