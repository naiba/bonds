package dav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
)

// Company names may already be blank. Their Jobs remain real records even
// though the wire represents them as TITLE without ORG, like independent roles.
func TestCardDAVTitleOnlyJobsRoundTrip(t *testing.T) {
	for _, scenario := range []string{"named and blank", "renamed blank", "blank selected", "only blank jobs", "only blank jobs with independent title", "independent title", "identical independent title", "duplicate blank roles", "invisible selected job"} {
		t.Run(scenario, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			user, email, password := createDAVHTTPTestUser(t, db)
			vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employment")
			jobs := services.NewContactJobService(db)
			companies := services.NewCompanyService(db)
			addJob := func(name, title string) uint {
				t.Helper()
				company, err := companies.Create(vault, dto.CreateCompanyRequest{Name: name})
				if err != nil {
					t.Fatal(err)
				}
				job, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: title})
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "renamed blank" && name == "To be renamed" {
					if _, err := companies.Update(company.ID, vault, dto.UpdateCompanyRequest{Name: " \t\u3000"}); err != nil {
						t.Fatal(err)
					}
				}
				return job.ID
			}
			if strings.Contains(scenario, "independent title") {
				title := "Independent artist"
				if scenario == "identical independent title" {
					title = "Whitespace role"
				}
				if _, err := jobs.LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: title}); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "invisible selected job" {
				addJob("   ", "  ")
			}
			if !strings.HasPrefix(scenario, "only blank jobs") && scenario != "blank selected" {
				addJob("Named employer", "Engineer")
			}
			name := " \t\u3000"
			if scenario == "renamed blank" {
				name = "To be renamed"
			}
			blankID := addJob(name, "Whitespace role")
			if scenario == "blank selected" {
				addJob("Named employer", "Engineer")
			}
			if strings.HasPrefix(scenario, "only blank jobs") {
				addJob("   ", "Another role")
			}
			if scenario == "duplicate blank roles" {
				addJob("   ", "Whitespace role")
			}
			note := models.Note{ContactID: id, VaultID: vault, Title: strPtrOrNil("Unrelated note"), Body: "Keep synthetic context"}
			if err := db.Create(&note).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&note, note.ID).Error; err != nil {
				t.Fatal(err)
			}
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
			request := func(method, body string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(method, href, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			read := func() (models.Contact, []models.ContactCompany) {
				t.Helper()
				var c models.Contact
				var j []models.ContactCompany
				if err := db.First(&c, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Where("contact_id = ?", id).Order("id").Find(&j).Error; err != nil {
					t.Fatal(err)
				}
				return c, j
			}
			before, originalJobs := read()
			var originalCompanies []models.Company
			if err := db.Where("vault_id = ?", vault).Order("id").Find(&originalCompanies).Error; err != nil {
				t.Fatal(err)
			}
			employmentLines := func(body string) []string {
				var lines []string
				for _, line := range strings.Split(body, "\r\n") {
					if strings.HasPrefix(line, "ORG:") || strings.HasPrefix(line, "TITLE:") || strings.Contains(line, ".ORG:") || strings.Contains(line, ".TITLE:") {
						lines = append(lines, line)
					}
				}
				sort.Strings(lines)
				return lines
			}
			for round := range 3 {
				get := request(http.MethodGet, "")
				if get.Code != 200 {
					t.Fatal(get.Code, get.Body.String())
				}
				put := request(http.MethodPut, get.Body.String())
				if put.Code != 201 {
					t.Fatalf("round %d PUT=%d: %s", round, put.Code, put.Body.String())
				}
				afterGET := request(http.MethodGet, "")
				if afterGET.Code != 200 || !reflect.DeepEqual(employmentLines(get.Body.String()), employmentLines(afterGET.Body.String())) {
					t.Errorf("round %d changed employment projection: before=%s after=%s", round, get.Body.String(), afterGET.Body.String())
				}
				after, afterJobs := read()
				if !reflect.DeepEqual(before.PreferredJobID, after.PreferredJobID) {
					t.Errorf("round %d changed preferred identity: %v -> %v", round, before.PreferredJobID, after.PreferredJobID)
				}
				if strings.Contains(scenario, "independent title") && (after.CompanyID != nil || after.PreferredJobID != nil || !reflect.DeepEqual(before.JobPosition, after.JobPosition)) {
					t.Error("independent title was retargeted")
				}
				if !reflect.DeepEqual(originalJobs, afterJobs) {
					t.Error("unchanged round trip modified, duplicated or removed Jobs")
				}
				if !reflect.DeepEqual(before.FirstName, after.FirstName) || !reflect.DeepEqual(before.LastName, after.LastName) {
					t.Error("round trip changed unrelated name")
				}
				var retained models.Note
				if err := db.First(&retained, note.ID).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(note, retained) {
					t.Error("round trip changed unrelated note")
				}
				var afterCompanies []models.Company
				if err := db.Where("vault_id = ?", vault).Order("id").Find(&afterCompanies).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(originalCompanies, afterCompanies) {
					t.Error("round trip modified or created companies")
				}
			}
			// The same matching rule must support identifiable edits, but reject a
			// partial edit of identical roles instead of guessing which row was meant.
			if scenario == "named and blank" || scenario == "duplicate blank roles" || scenario == "identical independent title" {
				beforeEdit, beforeJobs := read()
				get := request(http.MethodGet, "")
				body := strings.Replace(get.Body.String(), "TITLE:Whitespace role", "TITLE:Revised role", 1)
				body = strings.ReplaceAll(body, "Synthetic", "Changed")
				put := request(http.MethodPut, body)
				after, afterJobs := read()
				if scenario == "named and blank" {
					if put.Code != 201 {
						t.Fatalf("identifiable edit=%d: %s", put.Code, put.Body.String())
					}
					if len(afterJobs) != len(beforeJobs) || ptrToStr(after.FirstName) != "Changed" {
						t.Fatal("accepted edit missing or duplicated employment")
					}
					for i, job := range afterJobs {
						want := beforeJobs[i]
						if want.ID == blankID {
							want.JobPosition = strPtrOrNil("Revised role")
							want.UpdatedAt = job.UpdatedAt
						}
						if !reflect.DeepEqual(want, job) {
							t.Fatal("edit changed unrelated job or identity")
						}
					}
					if !reflect.DeepEqual(beforeEdit.PreferredJobID, after.PreferredJobID) {
						t.Fatal("title edit changed selection")
					}
					again := request(http.MethodGet, "")
					if put = request(http.MethodPut, again.Body.String()); put.Code != 201 {
						t.Fatal("edited job cannot round trip", put.Code, put.Body.String())
					}
				} else {
					if put.Code != 409 {
						t.Fatalf("ambiguous edit=%d, want409", put.Code)
					}
					if !reflect.DeepEqual(beforeEdit, after) || !reflect.DeepEqual(beforeJobs, afterJobs) {
						t.Fatal("rejected edit changed contact or Jobs")
					}
					afterGET := request(http.MethodGet, "")
					if afterGET.Body.String() != get.Body.String() || afterGET.Header().Get("ETag") != get.Header().Get("ETag") {
						t.Fatal("rejection changed DAV representation")
					}
				}
			}
		})
	}
}

// A partial TITLE-only update still targets the selected employment, even when
// another Job exports without ORG. Full grouped round trips are covered above.
func TestCardDAVSingleTitleEditWithBlankCompanyJob(t *testing.T) {
	for _, primary := range []string{"named job", "independent title", "blank job"} {
		t.Run(primary, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			user, email, password := createDAVHTTPTestUser(t, db)
			vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employment")
			jobs := services.NewContactJobService(db)
			if primary == "independent title" {
				if _, err := jobs.LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Independent artist"}); err != nil {
					t.Fatal(err)
				}
			}
			names := []string{"Named employer", "   "}
			if primary == "blank job" {
				names[0], names[1] = names[1], names[0]
			}
			for _, name := range names {
				company, err := services.NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: name})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Original role"}); err != nil {
					t.Fatal(err)
				}
			}
			var before models.Contact
			var originalJobs []models.ContactCompany
			if err := db.First(&before, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("contact_id = ?", id).Order("id").Find(&originalJobs).Error; err != nil {
				t.Fatal(err)
			}
			// Job creation selects an ID without populating the legacy company
			// mirror. A DAV edit may fill that mirror from the selected Job.
			expectedCompany := before.CompanyID
			for _, job := range originalJobs {
				if before.PreferredJobID != nil && job.ID == *before.PreferredJobID {
					expectedCompany = &job.CompanyID
				}
			}
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
			for _, title := range []string{"Edited primary role", ""} {
				body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Employment\r\nN:Employment;Synthetic;;;\r\nTITLE:" + title + "\r\nEND:VCARD\r\n"
				req := httptest.NewRequest(http.MethodPut, href, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				if rec.Code != http.StatusCreated {
					t.Fatalf("TITLE %q returned %d: %s", title, rec.Code, rec.Body.String())
				}
				var after models.Contact
				var afterJobs []models.ContactCompany
				if err := db.First(&after, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Where("contact_id = ?", id).Order("id").Find(&afterJobs).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before.PreferredJobID, after.PreferredJobID) || !reflect.DeepEqual(expectedCompany, after.CompanyID) || ptrToStr(after.JobPosition) != title {
					t.Error("partial TITLE edit changed selection or failed to update primary title")
				}
				if !reflect.DeepEqual(before.FirstName, after.FirstName) || !reflect.DeepEqual(before.LastName, after.LastName) {
					t.Error("partial edit changed unrelated name")
				}
				if len(originalJobs) != len(afterJobs) {
					t.Fatal("partial TITLE edit created or removed Jobs")
				}
				for i, job := range afterJobs {
					want := originalJobs[i]
					if before.PreferredJobID != nil && want.ID == *before.PreferredJobID {
						want.JobPosition = strPtrOrNil(title)
						want.UpdatedAt = job.UpdatedAt
					}
					if !reflect.DeepEqual(want, job) {
						t.Error("partial TITLE edit changed wrong Job or unrelated fields")
					}
				}
			}
		})
	}
}
