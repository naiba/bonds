package services

import (
	"errors"
	"reflect"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestEmploymentLegacyJobsLifecycle(t *testing.T) {
	for _, entry := range []string{"create job", "add employee", "existing mixed data"} {
		for _, mirror := range []bool{false, true} {
			t.Run(entry+map[bool]string{false: "/independent", true: "/mirror"}[mirror], func(t *testing.T) {
				svc, id, vault, user := setupVCardTest(t)
				original := models.Company{VaultID: vault, Name: "Original employer"}
				additional := models.Company{VaultID: vault, Name: "Additional employer"}
				for _, c := range []*models.Company{&original, &additional} {
					if err := svc.db.Create(c).Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": original.ID, "job_position": "Original title"}).Error; err != nil {
					t.Fatal(err)
				}
				if mirror {
					if err := svc.db.Create(&models.ContactCompany{ContactID: id, CompanyID: original.ID, JobPosition: strPtrOrNil("Original title")}).Error; err != nil {
						t.Fatal(err)
					}
				}
				jobs := NewContactJobService(svc.db)
				switch entry {
				case "create job":
					if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: additional.ID}); err != nil {
						t.Fatal(err)
					}
				case "add employee":
					if _, err := jobs.AddEmployee(additional.ID, vault, dto.AddEmployeeRequest{ContactID: id}); err != nil {
						t.Fatal(err)
					}
				case "existing mixed data":
					if err := svc.db.Create(&models.ContactCompany{ContactID: id, CompanyID: additional.ID}).Error; err != nil {
						t.Fatal(err)
					}
				}
				beforeRead, err := jobs.List(id, vault)
				if err != nil {
					t.Fatal(err)
				}
				card, err := svc.ExportContactToVCard(id, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(card[vcard.FieldOrganization]) != 2 || card.Value(vcard.FieldOrganization) != "Original employer" || card.Value(vcard.FieldTitle) != "Original title" {
					t.Errorf("adding employment hid old identity: orgs=%d first=%q title=%q", len(card[vcard.FieldOrganization]), card.Value(vcard.FieldOrganization), card.Value(vcard.FieldTitle))
				}
				afterRead, err := jobs.List(id, vault)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(beforeRead, afterRead) {
					t.Fatal("read unexpectedly materialized employment")
				}
				// A later title-only edit must target the original employer, including for
				// mixed records saved before materialization covered local Jobs entry points.
				if _, err := jobs.LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Edited original title"}); err != nil {
					t.Fatal(err)
				}
				rows, err := jobs.List(id, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != 2 {
					t.Fatalf("missing old relationship or duplicate mirror: %+v", rows)
				}
				var originalID, additionalID uint
				for _, j := range rows {
					switch j.CompanyID {
					case original.ID:
						originalID = j.ID
						if j.JobPosition != "Edited original title" {
							t.Error("original job not edited")
						}
					case additional.ID:
						additionalID = j.ID
						if j.JobPosition != "" {
							t.Error("unrelated job overwritten")
						}
					default:
						t.Fatal("unexpected company")
					}
				}
				if originalID == 0 || additionalID == 0 {
					t.Fatal("employment identities not retained")
				}
				var stored models.Contact
				if err := svc.db.First(&stored, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if stored.PreferredJobID == nil || *stored.PreferredJobID != originalID || stored.CompanyID == nil || *stored.CompanyID != original.ID {
					t.Fatal("selection switched to new job")
				}
				if entry == "add employee" {
					err = jobs.RemoveEmployee(additional.ID, vault, id)
				} else {
					err = jobs.Delete(id, vault, additionalID)
				}
				if err != nil {
					t.Fatal(err)
				}
				card, err = svc.ExportContactToVCard(id, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(card[vcard.FieldOrganization]) != 1 || card.Value(vcard.FieldOrganization) != "Original employer" || card.Value(vcard.FieldTitle) != "Edited original title" {
					t.Fatal("deleting unrelated job lost original employment")
				}
				if err := jobs.Delete(id, vault, originalID); err != nil {
					t.Fatal(err)
				}
				card, err = svc.ExportContactToVCard(id, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(card[vcard.FieldOrganization]) != 0 || len(card[vcard.FieldTitle]) != 0 {
					t.Fatal("deleting selected job resurrected compatibility fields")
				}
			})
		}
	}
}

func TestEmploymentRejectedJobCreationRollsBackLegacyMaterialization(t *testing.T) {
	for _, entry := range []string{"create job", "add employee"} {
		t.Run(entry, func(t *testing.T) {
			svc, id, vault, _ := setupVCardTest(t)
			original := models.Company{VaultID: vault, Name: "Original employer"}
			if err := svc.db.Create(&original).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": original.ID, "job_position": "Original title"}).Error; err != nil {
				t.Fatal(err)
			}
			var before, after models.Contact
			if err := svc.db.First(&before, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			jobs := NewContactJobService(svc.db)
			missing := original.ID + 1
			var err error
			if entry == "create job" {
				_, err = jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: missing})
			} else {
				_, err = jobs.AddEmployee(missing, vault, dto.AddEmployeeRequest{ContactID: id})
			}
			if !errors.Is(err, ErrCompanyNotFound) {
				t.Fatalf("want missing company error, got %v", err)
			}
			if err := svc.db.First(&after, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			rows, err := jobs.List(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) || len(rows) != 0 {
				t.Fatal("rejected operation retained materialization or changed original contact")
			}
		})
	}
}
