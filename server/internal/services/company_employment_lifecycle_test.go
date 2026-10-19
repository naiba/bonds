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
