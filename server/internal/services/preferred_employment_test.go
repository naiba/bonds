package services

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestPreferredEmploymentLegacyMigrationAndLifecycle(t *testing.T) {
	for _, removal := range []string{"job", "employee", "company", "legacy"} {
		t.Run(removal, func(t *testing.T) {
			svc, id, vault, user := setupVCardTest(t)
			jobs := NewContactJobService(svc.db)
			a := models.Company{VaultID: vault, Name: "Earlier employer"}
			b := models.Company{VaultID: vault, Name: "Selected employer"}
			for _, company := range []*models.Company{&a, &b} {
				if err := svc.db.Create(company).Error; err != nil {
					t.Fatal(err)
				}
			}
			earlier, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: a.ID, JobPosition: "Equal title"})
			if err != nil {
				t.Fatal(err)
			}
			selected, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: b.ID, JobPosition: "Equal title"})
			if err != nil {
				t.Fatal(err)
			}
			// Simulate an existing installation: there is no selection column yet.
			if err := svc.db.Migrator().DropColumn(&models.Contact{}, "PreferredJobID"); err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": b.ID, "job_position": "Equal title"}).Error; err != nil {
				t.Fatal(err)
			}
			if svc.db.Dialector.Name() == "postgres" {
				// An upgrade starts with fresh connections. This test first used
				// the new model to seed data, then removed its column: retire those
				// sessions so pgx cannot reuse a SELECT * plan for that old layout.
				pool, err := svc.db.DB()
				if err != nil {
					t.Fatal(err)
				}
				pool.SetMaxIdleConns(0)
				pool.SetMaxIdleConns(2)
			}
			if err := database.AutoMigrate(svc.db); err != nil {
				t.Fatal(err)
			}
			loaded, err := loadContactEmployment(svc.db, id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.PreferredJobID != nil || len(loaded.ContactCompanies) != 2 || ContactVCardEmployments(loaded)[0].JobID != selected.ID {
				t.Fatal("migration lost legacy employment or changed fallback")
			}
			// Equal titles cannot be used as identity. Freeze the old fallback before
			// the earlier secondary job moves into the selected company.
			if _, err := jobs.Update(id, vault, earlier.ID, dto.UpdateContactJobRequest{CompanyID: b.ID, JobPosition: "Equal title"}); err != nil {
				t.Fatal(err)
			}
			response, err := jobs.LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Selected edited"})
			if err != nil {
				t.Fatal(err)
			}
			if response.CompanyID == nil || *response.CompanyID != b.ID || response.JobPosition != "Selected edited" {
				t.Fatal("legacy response changed identity")
			}
			var first, second models.ContactCompany
			if err := svc.db.First(&first, earlier.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.First(&second, selected.ID).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(first.JobPosition) != "Equal title" || ptrToStr(second.JobPosition) != "Selected edited" {
				t.Fatal("legacy edit overwrote secondary job")
			}
			if err := database.AutoMigrate(svc.db); err != nil {
				t.Fatal(err)
			}
			loaded, err = loadContactEmployment(svc.db, id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.PreferredJobID == nil || *loaded.PreferredJobID != selected.ID {
				t.Fatal("selection not persisted across migration")
			}
			switch removal {
			case "job":
				err = jobs.Delete(id, vault, selected.ID)
			case "employee":
				err = jobs.RemoveEmployee(b.ID, vault, id)
			case "company":
				err = NewCompanyService(svc.db).Delete(b.ID, vault)
			case "legacy":
				_, err = jobs.LegacyDelete(id, vault, user)
			}
			if err != nil {
				t.Fatal(err)
			}
			loaded, err = loadContactEmployment(svc.db, id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.PreferredJobID != nil || loaded.CompanyID != nil || loaded.JobPosition != nil {
				t.Fatal("deleted employment left a stale selection")
			}
			wantJobs := 0
			if removal == "job" {
				wantJobs = 1
			}
			if len(loaded.ContactCompanies) != wantJobs {
				t.Fatal("unexpected remaining jobs")
			}
			if wantJobs == 1 && (loaded.ContactCompanies[0].ID != earlier.ID || ptrToStr(loaded.ContactCompanies[0].JobPosition) != "Equal title") {
				t.Fatal("unrelated job changed")
			}
		})
	}
}

func TestPreferredEmploymentMergePreservesChosenIdentity(t *testing.T) {
	for _, choice := range []string{"target", "source"} {
		t.Run(choice, func(t *testing.T) {
			svc, vault, user, _ := setupContactTest(t)
			target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Target"})
			if err != nil {
				t.Fatal(err)
			}
			source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Source"})
			if err != nil {
				t.Fatal(err)
			}
			company := models.Company{VaultID: vault, Name: "Shared employer"}
			if err := svc.db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			jobs := NewContactJobService(svc.db)
			sourceJob, err := jobs.Create(source.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Source role"})
			if err != nil {
				t.Fatal(err)
			}
			targetJob, err := jobs.Create(target.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Target role"})
			if err != nil {
				t.Fatal(err)
			}
			for _, job := range []*dto.ContactJobResponse{sourceJob, targetJob} {
				// NULL selection emulates pre-migration contacts while retaining the old profile.
				if err := svc.db.Model(&models.Contact{}).Where("id = ?", job.ContactID).Updates(map[string]any{"company_id": company.ID, "job_position": job.JobPosition, "preferred_job_id": nil}).Error; err != nil {
					t.Fatal(err)
				}
			}
			req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
			want := targetJob
			if choice == "source" {
				want = sourceJob
			}
			req.FieldChoices["job"] = want.ContactID
			if _, err := svc.MergeContacts(vault, user, req); err != nil {
				t.Fatal(err)
			}
			loaded, err := loadContactEmployment(svc.db, target.ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			projected := ContactVCardEmployments(loaded)
			if len(projected) != 2 || projected[0].JobID != want.ID || loaded.PreferredJobID == nil || *loaded.PreferredJobID != want.ID || ptrToStr(loaded.JobPosition) != want.JobPosition {
				t.Fatalf("merge lost selection: %+v", projected)
			}
			if _, err := jobs.LegacyUpdate(target.ID, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Edited chosen role"}); err != nil {
				t.Fatal(err)
			}
			var actual []models.ContactCompany
			if err := svc.db.Where("contact_id = ?", target.ID).Find(&actual).Error; err != nil {
				t.Fatal(err)
			}
			for _, job := range actual {
				expected := sourceJob.JobPosition
				if job.ID == targetJob.ID {
					expected = targetJob.JobPosition
				}
				if job.ID == want.ID {
					expected = "Edited chosen role"
				}
				if ptrToStr(job.JobPosition) != expected {
					t.Fatal("merge retargeted later edit")
				}
			}
		})
	}
}

func TestVCardSubscriptionAmbiguousCompanyRollsBack(t *testing.T) {
	svc, id, vault, user := setupVCardTest(t)
	for range 2 {
		if err := svc.db.Create(&models.Company{VaultID: vault, Name: "Shared name"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	var owner models.User
	if err := svc.db.First(&owner, "id = ?", user).Error; err != nil {
		t.Fatal(err)
	}
	var before models.Contact
	if err := svc.db.First(&before, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Must roll back\r\nORG:Shared name\r\nTITLE:Engineer\r\nEND:VCARD\r\n")).Decode()
	if err != nil {
		t.Fatal(err)
	}
	for _, existingID := range []string{id, ""} {
		err = svc.db.Transaction(func(tx *gorm.DB) error {
			_, _, err := svc.UpsertContactFromVCard(tx, card, vault, user, owner.AccountID, existingID, "/ambiguous.vcf", "new-etag", nil)
			return err
		})
		if !errors.Is(err, ErrVCardUnsafeReplacement) {
			t.Fatalf("want ambiguity error, got %v", err)
		}
		var after models.Contact
		if err := svc.db.First(&after, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatal("rejected subscription changed contact/remote markers")
		}
		for model, want := range map[any]int64{&models.Contact{}: 1, &models.Company{}: 2, &models.ContactCompany{}: 0} {
			var count int64
			if err := svc.db.Model(model).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != want {
				t.Fatalf("rollback count=%d want=%d", count, want)
			}
		}
	}
}
