package services

import (
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardSubscriptionStandaloneTitleRemoval(t *testing.T) {
	for _, identity := range []string{"subscription contact ID", "legacy URI"} {
		t.Run(identity, func(t *testing.T) {
			svc, _, vaultID, userID := setupVCardTest(t)
			var owner models.User
			if err := svc.db.First(&owner, "id = ?", userID).Error; err != nil {
				t.Fatal(err)
			}
			card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Title\r\nTITLE:Independent consultant\r\nEND:VCARD\r\n")).Decode()
			if err != nil {
				t.Fatal(err)
			}
			var id string
			apply := func(etag, wantAction string) {
				t.Helper()
				err := svc.db.Transaction(func(tx *gorm.DB) error {
					existingID := id
					if identity == "legacy URI" {
						existingID = ""
					}
					newID, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, owner.AccountID, existingID, "/standalone-title.vcf", etag, nil)
					if err != nil {
						return err
					}
					if action != wantAction || (id != "" && newID != id) {
						t.Fatalf("unexpected sync result %s %s", newID, action)
					}
					id = newID
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			assertTitle := func(want, etag string) {
				t.Helper()
				var contact models.Contact
				if err := svc.db.Preload("ContactCompanies").First(&contact, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if contact.CompanyID != nil || len(contact.ContactCompanies) != 0 || ptrToStr(contact.JobPosition) != want || (want == "" && contact.JobPosition != nil) || ptrToStr(contact.DistantEtag) != etag {
					t.Fatalf("stored title=%q etag=%q jobs=%d company=%v", ptrToStr(contact.JobPosition), ptrToStr(contact.DistantEtag), len(contact.ContactCompanies), contact.CompanyID)
				}
				wire, err := svc.ExportContact(id, vaultID)
				if err != nil {
					t.Fatal(err)
				}
				exported, err := vcard.NewDecoder(strings.NewReader(string(wire))).Decode()
				if err != nil {
					t.Fatal(err)
				}
				if exported.Value(vcard.FieldTitle) != want || (want == "" && len(exported[vcard.FieldTitle]) != 0) {
					t.Fatalf("exported TITLE=%q, want %q", exported.Value(vcard.FieldTitle), want)
				}
			}
			apply("initial", "created")
			assertTitle("Independent consultant", "initial")
			delete(card, vcard.FieldTitle)
			apply("title-removed", "updated")
			assertTitle("", "title-removed")
			apply("title-removed", "skipped")
			assertTitle("", "title-removed")
			card.SetValue(vcard.FieldTitle, "Restored consultant")
			// Birthday import calls the shared incremental field importer with only BDAY.
			// It must not clear the independently restored title in this full replacement.
			card.SetValue(vcard.FieldBirthday, "1990-05-12")
			apply("title-restored", "updated")
			assertTitle("Restored consultant", "title-restored")
			var dates []models.ContactImportantDate
			if err := svc.db.Where("contact_id = ?", id).Find(&dates).Error; err != nil {
				t.Fatal(err)
			}
			if len(dates) != 1 || dates[0].Year == nil || *dates[0].Year != 1990 || dates[0].Month == nil || *dates[0].Month != 5 || dates[0].Day == nil || *dates[0].Day != 12 {
				t.Fatalf("birthday not persisted: %+v", dates)
			}
		})
	}
}

func TestVCardOmittedEmploymentPreservesAssociatedJobs(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "normal Jobs"
		if legacy {
			name = "legacy company"
		}
		t.Run(name, func(t *testing.T) {
			svc, contactID, vaultID, userID := setupVCardTest(t)
			var owner models.User
			if err := svc.db.First(&owner, "id = ?", userID).Error; err != nil {
				t.Fatal(err)
			}
			company := models.Company{VaultID: vaultID, Name: "Retained employer"}
			if err := svc.db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			if legacy {
				if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Updates(map[string]any{"company_id": company.ID, "job_position": "Engineer"}).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := NewContactJobService(svc.db).Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"}); err != nil {
					t.Fatal(err)
				}
			}
			before, err := loadContactEmployment(svc.db, contactID, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Changed Name\r\nEND:VCARD\r\n")).Decode()
			if err != nil {
				t.Fatal(err)
			}
			err = svc.db.Transaction(func(tx *gorm.DB) error {
				id, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, owner.AccountID, contactID, "/employment.vcf", "omitted-employment", nil)
				if err == nil && (id != contactID || action != "updated") {
					t.Fatalf("unexpected sync result %s %s", id, action)
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			after, err := loadContactEmployment(svc.db, contactID, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			if ptrToStr(after.FirstName) != "Changed Name" || !reflect.DeepEqual(after.CompanyID, before.CompanyID) || !reflect.DeepEqual(after.JobPosition, before.JobPosition) || len(after.ContactCompanies) != len(before.ContactCompanies) {
				t.Fatal("omitted employment changed company/title/jobs or lost name edit")
			}
			for i, job := range before.ContactCompanies {
				got := after.ContactCompanies[i]
				if got.ID != job.ID || got.CompanyID != job.CompanyID || ptrToStr(got.JobPosition) != ptrToStr(job.JobPosition) || !got.UpdatedAt.Equal(job.UpdatedAt) {
					t.Fatal("omitted employment mutated a retained job")
				}
			}
			wire, err := svc.ExportContact(contactID, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			exported, err := vcard.NewDecoder(strings.NewReader(string(wire))).Decode()
			if err != nil {
				t.Fatal(err)
			}
			if exported.Value(vcard.FieldOrganization) != "Retained employer" || exported.Value(vcard.FieldTitle) != "Engineer" {
				t.Fatal("omitted employment no longer exported")
			}
		})
	}
}
