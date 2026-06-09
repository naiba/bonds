package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupVaultQuickFactTplTest(t *testing.T) (*VaultQuickFactTemplateService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vault-qft-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	return NewVaultQuickFactTemplateService(db), vault.ID
}

func TestVaultQuickFactTemplateCRUD(t *testing.T) {
	svc, vaultID := setupVaultQuickFactTplTest(t)

	tpls, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	seedCount := len(tpls)

	pos := 5
	created, err := svc.Create(vaultID, dto.CreateQuickFactTemplateRequest{Label: "Hobbies", Position: &pos})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Label != "Hobbies" {
		t.Errorf("Expected label 'Hobbies', got '%s'", created.Label)
	}

	pos2 := 10
	updated, err := svc.Update(created.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Sports", Position: &pos2})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Sports" {
		t.Errorf("Expected label 'Sports', got '%s'", updated.Label)
	}

	posUpdated, err := svc.UpdatePosition(created.ID, vaultID, 1)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}
	if posUpdated.Position != 1 {
		t.Errorf("Expected position 1, got %d", posUpdated.Position)
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	tpls, _ = svc.List(vaultID)
	if len(tpls) != seedCount {
		t.Errorf("Expected %d templates after delete, got %d", seedCount, len(tpls))
	}
}

func TestVaultQuickFactTemplateNotFound(t *testing.T) {
	svc, vaultID := setupVaultQuickFactTplTest(t)

	err := svc.Delete(9999, vaultID)
	if err != ErrQuickFactTplNotFound {
		t.Errorf("Expected ErrQuickFactTplNotFound, got %v", err)
	}
}

func TestVaultQuickFactTemplateRejectsUnsafeFieldTypeChangeWithFacts(t *testing.T) {
	tplSvc, vaultID := setupVaultQuickFactTplTest(t)
	quickFactSvc := NewQuickFactService(tplSvc.db)
	contactID := createQuickFactTemplateTestContact(t, tplSvc.db, vaultID)
	tpl, err := tplSvc.Create(vaultID, dto.CreateQuickFactTemplateRequest{Label: "Score", FieldType: QuickFactFieldNumber})
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	value := 7.0
	if _, err := quickFactSvc.Create(contactID, vaultID, tpl.ID, dto.CreateQuickFactRequest{ValueNumber: &value}); err != nil {
		t.Fatalf("Create quick fact failed: %v", err)
	}
	_, err = tplSvc.Update(tpl.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Score", FieldType: QuickFactFieldText})
	if !errors.Is(err, ErrQuickFactTemplateInUse) {
		t.Fatalf("Expected ErrQuickFactTemplateInUse for field type change, got %v", err)
	}
	helpText := "Updated help"
	updated, err := tplSvc.Update(tpl.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Score updated", FieldType: QuickFactFieldNumber, HelpText: &helpText})
	if err != nil {
		t.Fatalf("Expected compatible metadata update, got %v", err)
	}
	if updated.Label != "Score updated" || updated.HelpText == nil || *updated.HelpText != helpText {
		t.Fatalf("Unexpected compatible update response: %+v", updated)
	}
}

func TestVaultQuickFactTemplateRejectsSelectOptionRemovalWithFacts(t *testing.T) {
	tplSvc, vaultID := setupVaultQuickFactTplTest(t)
	quickFactSvc := NewQuickFactService(tplSvc.db)
	contactID := createQuickFactTemplateTestContact(t, tplSvc.db, vaultID)
	tpl, err := tplSvc.Create(vaultID, dto.CreateQuickFactTemplateRequest{Label: "Choice", FieldType: QuickFactFieldSelect, SelectOptions: []string{"A", "B"}})
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	option := "B"
	if _, err := quickFactSvc.Create(contactID, vaultID, tpl.ID, dto.CreateQuickFactRequest{ValueOption: &option}); err != nil {
		t.Fatalf("Create quick fact failed: %v", err)
	}
	_, err = tplSvc.Update(tpl.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Choice", FieldType: QuickFactFieldSelect, SelectOptions: []string{"A"}})
	if !errors.Is(err, ErrQuickFactTemplateInUse) {
		t.Fatalf("Expected ErrQuickFactTemplateInUse for option removal, got %v", err)
	}
	updated, err := tplSvc.Update(tpl.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Choice", FieldType: QuickFactFieldSelect, SelectOptions: []string{"A", "B", "C"}})
	if err != nil {
		t.Fatalf("Expected compatible option addition, got %v", err)
	}
	if len(updated.SelectOptions) != 3 {
		t.Fatalf("Expected three select options, got %+v", updated.SelectOptions)
	}
}

func TestVaultQuickFactTemplateRejectsRequiredUpgradeWithEmptyFacts(t *testing.T) {
	tplSvc, vaultID := setupVaultQuickFactTplTest(t)
	quickFactSvc := NewQuickFactService(tplSvc.db)
	contactID := createQuickFactTemplateTestContact(t, tplSvc.db, vaultID)
	tpl, err := tplSvc.Create(vaultID, dto.CreateQuickFactTemplateRequest{Label: "Optional note", FieldType: QuickFactFieldText})
	if err != nil {
		t.Fatalf("Create template failed: %v", err)
	}
	if _, err := quickFactSvc.Create(contactID, vaultID, tpl.ID, dto.CreateQuickFactRequest{}); err != nil {
		t.Fatalf("Create empty optional quick fact failed: %v", err)
	}

	_, err = tplSvc.Update(tpl.ID, vaultID, dto.UpdateQuickFactTemplateRequest{Label: "Optional note", FieldType: QuickFactFieldText, Required: true})
	if !errors.Is(err, ErrQuickFactTemplateInUse) {
		t.Fatalf("Expected ErrQuickFactTemplateInUse when required=true would invalidate empty facts, got %v", err)
	}
}

func createQuickFactTemplateTestContact(t *testing.T, db *gorm.DB, vaultID string) string {
	t.Helper()
	contact := models.Contact{VaultID: vaultID}
	firstName := "TemplateFact"
	contact.FirstName = &firstName
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}
	return contact.ID
}
