package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupSliceOfLifeTest(t *testing.T) (*SliceOfLifeService, uint, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "sol-test@example.com", Password: "password123",
	})
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})

	journalSvc := NewJournalService(db)
	journal, _ := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})

	return NewSliceOfLifeService(db), journal.ID, vault.ID
}

func TestCreateSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	slice, err := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{
		Name: "Summer 2024", Description: "My summer adventures",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if slice.Name != "Summer 2024" {
		t.Errorf("Expected name 'Summer 2024', got '%s'", slice.Name)
	}
}

func TestListSlicesOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	_, _ = svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "Slice 1"})
	_, _ = svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "Slice 2"})

	slices, err := svc.List(journalID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(slices) != 2 {
		t.Errorf("Expected 2 slices, got %d", len(slices))
	}
}

func TestGetSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	created, _ := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "My Slice"})
	got, err := svc.Get(created.ID, journalID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "My Slice" {
		t.Errorf("Expected name 'My Slice', got '%s'", got.Name)
	}
}

func TestUpdateSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	created, _ := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "Old"})
	updated, err := svc.Update(created.ID, journalID, vaultID, dto.UpdateSliceOfLifeRequest{Name: "New"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("Expected name 'New', got '%s'", updated.Name)
	}
}

func TestDeleteSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	created, _ := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "ToDelete"})
	if err := svc.Delete(created.ID, journalID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	slices, _ := svc.List(journalID, vaultID)
	if len(slices) != 0 {
		t.Errorf("Expected 0 slices, got %d", len(slices))
	}
}

func TestSliceOfLifeNotFound(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	_, err := svc.Get(9999, journalID, vaultID)
	if err != ErrSliceOfLifeNotFound {
		t.Errorf("Expected ErrSliceOfLifeNotFound, got %v", err)
	}
}
