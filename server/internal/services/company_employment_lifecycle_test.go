package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestCompanyEmploymentDeletionAtomicity(t *testing.T) {
	for _, failDelete := range []bool{false, true} {
		name := "commit"
		if failDelete {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			svc, contactID, vaultID, _ := setupVCardTest(t)
			old := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
			employer := models.Company{VaultID: vaultID, Name: "Acme"}
			retained := models.Company{VaultID: vaultID, Name: "Beta"}
			for _, company := range []*models.Company{&employer, &retained} {
				if err := svc.db.Create(company).Error; err != nil {
					t.Fatal(err)
				}
			}
			jobs := NewContactJobService(svc.db)
			removed, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: employer.ID, JobPosition: "Engineer"})
			if err != nil {
				t.Fatal(err)
			}
			kept, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: retained.ID, JobPosition: "Consultant"})
			if err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Updates(map[string]any{"company_id": retained.ID, "job_position": "Consultant", "last_updated_at": old}).Error; err != nil {
				t.Fatal(err)
			}
			legacy := models.Contact{VaultID: vaultID, CompanyID: &employer.ID, JobPosition: strPtrOrNil("Engineer"), LastUpdatedAt: &old}
			unrelated := models.Contact{VaultID: vaultID, JobPosition: strPtrOrNil("Artist"), LastUpdatedAt: &old}
			for _, contact := range []*models.Contact{&legacy, &unrelated} {
				if err := svc.db.Create(contact).Error; err != nil {
					t.Fatal(err)
				}
			}
			injected := errors.New("synthetic company delete failure")
			if failDelete {
				const hook = "employment:reject_company_delete"
				if err := svc.db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
					if tx.Statement.Table == "companies" {
						tx.AddError(injected)
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer svc.db.Callback().Delete().Remove(hook)
			}
			err = NewCompanyService(svc.db).Delete(employer.ID, vaultID)
			if failDelete {
				if !errors.Is(err, injected) {
					t.Fatalf("expected injected failure: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			actual, err := jobs.List(contactID, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			expected := 1
			if failDelete {
				expected = 2
			}
			if len(actual) != expected {
				t.Fatalf("jobs=%d want %d", len(actual), expected)
			}
			foundKept, foundRemoved := false, false
			for _, job := range actual {
				if job.ID == kept.ID && job.CompanyID == retained.ID && job.JobPosition == "Consultant" {
					foundKept = true
				}
				if job.ID == removed.ID && job.CompanyID == employer.ID && job.JobPosition == "Engineer" {
					foundRemoved = true
				}
			}
			if !foundKept || foundRemoved != failDelete {
				t.Fatal("employment identity or title changed")
			}
			var current, legacyAfter, unrelatedAfter models.Contact
			for id, dest := range map[string]*models.Contact{contactID: &current, legacy.ID: &legacyAfter, unrelated.ID: &unrelatedAfter} {
				if err := svc.db.First(dest, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
			}
			if current.CompanyID == nil || *current.CompanyID != retained.ID || ptrToStr(current.JobPosition) != "Consultant" {
				t.Fatal("unrelated selected employment changed")
			}
			if unrelatedAfter.LastUpdatedAt == nil || !unrelatedAfter.LastUpdatedAt.Equal(old) || ptrToStr(unrelatedAfter.JobPosition) != "Artist" {
				t.Fatal("unrelated contact was modified")
			}
			if failDelete {
				if current.LastUpdatedAt == nil || !current.LastUpdatedAt.Equal(old) || legacyAfter.LastUpdatedAt == nil || !legacyAfter.LastUpdatedAt.Equal(old) || legacyAfter.CompanyID == nil || *legacyAfter.CompanyID != employer.ID || ptrToStr(legacyAfter.JobPosition) != "Engineer" {
					t.Fatal("failed deletion partially changed contacts")
				}
			} else {
				if current.LastUpdatedAt == nil || !current.LastUpdatedAt.After(old) || legacyAfter.LastUpdatedAt == nil || !legacyAfter.LastUpdatedAt.After(old) || legacyAfter.CompanyID != nil || ptrToStr(legacyAfter.JobPosition) != "" {
					t.Fatal("deletion missed marker or legacy cleanup")
				}
			}
			var companyCount int64
			if err := svc.db.Model(&models.Company{}).Where("id = ?", employer.ID).Count(&companyCount).Error; err != nil {
				t.Fatal(err)
			}
			if (companyCount == 1) != failDelete {
				t.Fatal("company deletion/rollback mismatch")
			}
		})
	}
}

func TestCompanyRenameEmploymentAtomicity(t *testing.T) {
	for _, scenario := range []string{"rename", "type only", "rollback"} {
		t.Run(scenario, func(t *testing.T) {
			svc, id, vault, _ := setupVCardTest(t)
			company := models.Company{VaultID: vault, Name: "Acme"}
			if err := svc.db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			job, err := NewContactJobService(svc.db).Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
			if err != nil {
				t.Fatal(err)
			}
			old := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Update("last_updated_at", old).Error; err != nil {
				t.Fatal(err)
			}
			legacy := models.Contact{VaultID: vault, CompanyID: &company.ID, JobPosition: strPtrOrNil("Consultant"), LastUpdatedAt: &old}
			unrelated := models.Contact{VaultID: vault, LastUpdatedAt: &old}
			for _, c := range []*models.Contact{&legacy, &unrelated} {
				if err := svc.db.Create(c).Error; err != nil {
					t.Fatal(err)
				}
			}
			injected := errors.New("synthetic marker failure")
			if scenario == "rollback" {
				const hook = "employment:reject_rename_marker"
				if err := svc.db.Callback().Update().Before("gorm:update").Register(hook, func(tx *gorm.DB) {
					if tx.Statement.Table == "contacts" {
						tx.AddError(injected)
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer svc.db.Callback().Update().Remove(hook)
			}
			name := "Beta"
			if scenario == "type only" {
				name = "Acme"
			}
			_, err = NewCompanyService(svc.db).Update(company.ID, vault, dto.UpdateCompanyRequest{Name: name, Type: "Business"})
			if scenario == "rollback" {
				if !errors.Is(err, injected) {
					t.Fatalf("expected marker failure: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var actual models.Company
			if err := svc.db.First(&actual, company.ID).Error; err != nil {
				t.Fatal(err)
			}
			wantName, wantType := name, "Business"
			if scenario == "rollback" {
				wantName = "Acme"
				wantType = ""
			}
			if actual.Name != wantName || ptrToStr(actual.Type) != wantType {
				t.Fatal("company update did not commit/rollback atomically")
			}
			for _, contactID := range []string{id, legacy.ID, unrelated.ID} {
				var current models.Contact
				if err := svc.db.First(&current, "id = ?", contactID).Error; err != nil {
					t.Fatal(err)
				}
				changed := scenario == "rename" && contactID != unrelated.ID
				if current.LastUpdatedAt == nil || (changed && !current.LastUpdatedAt.After(old)) || (!changed && !current.LastUpdatedAt.Equal(old)) {
					t.Fatalf("wrong marker on %s in %s", contactID, scenario)
				}
			}
			jobs, err := NewContactJobService(svc.db).List(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if len(jobs) != 1 || jobs[0].ID != job.ID || jobs[0].CompanyID != company.ID || jobs[0].JobPosition != "Engineer" {
				t.Fatal("rename changed employment identity or title")
			}
		})
	}
}
