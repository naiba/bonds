package services

import (
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardOrganizationImportAndSubscription(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)
	body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Person\r\nN:Person;Synthetic;;;\r\nORG:Synthetic Company\r\nTITLE:Engineer\r\nEND:VCARD\r\n"
	imported, err := svc.ImportVCard(vaultID, userID, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if imported.ImportedCount != 1 || len(imported.Contacts) != 1 {
		t.Fatalf("unexpected import: %+v", imported)
	}
	assertCompany := func(id, name string) uint {
		t.Helper()
		var contact models.Contact
		if err := svc.db.Preload("Company").First(&contact, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if contact.Company == nil || contact.Company.Name != name || contact.Company.VaultID != vaultID {
			t.Fatalf("wrong company: %+v", contact.Company)
		}
		var jobs []models.ContactCompany
		if err := svc.db.Where("contact_id = ? AND company_id = ?", id, contact.Company.ID).Find(&jobs).Error; err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 || ptrToStr(jobs[0].JobPosition) != "Engineer" {
			t.Fatalf("wrong employment: %+v", jobs)
		}
		return contact.Company.ID
	}
	companyID := assertCompany(imported.Contacts[0].ID, "Synthetic Company")
	exported, err := svc.ExportContact(imported.Contacts[0].ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	exportedCard, err := vcard.NewDecoder(strings.NewReader(string(exported))).Decode()
	if err != nil || exportedCard.Value(vcard.FieldOrganization) != "Synthetic Company" {
		t.Fatalf("ORG export: %v %v", exportedCard, err)
	}
	var user models.User
	if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	card, err := vcard.NewDecoder(strings.NewReader(body)).Decode()
	if err != nil {
		t.Fatal(err)
	}
	var syncedID string
	syncCard := func(etag, wantAction string) {
		t.Helper()
		err := svc.db.Transaction(func(tx *gorm.DB) error {
			id, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, syncedID, "/synthetic-person.vcf", etag, nil)
			if err != nil {
				return err
			}
			if action != wantAction {
				t.Fatalf("action = %s, want %s", action, wantAction)
			}
			syncedID = id
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	syncCard("created", "created")
	if got := assertCompany(syncedID, "Synthetic Company"); got != companyID {
		t.Fatalf("did not reuse imported company: %d != %d", got, companyID)
	}
	syncCard("same-organization", "updated")
	assertCompany(syncedID, "Synthetic Company")
	card.SetValue(vcard.FieldOrganization, "New Company")
	syncCard("changed-organization", "updated")
	assertCompany(syncedID, "New Company")
	delete(card, vcard.FieldOrganization)
	syncCard("omitted-organization", "updated")
	assertCompany(syncedID, "New Company")
	var jobs int64
	if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ?", syncedID).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if jobs != 2 {
		t.Fatalf("changed ORG must preserve previous employment without duplicates, got %d", jobs)
	}
}
