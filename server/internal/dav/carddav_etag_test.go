package dav

import (
	"fmt"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
)

func TestGetAddressObjectChangesETagWhenContactInformationChanges(t *testing.T) {
	// Given a CardDAV contact whose current card and database timestamp are known.
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Alice", "Smith")
	path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID)

	firstObject, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatalf("GetAddressObject before contact information change failed: %v", err)
	}
	var before models.Contact
	if err := db.First(&before, "id = ?", contact.ID).Error; err != nil {
		t.Fatalf("load contact before contact information change: %v", err)
	}

	var vault models.Vault
	if err := db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("load vault: %v", err)
	}
	var phoneType models.ContactInformationType
	if err := db.Where("account_id = ? AND type = ?", vault.AccountID, "phone").First(&phoneType).Error; err != nil {
		t.Fatalf("load seeded phone type: %v", err)
	}

	// When a phone number is added without updating the Contact row.
	phoneValue := "+1-555-0100"
	if err := db.Create(&models.ContactInformation{
		ContactID: contact.ID,
		TypeID:    phoneType.ID,
		Data:      phoneValue,
	}).Error; err != nil {
		t.Fatalf("create phone contact information: %v", err)
	}

	secondObject, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatalf("GetAddressObject after contact information change failed: %v", err)
	}
	var after models.Contact
	if err := db.First(&after, "id = ?", contact.ID).Error; err != nil {
		t.Fatalf("load contact after contact information change: %v", err)
	}

	// Then the vCard includes the phone, the Contact timestamp is unchanged, and the ETag changes.
	if got := secondObject.Card.Value(vcard.FieldTelephone); got != phoneValue {
		t.Fatalf("expected second card TEL %q, got %q", phoneValue, got)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("expected Contact.UpdatedAt to remain unchanged, before %v after %v", before.UpdatedAt, after.UpdatedAt)
	}
	if secondObject.ETag == firstObject.ETag {
		t.Fatalf("expected ETag to change after adding phone %q, both were %q", phoneValue, firstObject.ETag)
	}
}
