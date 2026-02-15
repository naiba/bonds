package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPetTest(t *testing.T) (*PetService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "pet-test@example.com",
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

	return NewPetService(db), contact.ID, vault.ID
}

func TestCreatePet(t *testing.T) {
	svc, contactID, vaultID := setupPetTest(t)

	pet, err := svc.Create(contactID, vaultID, dto.CreatePetRequest{
		PetCategoryID: 1,
		Name:          "Buddy",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if pet.Name != "Buddy" {
		t.Errorf("Expected name 'Buddy', got '%s'", pet.Name)
	}
	if pet.PetCategoryID != 1 {
		t.Errorf("Expected pet_category_id 1, got %d", pet.PetCategoryID)
	}
	if pet.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, pet.ContactID)
	}
	if pet.ID == 0 {
		t.Error("Expected pet ID to be non-zero")
	}
}

func TestListPets(t *testing.T) {
	svc, contactID, vaultID := setupPetTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreatePetRequest{PetCategoryID: 1, Name: "Pet 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreatePetRequest{PetCategoryID: 2, Name: "Pet 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	pets, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(pets) != 2 {
		t.Errorf("Expected 2 pets, got %d", len(pets))
	}
}

func TestUpdatePet(t *testing.T) {
	svc, contactID, vaultID := setupPetTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreatePetRequest{PetCategoryID: 1, Name: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdatePetRequest{
		PetCategoryID: 2,
		Name:          "Updated",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", updated.Name)
	}
	if updated.PetCategoryID != 2 {
		t.Errorf("Expected pet_category_id 2, got %d", updated.PetCategoryID)
	}
}

func TestDeletePet(t *testing.T) {
	svc, contactID, vaultID := setupPetTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreatePetRequest{PetCategoryID: 1, Name: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	pets, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(pets) != 0 {
		t.Errorf("Expected 0 pets after delete, got %d", len(pets))
	}
}

func TestPetNotFound(t *testing.T) {
	svc, contactID, vaultID := setupPetTest(t)

	_, err := svc.Update(9999, contactID, vaultID, dto.UpdatePetRequest{PetCategoryID: 1, Name: "nope"})
	if err != ErrPetNotFound {
		t.Errorf("Expected ErrPetNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID, vaultID)
	if err != ErrPetNotFound {
		t.Errorf("Expected ErrPetNotFound, got %v", err)
	}
}
