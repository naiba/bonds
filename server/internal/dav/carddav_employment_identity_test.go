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

func TestCardDAVRoundTripPreservesEmploymentIdentity(t *testing.T) {
	for _, scenario := range []string{"company whitespace", "duplicate company names", "title whitespace"} {
		t.Run(scenario, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			userID, email, password := createDAVHTTPTestUser(t, db)
			vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Synthetic", "Identity")
			name, title := "Acme", "Engineer"
			if scenario == "company whitespace" {
				name = " \tAcme\u3000"
			}
			if scenario == "title whitespace" {
				title = " \tEngineer\u3000"
			}
			if scenario == "duplicate company names" {
				if err := db.Create(&models.Company{VaultID: vaultID, Name: name}).Error; err != nil {
					t.Fatal(err)
				}
			}
			company := models.Company{VaultID: vaultID, Name: name}
			if err := db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			jobs := services.NewContactJobService(db)
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
			if err := db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&beforeCompanies).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Employment identity", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"organization", "job_title"}}}); err != nil {
				t.Fatal(err)
			}
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contactID)
			request := func(method, body string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(method, href, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			for range 2 {
				before := request(http.MethodGet, "")
				if before.Code != 200 {
					t.Fatal(before.Code, before.Body.String())
				}
				written := request(http.MethodPut, before.Body.String())
				if written.Code != 201 {
					t.Fatal(written.Code, written.Body.String())
				}
				after := request(http.MethodGet, "")
				if after.Code != 200 {
					t.Fatal(after.Code, after.Body.String())
				}
				// PUT may advance REV; employment values and their order must stay stable.
				employmentLines := func(body string) string {
					var result []string
					for _, line := range strings.Split(body, "\r\n") {
						if strings.HasPrefix(line, "ORG:") || strings.HasPrefix(line, "TITLE:") || strings.Contains(line, ".ORG:") || strings.Contains(line, ".TITLE:") {
							result = append(result, line)
						}
					}
					return strings.Join(result, "\r\n")
				}
				if employmentLines(after.Body.String()) != employmentLines(before.Body.String()) {
					t.Fatal("unchanged PUT changed exported employment")
				}
				var stored models.Contact
				if err := db.First(&stored, "id = ?", contactID).Error; err != nil {
					t.Fatal(err)
				}
				if stored.CompanyID == nil || *stored.CompanyID != company.ID {
					t.Fatal("roundtrip changed primary company identity")
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
						t.Fatalf("employment identity changed: %+v", job)
					}
				}
				var afterCompanies int64
				if err := db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&afterCompanies).Error; err != nil {
					t.Fatal(err)
				}
				if afterCompanies != beforeCompanies {
					t.Fatal("roundtrip created a company")
				}
			}
		})
	}
}
