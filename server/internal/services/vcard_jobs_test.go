package services

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardJobsExportLifecycle(t *testing.T) {
	svc, contactID, vaultID, userID := setupVCardTest(t)
	jobs := NewContactJobService(svc.db)
	companies := NewCompanyService(svc.db)
	createCompany := func(name string) uint {
		t.Helper()
		company, err := companies.Create(vaultID, dto.CreateCompanyRequest{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return company.ID
	}
	olderCompany := createCompany("Earlier company")
	newerCompany := createCompany("最新 Company, Inc.")
	older, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: olderCompany, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	newer, err := jobs.AddEmployee(newerCompany, vaultID, dto.AddEmployeeRequest{ContactID: contactID, JobPosition: "Director"})
	if err != nil {
		t.Fatal(err)
	}
	// Equal creation timestamps still have a stable newest entry, independently
	// of database row order. Editing an older job must not promote it.
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ?", contactID).Update("created_at", created).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Update(contactID, vaultID, older.ID, dto.UpdateContactJobRequest{CompanyID: olderCompany, JobPosition: "Senior engineer"}); err != nil {
		t.Fatal(err)
	}
	second, err := NewContactService(svc.db).CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Create(second.ID, vaultID, dto.CreateContactJobRequest{CompanyID: olderCompany, JobPosition: "Analyst"}); err != nil {
		t.Fatal(err)
	}

	assertExports := func(org, title string) {
		t.Helper()
		var before, after []models.ContactCompany
		if err := svc.db.Order("id").Find(&before).Error; err != nil {
			t.Fatal(err)
		}
		data, err := svc.ExportContact(contactID, vaultID)
		if err != nil {
			t.Fatal(err)
		}
		assertVCardJob(t, mustDecodeVCard(t, data), org, title)
		card, err := svc.ExportContactToVCard(contactID, vaultID)
		if err != nil {
			t.Fatal(err)
		}
		assertVCardJob(t, card, org, title)
		data, err = svc.ExportVault(vaultID)
		if err != nil {
			t.Fatal(err)
		}
		decoder := vcard.NewDecoder(bytes.NewReader(data))
		seen := map[string]bool{}
		for {
			card, err := decoder.Decode()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			id := card.Value(vcard.FieldUID)
			if seen[id] {
				t.Fatalf("duplicate contact %s", id)
			}
			seen[id] = true
			switch id {
			case contactID:
				assertVCardJob(t, card, org, title)
			case second.ID:
				assertVCardJob(t, card, "Earlier company", "Analyst")
			default:
				t.Fatalf("unexpected contact %s", id)
			}
		}
		if len(seen) != 2 {
			t.Fatalf("expected both contacts, got %v", seen)
		}
		if err := svc.db.Order("id").Find(&after).Error; err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatal("export modified job records")
		}
		loaded, err := svc.loadContactForVCard(contactID, vaultID)
		if err != nil {
			t.Fatal(err)
		}
		wantJobs := 0
		for _, job := range before {
			if job.ContactID == contactID {
				wantJobs = 1
			}
		}
		if len(loaded.ContactCompanies) != wantJobs {
			t.Fatalf("projection loaded %d jobs, want %d", len(loaded.ContactCompanies), wantJobs)
		}
	}
	assertExports("最新 Company, Inc.", "Director")
	if _, err := companies.Update(newerCompany, vaultID, dto.UpdateCompanyRequest{Name: "Renamed company"}); err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Update(contactID, vaultID, newer.ID, dto.UpdateContactJobRequest{CompanyID: newerCompany}); err != nil {
		t.Fatal(err)
	}
	assertExports("Renamed company", "")
	// CreatedAt takes precedence over ID (including imported/backfilled rows).
	if err := svc.db.Model(&models.ContactCompany{}).Where("id = ?", older.ID).Update("created_at", created.Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	assertExports("Earlier company", "Senior engineer")
	if err := jobs.Delete(contactID, vaultID, older.ID); err != nil {
		t.Fatal(err)
	}
	assertExports("Renamed company", "")
	if err := companies.Delete(newerCompany, vaultID); err != nil {
		t.Fatal(err)
	}
	assertExports("", "")
}

func TestVCardJobsLegacyAndVaultIsolation(t *testing.T) {
	svc, contactID, vaultID, userID := setupVCardTest(t)
	company := models.Company{VaultID: vaultID, Name: "Legacy company"}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Updates(map[string]any{"company_id": company.ID, "job_position": "Legacy title"}).Error; err != nil {
		t.Fatal(err)
	}
	check := func(org, title string) {
		t.Helper()
		data, err := svc.ExportContact(contactID, vaultID)
		if err != nil {
			t.Fatal(err)
		}
		assertVCardJob(t, mustDecodeVCard(t, data), org, title)
	}
	check("Legacy company", "Legacy title")
	job, err := NewContactJobService(svc.db).Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID})
	if err != nil {
		t.Fatal(err)
	}
	check("Legacy company", "") // Never borrow a title from a different source.
	if err := svc.db.Model(&models.ContactCompany{}).Where("id = ?", job.ID).Update("job_position", "").Error; err != nil {
		t.Fatal(err)
	}
	check("Legacy company", "")
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatal(err)
	}
	other, err := NewVaultService(svc.db).CreateVault(vault.AccountID, userID, dto.CreateVaultRequest{Name: "Private vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	privateCompany := models.Company{VaultID: other.ID, Name: "Private company"}
	if err := svc.db.Create(&privateCompany).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactJobService(svc.db).Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: privateCompany.ID}); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("cross-vault create: %v", err)
	}
	var count int64
	if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ?", contactID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rejected create changed jobs: %d", count)
	}
	if _, err := svc.ExportContact(contactID, other.ID); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("cross-vault export: %v", err)
	}
	// Historical invalid references must not expose a company from another vault.
	if err := svc.db.Model(&models.ContactCompany{}).Where("id = ?", job.ID).Updates(map[string]any{"company_id": privateCompany.ID, "job_position": "Private position"}).Error; err != nil {
		t.Fatal(err)
	}
	check("", "")
	if err := svc.db.Model(&models.ContactCompany{}).Where("id = ?", job.ID).Update("company_id", privateCompany.ID+1000).Error; err != nil {
		t.Fatal(err)
	}
	check("", "")
	if err := svc.db.Delete(&models.ContactCompany{}, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("company_id", privateCompany.ID).Error; err != nil {
		t.Fatal(err)
	}
	check("", "Legacy title")
}

func assertVCardJob(t *testing.T, card vcard.Card, org, title string) {
	t.Helper()
	for field, want := range map[string]string{vcard.FieldOrganization: org, vcard.FieldTitle: title} {
		if got := card.Value(field); got != want {
			t.Fatalf("%s = %q, want %q", field, got, want)
		}
		count := 0
		if want != "" {
			count = 1
		}
		if len(card[field]) != count {
			t.Fatalf("%s has %d fields, want %d", field, len(card[field]), count)
		}
	}
}

func TestVCardJobsPullPreservesLegacyTitle(t *testing.T) {
	svc, contactID, vaultID, _ := setupVCardTest(t)
	company := models.Company{VaultID: vaultID, Name: "Local company"}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	job, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Current title"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("job_position", "Legacy title").Error; err != nil {
		t.Fatal(err)
	}
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatal(err)
	}
	card, err := svc.ExportContactToVCard(contactID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	card.SetName(&vcard.Name{GivenName: "Remote name"})
	pull := func(etag string) error {
		return svc.db.Transaction(func(tx *gorm.DB) error {
			id, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, "", vault.AccountID, contactID, "/contact.vcf", etag, nil)
			if err == nil && (id != contactID || action != "updated") {
				t.Fatalf("unexpected pull result %q / %q", id, action)
			}
			return err
		})
	}
	if err := pull("remote-etag"); err != nil {
		t.Fatal(err)
	}
	var stored models.Contact
	if err := svc.db.First(&stored, "id = ?", contactID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(stored.JobPosition) != "Legacy title" || ptrToStr(stored.FirstName) != "Remote name" {
		t.Fatalf("pull failed to preserve legacy title or apply name: %q / %q", ptrToStr(stored.JobPosition), ptrToStr(stored.FirstName))
	}
	var storedJob models.ContactCompany
	if err := svc.db.First(&storedJob, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedJob.CompanyID != company.ID || ptrToStr(storedJob.JobPosition) != "Current title" {
		t.Fatal("pull changed local job")
	}
	if err := jobs.Delete(contactID, vaultID, job.ID); err != nil {
		t.Fatal(err)
	}
	card.SetValue(vcard.FieldTitle, "New legacy title")
	card.SetValue(vcard.FieldBirthday, "invalid")
	card.SetName(&vcard.Name{GivenName: "Rejected name"})
	if err := pull("invalid-birthday"); !errors.Is(err, ErrVCardInvalidData) {
		t.Fatalf("expected invalid birthday, got %v", err)
	}
	if err := svc.db.First(&stored, "id = ?", contactID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(stored.JobPosition) != "Legacy title" || ptrToStr(stored.FirstName) != "Remote name" || ptrToStr(stored.DistantEtag) != "remote-etag" {
		t.Fatal("rejected pull did not roll back title, name and sync state")
	}
	delete(card, vcard.FieldBirthday)
	if err := pull("valid-legacy-title"); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.First(&stored, "id = ?", contactID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(stored.JobPosition) != "New legacy title" {
		t.Fatal("contact without jobs no longer accepts TITLE updates")
	}
	delete(card, vcard.FieldTitle)
	if err := pull("clear-legacy-title"); err != nil {
		t.Fatal(err)
	}
	// Reload into a fresh struct so NULL cannot retain a previous pointer value.
	stored = models.Contact{}
	if err := svc.db.First(&stored, "id = ?", contactID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.JobPosition != nil {
		t.Fatal("omitted TITLE did not clear legacy-only title")
	}
}
