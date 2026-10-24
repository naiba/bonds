package services

import (
	"errors"
	"reflect"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestMergeSelectedLegacyEmploymentPreservesIdentity(t *testing.T) {
	for _, legacySide := range []string{"target", "source"} {
		for _, scenario := range []string{"legacy selection", "normal selection", "same company", "rollback"} {
			t.Run(legacySide+"/"+scenario, func(t *testing.T) {
				svc, vault, user, _ := setupContactTest(t)
				target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Synthetic target"})
				if err != nil {
					t.Fatal(err)
				}
				source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Synthetic source"})
				if err != nil {
					t.Fatal(err)
				}
				legacyID, normalID := target.ID, source.ID
				if legacySide == "source" {
					legacyID, normalID = source.ID, target.ID
				}
				legacyCompany := models.Company{VaultID: vault, Name: "Legacy employer"}
				if err := svc.db.Create(&legacyCompany).Error; err != nil {
					t.Fatal(err)
				}
				normalCompany := legacyCompany
				if scenario != "same company" {
					normalCompany = models.Company{VaultID: vault, Name: "Normal employer"}
					if err := svc.db.Create(&normalCompany).Error; err != nil {
						t.Fatal(err)
					}
				}
				// Valid pre-upgrade data: no Jobs row or preferred ID for this employer.
				if err := svc.db.Model(&models.Contact{}).Where("id = ?", legacyID).Updates(map[string]any{"company_id": legacyCompany.ID, "job_position": "Engineer"}).Error; err != nil {
					t.Fatal(err)
				}
				jobs := NewContactJobService(svc.db)
				if _, err := jobs.LegacyUpdate(normalID, vault, user, dto.UpdateJobInfoRequest{CompanyID: &normalCompany.ID, JobPosition: "Consultant"}); err != nil {
					t.Fatal(err)
				}
				beforeJobs, err := jobs.List(normalID, vault)
				if err != nil || len(beforeJobs) != 1 {
					t.Fatalf("normal fixture: %v %+v", err, beforeJobs)
				}
				normalJob := beforeJobs[0]
				req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
				req.FieldChoices["job"] = legacyID
				wantCompany, wantTitle := legacyCompany, "Engineer"
				if scenario == "normal selection" {
					req.FieldChoices["job"] = normalID
					wantCompany, wantTitle = normalCompany, "Consultant"
				}

				if scenario == "rollback" {
					var beforeContacts, afterContacts []models.Contact
					var beforeRows, afterRows []models.ContactCompany
					var beforeCompanies, afterCompanies []models.Company
					read := func(contacts *[]models.Contact, rows *[]models.ContactCompany, companies *[]models.Company) {
						t.Helper()
						for _, dest := range []any{contacts, rows, companies} {
							if err := svc.db.Unscoped().Order("id ASC").Find(dest).Error; err != nil {
								t.Fatal(err)
							}
						}
					}
					read(&beforeContacts, &beforeRows, &beforeCompanies)
					injected := errors.New("synthetic merged source deletion failure")
					const hook = "employment:reject_merged_source_deletion"
					if err := svc.db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
						if tx.Statement.Table == "contacts" {
							tx.AddError(injected)
						}
					}); err != nil {
						t.Fatal(err)
					}
					defer svc.db.Callback().Delete().Remove(hook)
					if _, err := svc.MergeContacts(vault, user, req); !errors.Is(err, injected) {
						t.Fatalf("expected injected final deletion error, got %v", err)
					}
					read(&afterContacts, &afterRows, &afterCompanies)
					if !reflect.DeepEqual(beforeContacts, afterContacts) || !reflect.DeepEqual(beforeRows, afterRows) || !reflect.DeepEqual(beforeCompanies, afterCompanies) {
						t.Fatal("failed merge retained materialized employment or changed original records")
					}
					return
				}

				response, err := svc.MergeContacts(vault, user, req)
				if err != nil {
					t.Fatal(err)
				}
				if response.CompanyID == nil || *response.CompanyID != wantCompany.ID || response.JobPosition != wantTitle {
					t.Fatal("merge response lost reviewed employment choice")
				}
				loaded, err := loadContactEmployment(svc.db, target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				projection := ContactVCardEmployments(loaded)
				if len(projection) != 2 || projection[0].CompanyID != wantCompany.ID || projection[0].Title != wantTitle || loaded.PreferredJobID == nil || *loaded.PreferredJobID != projection[0].JobID {
					t.Fatalf("merge lost chosen identity: %+v", projection)
				}
				selectedID := projection[0].JobID
				if (selectedID == normalJob.ID) != (scenario == "normal selection") {
					t.Fatal("selected the wrong persistent job")
				}
				actual, err := jobs.List(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(actual) != 2 {
					t.Fatalf("unexpected jobs: %+v", actual)
				}
				for _, job := range actual {
					if job.ID == normalJob.ID {
						if job.CompanyID != normalJob.CompanyID || job.JobPosition != normalJob.JobPosition {
							t.Fatal("merge changed normal employment")
						}
					} else if job.CompanyID != legacyCompany.ID || job.JobPosition != "Engineer" {
						t.Fatal("legacy employment was not retained")
					}
				}
				// Load and export through the public service, not just an in-memory model.
				card, err := NewVCardService(svc.db).ExportContactToVCard(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(card[vcard.FieldOrganization]) != 2 || card.Get(vcard.FieldOrganization).Organization()[0] != wantCompany.Name || card.Value(vcard.FieldTitle) != wantTitle {
					t.Fatal("persisted export disagrees with reviewed selection")
				}
				var sourceAfter models.Contact
				if err := svc.db.Unscoped().First(&sourceAfter, "id = ?", source.ID).Error; err != nil {
					t.Fatal(err)
				}
				if !sourceAfter.DeletedAt.Valid {
					t.Fatal("source was not merged")
				}
				if _, err := jobs.LegacyUpdate(target.ID, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Edited selected role"}); err != nil {
					t.Fatal(err)
				}
				after, err := jobs.List(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(after) != len(actual) {
					t.Fatal("title-only edit created or removed jobs")
				}
				for _, before := range actual {
					found := false
					for _, job := range after {
						if job.ID == before.ID {
							found = true
							want := before.JobPosition
							if before.ID == selectedID {
								want = "Edited selected role"
							}
							if job.CompanyID != before.CompanyID || job.JobPosition != want {
								t.Fatal("title-only edit changed unrelated job or employer")
							}
						}
					}
					if !found {
						t.Fatal("title-only edit lost job identity")
					}
				}
				loaded, err = loadContactEmployment(svc.db, target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if loaded.CompanyID == nil || *loaded.CompanyID != wantCompany.ID || ptrToStr(loaded.JobPosition) != "Edited selected role" {
					t.Fatal("title-only edit lost selected company")
				}
			})
		}
	}
}
