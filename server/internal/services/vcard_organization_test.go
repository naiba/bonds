package services

import (
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
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

func TestVCardStructuredOrganizationImport(t *testing.T) {
	for _, tc := range []struct{ name, wire, want, exported string }{
		{"department", "Acme;Sales", "Acme", "Acme"},
		{"empty department", "Acme;", "Acme", "Acme"},
		{"empty organization", ";Sales", "", ""},
		{"empty components", ";", "", ""},
		{"escaped semicolon", `Acme\;Partners;Sales`, "Acme;Partners", `Acme\;Partners`},
		{"escaped slash before separator", `Acme\\;Sales`, `Acme\`, `Acme\\`},
		{"slash and literal semicolon", `Acme\\\;Partners;Sales`, `Acme\;Partners`, `Acme\\\;Partners`},
		{"escaped comma", `Acme\, Inc.;Sales`, "Acme, Inc.", `Acme\, Inc.`},
		{"escaped newline", `Acme\nPartners;Sales`, "Acme\nPartners", `Acme\nPartners`},
		{"unicode", `公司\;研发;销售`, "公司;研发", `公司\;研发`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, vaultID, userID := setupVCardTest(t)
			if tc.want != "" {
				if err := svc.db.Create(&models.Company{VaultID: vaultID, Name: tc.want}).Error; err != nil {
					t.Fatal(err)
				}
			}
			body := func(org string) string {
				return "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nORG:" + org + "\r\nEND:VCARD\r\n"
			}
			result, err := svc.ImportVCard(vaultID, userID, strings.NewReader(body(tc.wire)))
			if err != nil {
				t.Fatal(err)
			}
			if result.ImportedCount != 1 {
				t.Fatalf("imported %d", result.ImportedCount)
			}
			var user models.User
			if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
				t.Fatal(err)
			}
			var syncedID string
			for _, revision := range []string{"created", "updated"} {
				wire := tc.wire
				if revision == "updated" {
					wire += ";Remote"
				}
				card, err := vcard.NewDecoder(strings.NewReader(body(wire))).Decode()
				if err != nil {
					t.Fatal(err)
				}
				var action string
				err = svc.db.Transaction(func(tx *gorm.DB) error {
					var err error
					syncedID, action, err = svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, syncedID, "/structured.vcf", revision, nil)
					return err
				})
				if err != nil || action != revision {
					t.Fatalf("subscription %s: %s %v", revision, action, err)
				}
			}
			for _, id := range []string{result.Contacts[0].ID, syncedID} {
				exported, err := svc.ExportContact(id, vaultID)
				if err != nil {
					t.Fatal(err)
				}
				card, err := vcard.NewDecoder(strings.NewReader(string(exported))).Decode()
				if err != nil {
					t.Fatal(err)
				}
				field := card.Get(vcard.FieldOrganization)
				if tc.want == "" {
					if field != nil {
						t.Fatal("empty organization exported as company")
					}
				} else {
					if !strings.Contains(string(exported), "ORG:"+tc.exported+"\r\n") {
						t.Fatalf("wrong organization wire escaping: %q", string(exported))
					}
					parts := field.Organization()
					if len(parts) != 1 || parts[0] != tc.want {
						t.Fatalf("exported components %q, want company %q only", parts, tc.want)
					}
				}
			}
			var companies []models.Company
			if err := svc.db.Where("vault_id = ?", vaultID).Find(&companies).Error; err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(companies) != 0 {
					t.Fatalf("empty organization created %d companies", len(companies))
				}
			} else if len(companies) != 1 || companies[0].Name != tc.want {
				t.Fatalf("did not reuse company %q (count %d)", tc.want, len(companies))
			}
		})
	}
}

func TestVCardImportedEmploymentNormalEditsDoNotResurrectLegacyFields(t *testing.T) {
	for _, remove := range []string{"job", "employee"} {
		t.Run(remove, func(t *testing.T) {
			svc, _, vaultID, userID := setupVCardTest(t)
			result, err := svc.ImportVCard(vaultID, userID, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nORG:Acme\r\nTITLE:Engineer\r\nEND:VCARD\r\n"))
			if err != nil {
				t.Fatal(err)
			}
			id := result.Contacts[0].ID
			jobs := NewContactJobService(svc.db)
			records, err := jobs.List(id, vaultID)
			if err != nil || len(records) != 1 {
				t.Fatalf("jobs %v %v", records, err)
			}
			if remove == "job" {
				_, err = jobs.Update(id, vaultID, records[0].ID, dto.UpdateContactJobRequest{CompanyID: records[0].CompanyID, JobPosition: "Architect"})
				if err != nil {
					t.Fatal(err)
				}
				data, err := svc.ExportContact(id, vaultID)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), "TITLE:Architect") || strings.Contains(string(data), "TITLE:Engineer") {
					t.Fatal("normal edit did not replace projected title")
				}
				err = jobs.Delete(id, vaultID, records[0].ID)
			} else {
				err = jobs.RemoveEmployee(records[0].CompanyID, vaultID, id)
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := svc.ExportContact(id, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "ORG:") || strings.Contains(string(data), "TITLE:") {
				t.Fatal("deleted employment resurfaced from legacy fields")
			}
		})
	}
}

func TestVCardMultipleJobsRoundTripPreservesEmploymentIdentities(t *testing.T) {
	svc, contactID, vaultID, userID := setupVCardTest(t)
	company := models.Company{VaultID: vaultID, Name: "Acme"}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	first, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Consultant"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := svc.ExportContact(contactID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	card, err := vcard.NewDecoder(strings.NewReader(string(data))).Decode()
	if err != nil {
		t.Fatal(err)
	}
	if len(card[vcard.FieldOrganization]) != 2 || len(card[vcard.FieldTitle]) != 2 {
		t.Fatal("lost an employment in export")
	}
	if card[vcard.FieldOrganization][0].Group != card[vcard.FieldTitle][0].Group || card[vcard.FieldOrganization][1].Group != card[vcard.FieldTitle][1].Group {
		t.Fatal("employment groups are not paired")
	}
	var user models.User
	if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	for _, revision := range []string{"roundtrip", "edited"} {
		if revision == "edited" {
			card[vcard.FieldTitle][1].Value = "Lead"
		}
		err := svc.db.Transaction(func(tx *gorm.DB) error {
			_, _, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, contactID, "/jobs.vcf", revision, nil)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		records, err := jobs.List(contactID, vaultID)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 2 || records[0].ID != first.ID || records[1].ID != second.ID || records[0].JobPosition != "Engineer" {
			t.Fatalf("employment identities changed: %+v", records)
		}
		want := "Consultant"
		if revision == "edited" {
			want = "Lead"
		}
		if records[1].JobPosition != want {
			t.Fatalf("second title = %q, want %q", records[1].JobPosition, want)
		}
	}
}
