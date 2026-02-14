package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
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
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
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
	if loan.Settled {
		t.Error("Expected loan to not be settled")
	}
	if loan.ID == 0 {
		t.Error("Expected loan ID to be non-zero")
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
