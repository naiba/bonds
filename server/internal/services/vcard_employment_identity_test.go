package services

import (
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardSubscriptionRoundTripPreservesEmploymentIdentity(t *testing.T) {
	for _, scenario := range []string{"company whitespace", "duplicate company names", "title whitespace"} {
		t.Run(scenario, func(t *testing.T) {
			svc, contactID, vaultID, userID := setupVCardTest(t)
			name, title := "Acme", "Engineer"
			if scenario == "company whitespace" {
				name = " \tAcme\u3000"
			}
			if scenario == "title whitespace" {
				title = " \tEngineer\u3000"
			}
			if scenario == "duplicate company names" {
				if err := svc.db.Create(&models.Company{VaultID: vaultID, Name: name}).Error; err != nil {
					t.Fatal(err)
				}
			}
			company := models.Company{VaultID: vaultID, Name: name}
			if err := svc.db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			jobs := NewContactJobService(svc.db)
			first, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: title})
			if err != nil {
				t.Fatal(err)
			}
			expected := map[uint]string{first.ID: title}
			if scenario == "title whitespace" {
				second, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Consultant"})
				if err != nil {
					t.Fatal(err)
				}
				expected[second.ID] = "Consultant"
			}
			var beforeCompanies int64
			if err := svc.db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&beforeCompanies).Error; err != nil {
				t.Fatal(err)
			}
			var user models.User
			if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
				t.Fatal(err)
			}
			for _, etag := range []string{"first roundtrip", "second roundtrip"} {
				wire, err := svc.ExportContact(contactID, vaultID)
				if err != nil {
					t.Fatal(err)
				}
				card, err := vcard.NewDecoder(strings.NewReader(string(wire))).Decode()
				if err != nil {
					t.Fatal(err)
				}
				err = svc.db.Transaction(func(tx *gorm.DB) error {
					id, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, contactID, "/identity.vcf", etag, nil)
					if err == nil && (id != contactID || action != "updated") {
						t.Fatalf("wrong upsert result: %s %s", id, action)
					}
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				actual, err := jobs.List(contactID, vaultID)
				if err != nil {
					t.Fatal(err)
				}
				if len(actual) != len(expected) {
					t.Fatalf("roundtrip created jobs: got %d want %d", len(actual), len(expected))
				}
				for _, job := range actual {
					want, exists := expected[job.ID]
					if !exists || job.CompanyID != company.ID || job.JobPosition != want {
						t.Fatalf("employment identity or stored title changed: %+v", job)
					}
				}
				var afterCompanies int64
				if err := svc.db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&afterCompanies).Error; err != nil {
					t.Fatal(err)
				}
				if afterCompanies != beforeCompanies {
					t.Fatal("roundtrip created a company")
				}
				var stored models.Company
				if err := svc.db.First(&stored, company.ID).Error; err != nil {
					t.Fatal(err)
				}
				if stored.Name != name {
					t.Fatal("roundtrip renamed the existing company")
				}
			}
		})
	}
}

func TestContactJobMutationsPreserveSelectedEmployment(t *testing.T) {
	for _, mutation := range []string{"secondary title", "secondary company", "secondary deletion", "primary company"} {
		t.Run(mutation, func(t *testing.T) {
			svc, _, vaultID, userID := setupVCardTest(t)
			imported, err := svc.ImportVCard(vaultID, userID, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nORG:Acme\r\nTITLE:Engineer\r\nEND:VCARD\r\n"))
			if err != nil {
				t.Fatal(err)
			}
			if len(imported.Contacts) != 1 {
				t.Fatal("missing imported contact")
			}
			id := imported.Contacts[0].ID
			jobs := NewContactJobService(svc.db)
			existing, err := jobs.List(id, vaultID)
			if err != nil || len(existing) != 1 {
				t.Fatalf("initial jobs: %v %v", existing, err)
			}
			primary := existing[0]
			secondary, err := jobs.Create(id, vaultID, dto.CreateContactJobRequest{CompanyID: primary.CompanyID, JobPosition: "Consultant"})
			if err != nil {
				t.Fatal(err)
			}
			beta := models.Company{VaultID: vaultID, Name: "Beta"}
			if err := svc.db.Create(&beta).Error; err != nil {
				t.Fatal(err)
			}
			wantCompany, wantName, wantTitle := primary.CompanyID, "Acme", "Engineer"
			switch mutation {
			case "secondary title":
				_, err = jobs.Update(id, vaultID, secondary.ID, dto.UpdateContactJobRequest{CompanyID: primary.CompanyID, JobPosition: "Lead consultant"})
			case "secondary company":
				_, err = jobs.Update(id, vaultID, secondary.ID, dto.UpdateContactJobRequest{CompanyID: beta.ID, JobPosition: "Consultant"})
			case "secondary deletion":
				err = jobs.Delete(id, vaultID, secondary.ID)
			case "primary company":
				_, err = jobs.Update(id, vaultID, primary.ID, dto.UpdateContactJobRequest{CompanyID: beta.ID, JobPosition: "Architect"})
				wantCompany, wantName, wantTitle = beta.ID, "Beta", "Architect"
			}
			if err != nil {
				t.Fatal(err)
			}
			var stored models.Contact
			if err := preloadContactVCardRelations(svc.db).First(&stored, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if stored.CompanyID == nil || *stored.CompanyID != wantCompany || ptrToStr(stored.JobPosition) != wantTitle {
				t.Fatalf("wrong legacy selection: company %v title %q", stored.CompanyID, ptrToStr(stored.JobPosition))
			}
			projected := ContactVCardEmployments(&stored)
			if len(projected) == 0 || projected[0].JobID != primary.ID || projected[0].Company != wantName || projected[0].Title != wantTitle {
				t.Fatalf("primary employment changed: %+v", projected)
			}
			card := BuildContactVCard(&stored)
			if card.Get(vcard.FieldOrganization).Organization()[0] != wantName || card.Value(vcard.FieldTitle) != wantTitle {
				t.Fatal("first exported employment changed")
			}
		})
	}
}

func TestVCardAdditionalRoleReusesProjectedCompany(t *testing.T) {
	svc, contactID, vaultID, userID := setupVCardTest(t)
	company := models.Company{VaultID: vaultID, Name: " Acme "}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	original, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	wire, err := svc.ExportContact(contactID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	card, err := vcard.NewDecoder(strings.NewReader(string(wire))).Decode()
	if err != nil {
		t.Fatal(err)
	}
	organization := vcard.NewOrganization([]string{"Acme"})
	organization.Group = "additional-role"
	card.Add(vcard.FieldOrganization, organization)
	card.Add(vcard.FieldTitle, &vcard.Field{Value: "Consultant", Group: "additional-role"})
	var user models.User
	if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	err = svc.db.Transaction(func(tx *gorm.DB) error {
		_, _, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, contactID, "/additional-role.vcf", "updated", nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	actual, err := jobs.List(contactID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != 2 {
		t.Fatalf("expected exactly two roles, got %d", len(actual))
	}
	preserved, added := false, false
	for _, job := range actual {
		if job.CompanyID != company.ID {
			t.Fatal("additional role created another company")
		}
		if job.ID == original.ID && job.JobPosition == "Engineer" {
			preserved = true
		}
		if job.ID != original.ID && job.JobPosition == "Consultant" {
			added = true
		}
	}
	if !preserved || !added {
		t.Fatalf("wrong employment identities: %+v", actual)
	}
	var count int64
	if err := svc.db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("created %d companies, want 1", count)
	}
}
