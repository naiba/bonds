package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestCreateRelationship_WithExternalContactCreatesHiddenNeedsVerificationContact(t *testing.T) {
	ctx := setupRelationshipTestFull(t)

	relationship, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
		RelationshipTypeID:  1,
		ExternalContactName: "Aunt Mary",
	})
	if err != nil {
		t.Fatalf("Create relationship with external contact failed: %v", err)
	}
	if relationship.RelatedContactID == "" || relationship.RelatedContactID == ctx.relatedContactID {
		t.Fatalf("Expected a new related contact ID, got %q", relationship.RelatedContactID)
	}
	if relationship.RelatedContactName != "Aunt Mary" {
		t.Fatalf("Expected related contact name Aunt Mary, got %q", relationship.RelatedContactName)
	}

	var externalContact models.Contact
	if err := ctx.db.First(&externalContact, "id = ?", relationship.RelatedContactID).Error; err != nil {
		t.Fatalf("Load external contact failed: %v", err)
	}
	if externalContact.VaultID != ctx.vaultID {
		t.Fatalf("Expected external contact in vault %s, got %s", ctx.vaultID, externalContact.VaultID)
	}
	if externalContact.Listed {
		t.Fatalf("Expected external contact to be hidden from the main list")
	}
	if !externalContact.NeedsVerification {
		t.Fatalf("Expected external contact to be marked as needing verification")
	}
	if !externalContact.CanBeDeleted {
		t.Fatalf("Expected external contact to remain user-deletable")
	}

	contacts, _, err := NewContactService(ctx.db).ListContacts(ctx.vaultID, ctx.userID, 1, 20, "Aunt Mary", "first_name", "")
	if err != nil {
		t.Fatalf("List contacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Fatalf("Expected external contact hidden from active contact list, got %+v", contacts)
	}

	relationships, err := ctx.svc.List(ctx.contactID, ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("List relationships failed: %v", err)
	}
	if !relationshipListContainsContact(relationships, relationship.RelatedContactID, "Aunt Mary") {
		t.Fatalf("Expected relationships list to include hidden external contact, got %+v", relationships)
	}
}

func TestUpdateContact_PromotesExternalRelationshipContact(t *testing.T) {
	ctx := setupRelationshipTestFull(t)

	relationship, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
		RelationshipTypeID:  1,
		ExternalContactName: "Aunt Mary",
	})
	if err != nil {
		t.Fatalf("Create relationship with external contact failed: %v", err)
	}

	listed := true
	needsVerification := false
	contactSvc := NewContactService(ctx.db)
	updated, err := contactSvc.UpdateContact(relationship.RelatedContactID, ctx.vaultID, ctx.userID, dto.UpdateContactRequest{
		FirstName:         "Aunt Mary",
		Listed:            &listed,
		NeedsVerification: &needsVerification,
	})
	if err != nil {
		t.Fatalf("Promote external contact failed: %v", err)
	}
	if !updated.Listed {
		t.Fatalf("Expected promoted contact to be listed")
	}
	if updated.NeedsVerification {
		t.Fatalf("Expected promoted contact to clear needs_verification")
	}

	contacts, _, err := contactSvc.ListContacts(ctx.vaultID, ctx.userID, 1, 20, "Aunt Mary", "first_name", "")
	if err != nil {
		t.Fatalf("List contacts failed: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ID != relationship.RelatedContactID {
		t.Fatalf("Expected promoted contact in active list, got %+v", contacts)
	}
}

func relationshipListContainsContact(relationships []dto.RelationshipResponse, contactID, contactName string) bool {
	for _, relationship := range relationships {
		if relationship.RelatedContactID == contactID && relationship.RelatedContactName == contactName {
			return true
		}
	}
	return false
}
