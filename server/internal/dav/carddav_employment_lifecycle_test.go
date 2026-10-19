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

func TestCardDAVLastEmploymentRemovalClearsTitle(t *testing.T) {
	for _, removal := range []string{"job", "employee", "company"} {
		t.Run(removal, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			userID, email, password := createDAVHTTPTestUser(t, db)
			vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Synthetic", "Employment")
			company := models.Company{VaultID: vaultID, Name: "Acme"}
			if err := db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			jobs := services.NewContactJobService(db)
			job, err := jobs.Create(contactID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
			if err != nil {
				t.Fatal(err)
			}
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contactID)
			request := func(method, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, href, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			rec := request(http.MethodPut, "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Employment\r\nTITLE:Architect\r\nEND:VCARD\r\n")
			if rec.Code != 201 {
				t.Fatalf("title-only edit=%d", rec.Code)
			}
			before, err := jobs.List(contactID, vaultID)
			if err != nil || len(before) != 1 || before[0].ID != job.ID || before[0].JobPosition != "Architect" {
				t.Fatalf("edit not applied: %+v %v", before, err)
			}
			switch removal {
			case "job":
				err = jobs.Delete(contactID, vaultID, job.ID)
			case "employee":
				err = jobs.RemoveEmployee(company.ID, vaultID, contactID)
			case "company":
				err = services.NewCompanyService(db).Delete(company.ID, vaultID)
			}
			if err != nil {
				t.Fatal(err)
			}
			after, err := jobs.List(contactID, vaultID)
			if err != nil || len(after) != 0 {
				t.Fatalf("employment not removed: %+v %v", after, err)
			}
			rec = request(http.MethodGet, "")
			if rec.Code != 200 || strings.Contains(rec.Body.String(), "TITLE:") || strings.Contains(rec.Body.String(), "ORG:") {
				t.Fatalf("deleted employment resurrected: %d %s", rec.Code, rec.Body.String())
			}
			var contact models.Contact
			if err := db.First(&contact, "id = ?", contactID).Error; err != nil {
				t.Fatal(err)
			}
			if contact.CompanyID != nil || ptrToStr(contact.JobPosition) != "" {
				t.Fatal("obsolete compatibility fields remain")
			}
			if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Employment", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}); err != nil {
				t.Fatal(err)
			}
			if rec := request(http.MethodGet, ""); rec.Code != 404 {
				t.Fatalf("empty employment matches job_title filter: %d", rec.Code)
			}
		})
	}
}

func TestCardDAVOptionalEmploymentGroupsRespectTitleFilter(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, _ := createDAVHTTPTestContact(t, db, userID, "Existing", "Contact")
	if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Employment", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, fields string
		status       int
	}{
		{"grouped organization", "item1.ORG:Acme\r\nTITLE:Engineer\r\n", 201},
		{"grouped title", "ORG:Acme\r\nitem1.TITLE:Engineer\r\n", 201},
		{"ambiguous title", "item1.ORG:Acme\r\nitem2.ORG:Beta\r\nTITLE:Engineer\r\n", 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			countRows := func() []int64 {
				t.Helper()
				counts := make([]int64, 3)
				for i, model := range []any{&models.Contact{}, &models.Company{}, &models.ContactCompany{}} {
					if err := db.Model(model).Count(&counts[i]).Error; err != nil {
						t.Fatal(err)
					}
				}
				return counts
			}
			before := countRows()
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/optional-group.vcf", userID, vaultID)
			req := httptest.NewRequest(http.MethodPut, href, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\n"+tc.fields+"END:VCARD\r\n"))
			req.SetBasicAuth(email, password)
			req.Header.Set("Content-Type", "text/vcard")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("PUT=%d want %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			if tc.status == 409 {
				after := countRows()
				for i := range before {
					if before[i] != after[i] {
						t.Fatal("ambiguous write left domain records")
					}
				}
				return
			}
			req = httptest.NewRequest(http.MethodGet, rec.Header().Get("Location"), nil)
			req.SetBasicAuth(email, password)
			rec = httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != 200 || !strings.Contains(rec.Body.String(), "TITLE:Engineer") || !strings.Contains(rec.Body.String(), "ORG:Acme") {
				t.Fatalf("created employment missing: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}
