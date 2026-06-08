package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupGiftContactModuleBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Account{}, &Template{}, &TemplatePage{}, &Module{}, &ModuleTemplatePage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func createBackfillAccountWithInformationPage(t *testing.T, db *gorm.DB, accountID string) (TemplatePage, TemplatePage) {
	t.Helper()
	account := Account{ID: accountID}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}

	defaultTemplateName := "Default template"
	defaultTemplate := Template{AccountID: accountID, Name: &defaultTemplateName}
	if err := db.Create(&defaultTemplate).Error; err != nil {
		t.Fatalf("create default template: %v", err)
	}
	if err := db.Model(&defaultTemplate).Update("can_be_deleted", false).Error; err != nil {
		t.Fatalf("mark default template undeletable: %v", err)
	}

	position := 5
	informationPage := TemplatePage{TemplateID: defaultTemplate.ID, Slug: "information", Position: &position}
	if err := db.Create(&informationPage).Error; err != nil {
		t.Fatalf("create default information page: %v", err)
	}

	customTemplateName := "Custom template"
	customTemplate := Template{AccountID: accountID, Name: &customTemplateName}
	if err := db.Create(&customTemplate).Error; err != nil {
		t.Fatalf("create custom template: %v", err)
	}
	customInformationPage := TemplatePage{TemplateID: customTemplate.ID, Slug: "information", Position: &position}
	if err := db.Create(&customInformationPage).Error; err != nil {
		t.Fatalf("create custom information page: %v", err)
	}

	return informationPage, customInformationPage
}

func createBackfillModule(t *testing.T, db *gorm.DB, accountID, name, moduleType string) Module {
	t.Helper()
	module := Module{AccountID: accountID, Name: &name, Type: &moduleType}
	if err := db.Create(&module).Error; err != nil {
		t.Fatalf("create module %s: %v", name, err)
	}
	return module
}

func bindBackfillModule(t *testing.T, db *gorm.DB, pageID, moduleID uint, position int) {
	t.Helper()
	if err := db.Create(&ModuleTemplatePage{TemplatePageID: pageID, ModuleID: moduleID, Position: &position}).Error; err != nil {
		t.Fatalf("bind module %d to page %d: %v", moduleID, pageID, err)
	}
}

func TestBackfillGiftContactModulesAddsDefaultModuleToInformationPage(t *testing.T) {
	db := setupGiftContactModuleBackfillDB(t)
	informationPage, customInformationPage := createBackfillAccountWithInformationPage(t, db, "account-gifts-missing")

	loans := createBackfillModule(t, db, "account-gifts-missing", "Loans", "loans")
	tasks := createBackfillModule(t, db, "account-gifts-missing", "Tasks", "tasks")
	customModule := createBackfillModule(t, db, "account-gifts-missing", "Custom", "custom")
	bindBackfillModule(t, db, informationPage.ID, loans.ID, 5)
	bindBackfillModule(t, db, informationPage.ID, tasks.ID, 8)
	bindBackfillModule(t, db, customInformationPage.ID, customModule.ID, 1)

	if err := BackfillGiftContactModules(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if err := BackfillGiftContactModules(db); err != nil {
		t.Fatalf("backfill second run: %v", err)
	}

	var modules []Module
	if err := db.Where(
		"account_id = ? AND type = ? AND name_translation_key = ?",
		"account-gifts-missing",
		defaultGiftContactModuleType,
		defaultGiftContactModuleTranslationKey,
	).Find(&modules).Error; err != nil {
		t.Fatalf("list gift modules: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected one default gifts module, got %d", len(modules))
	}
	if modules[0].Name == nil || *modules[0].Name != "Gifts" {
		t.Fatalf("expected default gift module name Gifts, got %v", modules[0].Name)
	}
	if modules[0].CanBeDeleted {
		t.Fatal("default gift module should be non-deletable")
	}

	var defaultBindings []ModuleTemplatePage
	if err := db.Where("template_page_id = ? AND module_id = ?", informationPage.ID, modules[0].ID).Find(&defaultBindings).Error; err != nil {
		t.Fatalf("list default page gift bindings: %v", err)
	}
	if len(defaultBindings) != 1 {
		t.Fatalf("expected one default information page binding, got %d", len(defaultBindings))
	}
	if defaultBindings[0].Position == nil || *defaultBindings[0].Position != 9 {
		t.Fatalf("expected gift module to append at position 9, got %v", defaultBindings[0].Position)
	}

	var customBindingCount int64
	if err := db.Model(&ModuleTemplatePage{}).
		Where("template_page_id = ? AND module_id = ?", customInformationPage.ID, modules[0].ID).
		Count(&customBindingCount).Error; err != nil {
		t.Fatalf("count custom page gift bindings: %v", err)
	}
	if customBindingCount != 0 {
		t.Fatalf("expected custom information page to remain untouched, got %d gifts bindings", customBindingCount)
	}
}

func TestBackfillGiftContactModulesReusesExistingDefaultModule(t *testing.T) {
	db := setupGiftContactModuleBackfillDB(t)
	informationPage, _ := createBackfillAccountWithInformationPage(t, db, "account-gifts-existing")

	name := "Cadeaux"
	moduleType := defaultGiftContactModuleType
	translationKey := defaultGiftContactModuleTranslationKey
	existingModule := Module{
		AccountID:          "account-gifts-existing",
		Name:               &name,
		NameTranslationKey: &translationKey,
		Type:               &moduleType,
	}
	if err := db.Create(&existingModule).Error; err != nil {
		t.Fatalf("create existing gift module: %v", err)
	}

	if err := BackfillGiftContactModules(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var modules []Module
	if err := db.Where(
		"account_id = ? AND type = ? AND name_translation_key = ?",
		"account-gifts-existing",
		defaultGiftContactModuleType,
		defaultGiftContactModuleTranslationKey,
	).Find(&modules).Error; err != nil {
		t.Fatalf("list gift modules: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected existing default gifts module to be reused, got %d modules", len(modules))
	}
	if modules[0].ID != existingModule.ID {
		t.Fatalf("expected module ID %d to be reused, got %d", existingModule.ID, modules[0].ID)
	}
	if modules[0].Name == nil || *modules[0].Name != "Cadeaux" {
		t.Fatalf("expected localized module name to be preserved, got %v", modules[0].Name)
	}
	if modules[0].CanBeDeleted {
		t.Fatal("reused default gift module should be non-deletable")
	}

	var bindingCount int64
	if err := db.Model(&ModuleTemplatePage{}).Where("template_page_id = ? AND module_id = ?", informationPage.ID, existingModule.ID).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count gift bindings: %v", err)
	}
	if bindingCount != 1 {
		t.Fatalf("expected one binding for reused gift module, got %d", bindingCount)
	}
}

func TestBackfillGiftContactModulesDoesNotDuplicateExistingBinding(t *testing.T) {
	db := setupGiftContactModuleBackfillDB(t)
	informationPage, _ := createBackfillAccountWithInformationPage(t, db, "account-gifts-bound")

	name := "Gifts"
	moduleType := defaultGiftContactModuleType
	translationKey := defaultGiftContactModuleTranslationKey
	existingModule := Module{
		AccountID:          "account-gifts-bound",
		Name:               &name,
		NameTranslationKey: &translationKey,
		Type:               &moduleType,
	}
	if err := db.Create(&existingModule).Error; err != nil {
		t.Fatalf("create existing gift module: %v", err)
	}
	bindBackfillModule(t, db, informationPage.ID, existingModule.ID, 3)

	if err := BackfillGiftContactModules(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if err := BackfillGiftContactModules(db); err != nil {
		t.Fatalf("backfill second run: %v", err)
	}

	var bindings []ModuleTemplatePage
	if err := db.Where("template_page_id = ? AND module_id = ?", informationPage.ID, existingModule.ID).Find(&bindings).Error; err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected one existing binding after repeated backfills, got %d", len(bindings))
	}
	if bindings[0].Position == nil || *bindings[0].Position != 3 {
		t.Fatalf("expected existing binding position 3 to remain, got %v", bindings[0].Position)
	}
}
