package dav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
)

func TestCardDAVEmploymentBatchMatching(t *testing.T) {
	for _, scenario := range []struct {
		name                        string
		initial, incoming, expected []string
		status                      int
		filter                      bool
	}{
		{"edit first", []string{"Engineer", "Consultant"}, []string{"Architect", "Consultant"}, []string{"Architect", "Consultant"}, 201, false},
		{"prepend to single role", []string{"Engineer"}, []string{"Architect", "Engineer"}, []string{"Engineer", "Architect"}, 201, false},
		{"prepend new", []string{"Engineer", "Consultant"}, []string{"Architect", "Engineer", "Consultant"}, []string{"Engineer", "Consultant", "Architect"}, 201, false},
		{"reorder empty title", []string{"Engineer", ""}, []string{"", "Engineer"}, []string{"Engineer", ""}, 201, false},
		{"delete title", []string{"Engineer"}, []string{""}, []string{""}, 201, false},
		{"filtered deletion rolls back", []string{"Engineer"}, []string{""}, []string{"Engineer"}, 409, true},
		{"indistinguishable previous roles", []string{"Engineer", "Engineer"}, []string{"Engineer", "Architect"}, []string{"Engineer", "Engineer"}, 409, false},
		{"ambiguous edit rolls back", []string{"Engineer", "Consultant"}, []string{"Architect", "Director"}, []string{"Engineer", "Consultant"}, 409, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			userID, email, password := createDAVHTTPTestUser(t, db)
			vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Synthetic", "Batch")
			company := models.Company{VaultID: vaultID, Name: "Acme"}
			if err := db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			jobs := services.NewContactJobService(db)
			ids := []uint{}
			for _, title := range scenario.initial {
				job, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: title})
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, job.ID)
			}
			if scenario.filter {
				if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Filtered", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}); err != nil {
					t.Fatal(err)
				}
			}
			body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Batch\r\n"
			for i, title := range scenario.incoming {
				body += fmt.Sprintf("item%d.ORG:Acme\r\n", i)
				if title != "" {
					body += fmt.Sprintf("item%d.TITLE:%s\r\n", i, title)
				}
			}
			body += "END:VCARD\r\n"
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contactID)
			req := httptest.NewRequest(http.MethodPut, href, strings.NewReader(body))
			req.SetBasicAuth(email, password)
			req.Header.Set("Content-Type", "text/vcard")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != scenario.status {
				t.Fatalf("PUT=%d want %d: %s", rec.Code, scenario.status, rec.Body.String())
			}
			var actual []models.ContactCompany
			if err := db.Where("contact_id = ?", contactID).Order("id ASC").Find(&actual).Error; err != nil {
				t.Fatal(err)
			}
			if len(actual) != len(scenario.expected) {
				t.Fatalf("jobs=%d want %d", len(actual), len(scenario.expected))
			}
			for i, job := range actual {
				if ptrToStr(job.JobPosition) != scenario.expected[i] || job.CompanyID != company.ID || (i < len(ids) && job.ID != ids[i]) {
					t.Fatalf("job %d lost identity/title: id=%d company=%d title=%q", i, job.ID, job.CompanyID, ptrToStr(job.JobPosition))
				}
			}
			var count int64
			if err := db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("companies=%d", count)
			}
		})
	}
}

func TestCardDAVEditorWritesMixedCaseEmploymentGroups(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, _ := createDAVHTTPTestContact(t, db, userID, "Synthetic", "Group")
	if err := db.Model(&models.UserVault{}).Where("user_id = ? AND vault_id = ?", userID, vaultID).Update("permission", models.PermissionEditor).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Grouped", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"organization", "job_title"}}}); err != nil {
		t.Fatal(err)
	}
	body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Mixed Group\r\nN:Group;Mixed;;;\r\nitem1.ORG:Acme\r\nITEM1.TITLE:Engineer\r\nEND:VCARD\r\n"
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/grouped.vcf", userID, vaultID)
	req := httptest.NewRequest(http.MethodPut, href, strings.NewReader(body))
	req.SetBasicAuth(email, password)
	req.Header.Set("Content-Type", "text/vcard")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("valid grouped create=%d: %s", rec.Code, rec.Body.String())
	}
	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Mixed").First(&contact).Error; err != nil {
		t.Fatal(err)
	}
	jobs, err := services.NewContactJobService(db).List(contact.ID, vaultID)
	if err != nil || len(jobs) != 1 || jobs[0].JobPosition != "Engineer" || jobs[0].CompanyName != "Acme" {
		t.Fatalf("incorrect grouped employment: %+v %v", jobs, err)
	}
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID), nil)
	req.SetBasicAuth(email, password)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "TITLE:Engineer") {
		t.Fatalf("created employment not readable: %d %s", rec.Code, rec.Body.String())
	}
}
