package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupLoanTest(t *testing.T) (*LoanService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "loan-test@example.com",
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

	return NewLoanService(db), contact.ID, vault.ID
}

func TestCreateLoan(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)

	loan, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{
		Name:        "Book",
		Type:        "object",
		Description: "Lent a book",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if loan.Name != "Book" {
		t.Errorf("Expected name 'Book', got '%s'", loan.Name)
	}
	if loan.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", loan.Type)
	}
	if loan.Description != "Lent a book" {
		t.Errorf("Expected description 'Lent a book', got '%s'", loan.Description)
	}
	if loan.Category != "money" {
		t.Errorf("Expected default category 'money', got '%s'", loan.Category)
	}
	if loan.Settled {
		t.Error("Expected loan to not be settled")
	}
	if loan.ID == 0 {
		t.Error("Expected loan ID to be non-zero")
	}
}

func TestCreateItemLoanIncludesLendingFields(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)
	dueAt := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	quantity := uint(2)

	loan, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{
		Name:        "Board game",
		Type:        "lent",
		Category:    "item",
		ItemName:    "Catan",
		Quantity:    &quantity,
		DueAt:       &dueAt,
		Description: "Expansion included",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if loan.Category != "item" {
		t.Fatalf("expected category=item, got %q", loan.Category)
	}
	if loan.ItemName != "Catan" {
		t.Fatalf("expected item_name=Catan, got %q", loan.ItemName)
	}
	if loan.Quantity == nil || *loan.Quantity != quantity {
		t.Fatalf("expected quantity=%d, got %v", quantity, loan.Quantity)
	}
	if loan.DueAt == nil || !loan.DueAt.Equal(dueAt) {
		t.Fatalf("expected due_at=%s, got %v", dueAt.Format(time.RFC3339), loan.DueAt)
	}
	if loan.ReturnedAt != nil {
		t.Fatalf("expected returned_at to start nil, got %v", loan.ReturnedAt)
	}
	if loan.AmountLent != nil || loan.CurrencyID != nil {
		t.Fatalf("expected item loan to preserve nil money fields, got amount=%v currency=%v", loan.AmountLent, loan.CurrencyID)
	}
}

func TestListLegacyLoanWithoutCategoryDefaultsToMoney(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)
	loan := models.Loan{VaultID: vaultID, Name: "Legacy loan", Type: "debt"}
	if err := svc.db.Create(&loan).Error; err != nil {
		t.Fatalf("create legacy loan failed: %v", err)
	}
	if err := svc.db.Create(&models.ContactLoan{LoanID: loan.ID, LoanerID: contactID, LoaneeID: contactID}).Error; err != nil {
		t.Fatalf("create legacy contact loan failed: %v", err)
	}

	loans, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(loans) != 1 {
		t.Fatalf("expected 1 loan, got %d", len(loans))
	}
	if loans[0].Category != "money" {
		t.Fatalf("expected legacy empty category to return money, got %q", loans[0].Category)
	}
}

func TestListLoans(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{Name: "Loan 1", Type: "object"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreateLoanRequest{Name: "Loan 2", Type: "monetary"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	loans, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(loans) != 2 {
		t.Errorf("Expected 2 loans, got %d", len(loans))
	}
}

func TestUpdateLoan(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{Name: "Original", Type: "object"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, vaultID, dto.UpdateLoanRequest{
		Name:        "Updated",
		Type:        "monetary",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", updated.Name)
	}
	if updated.Type != "monetary" {
		t.Errorf("Expected type 'monetary', got '%s'", updated.Type)
	}
}

func TestUpdateItemLoanFields(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)
	created, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{Name: "Original item", Type: "lent", Category: "item", ItemName: "Tent"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	dueAt := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	quantity := uint(1)

	updated, err := svc.Update(created.ID, vaultID, dto.UpdateLoanRequest{
		Name:     "Updated item",
		Type:     "borrowed",
		Category: "item",
		ItemName: "Camping tent",
		Quantity: &quantity,
		DueAt:    &dueAt,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Category != "item" || updated.ItemName != "Camping tent" {
		t.Fatalf("unexpected item fields after update: %+v", updated)
	}
	if updated.Quantity == nil || *updated.Quantity != quantity {
		t.Fatalf("expected quantity=%d, got %v", quantity, updated.Quantity)
	}
	if updated.DueAt == nil || !updated.DueAt.Equal(dueAt) {
		t.Fatalf("expected due_at=%s, got %v", dueAt.Format(time.RFC3339), updated.DueAt)
	}
}

func TestDeleteLoan(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{Name: "To delete", Type: "object"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	loans, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(loans) != 0 {
		t.Errorf("Expected 0 loans after delete, got %d", len(loans))
	}
}

func TestToggleLoanSettled(t *testing.T) {
	svc, contactID, vaultID := setupLoanTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateLoanRequest{Name: "Toggle me", Type: "object"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	toggled, err := svc.ToggleSettled(created.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleSettled failed: %v", err)
	}
	if !toggled.Settled {
		t.Error("Expected loan to be settled after toggle")
	}
	if toggled.SettledAt == nil {
		t.Error("Expected SettledAt to be set")
	}
	if toggled.ReturnedAt == nil {
		t.Error("Expected ReturnedAt to be set with SettledAt")
	}

	toggledBack, err := svc.ToggleSettled(created.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleSettled back failed: %v", err)
	}
	if toggledBack.Settled {
		t.Error("Expected loan to not be settled after second toggle")
	}
	if toggledBack.SettledAt != nil {
		t.Error("Expected SettledAt to be nil after second toggle")
	}
	if toggledBack.ReturnedAt != nil {
		t.Error("Expected ReturnedAt to be nil after second toggle")
	}
}

func TestLoanNotFound(t *testing.T) {
	svc, _, vaultID := setupLoanTest(t)

	_, err := svc.Update(9999, vaultID, dto.UpdateLoanRequest{Name: "nope", Type: "object"})
	if err != ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}

	err = svc.Delete(9999, vaultID)
	if err != ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}

	_, err = svc.ToggleSettled(9999, vaultID)
	if err != ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}
