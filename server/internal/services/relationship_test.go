package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupRelationshipTest(t *testing.T) (*RelationshipService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "relationship-test@example.com",
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

	relatedContact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact (related) failed: %v", err)
	}

	return NewRelationshipService(db), contact.ID, relatedContact.ID
}

func TestCreateRelationship(t *testing.T) {
	svc, contactID, relatedContactID := setupRelationshipTest(t)

	rel, err := svc.Create(contactID, dto.CreateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if rel.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, rel.ContactID)
	}
	if rel.RelatedContactID != relatedContactID {
		t.Errorf("Expected related_contact_id '%s', got '%s'", relatedContactID, rel.RelatedContactID)
	}
	if rel.RelationshipTypeID != 1 {
		t.Errorf("Expected relationship_type_id 1, got %d", rel.RelationshipTypeID)
	}
	if rel.ID == 0 {
		t.Error("Expected relationship ID to be non-zero")
	}
}

func TestListRelationships(t *testing.T) {
	svc, contactID, relatedContactID := setupRelationshipTest(t)

	_, err := svc.Create(contactID, dto.CreateRelationshipRequest{RelationshipTypeID: 1, RelatedContactID: relatedContactID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, dto.CreateRelationshipRequest{RelationshipTypeID: 2, RelatedContactID: relatedContactID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	relationships, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(relationships))
	}
}

func TestUpdateRelationship(t *testing.T) {
	svc, contactID, relatedContactID := setupRelationshipTest(t)

	created, err := svc.Create(contactID, dto.CreateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, dto.UpdateRelationshipRequest{
		RelationshipTypeID: 3,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.RelationshipTypeID != 3 {
		t.Errorf("Expected relationship_type_id 3, got %d", updated.RelationshipTypeID)
	}
}

func TestDeleteRelationship(t *testing.T) {
	svc, contactID, relatedContactID := setupRelationshipTest(t)

	created, err := svc.Create(contactID, dto.CreateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	relationships, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(relationships) != 0 {
		t.Errorf("Expected 0 relationships after delete, got %d", len(relationships))
	}
}

func TestRelationshipNotFound(t *testing.T) {
	svc, contactID, relatedContactID := setupRelationshipTest(t)

	_, err := svc.Update(9999, contactID, dto.UpdateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != ErrRelationshipNotFound {
		t.Errorf("Expected ErrRelationshipNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID)
	if err != ErrRelationshipNotFound {
		t.Errorf("Expected ErrRelationshipNotFound, got %v", err)
	}
}
