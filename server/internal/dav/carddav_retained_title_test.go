package dav

import (
	"fmt"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestCardDAVRetainedJobDoesNotOverwriteOmittedStandaloneTitle(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employment")
	jobs := services.NewContactJobService(db)
	if _, err := jobs.LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Artist"}); err != nil {
		t.Fatal(err)
	}
	company, err := services.NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "   "})
	if err != nil {
		t.Fatal(err)
	}
	job, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	var before models.Contact
	var originalJob models.ContactCompany
	if err := db.First(&before, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&originalJob, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if before.CompanyID != nil || before.PreferredJobID != nil || ptrToStr(before.JobPosition) != "Artist" {
		t.Fatal("fixture did not retain independent employment")
	}
	note := models.Note{ContactID: id, VaultID: vault, Title: strPtrOrNil("Unrelated context"), Body: "Preserve unrelated data"}
	if err := db.Create(&note).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&note, note.ID).Error; err != nil {
		t.Fatal(err)
	}
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	request := func(method, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, href, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	get := request(http.MethodGet, "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), "employment1.TITLE:Artist\r\n") || !strings.Contains(get.Body.String(), "employment2.TITLE:Engineer\r\n") {
		t.Fatalf("unexpected original projection: %d %s", get.Code, get.Body.String())
	}
	body := strings.Replace(get.Body.String(), "employment1.TITLE:Artist\r\n", "", 1)
	// Group labels pair fields; arbitrary labels must retain the same identity.
	body = strings.ReplaceAll(body, "employment2.", "retainedRole.")
	put := request(http.MethodPut, body)
	afterGet := request(http.MethodGet, "")
	var after models.Contact
	var retainedJob models.ContactCompany
	var retainedNote models.Note
	if err := db.First(&after, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&retainedJob, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&retainedNote, note.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(originalJob, retainedJob) || !reflect.DeepEqual(note, retainedNote) {
		t.Error("unrelated persisted job/note changed")
	}
	t.Logf("PUT status=%d; standalone before=%q after=%q; GET=%s", put.Code, ptrToStr(before.JobPosition), ptrToStr(after.JobPosition), afterGet.Body.String())

	if put.Code != http.StatusCreated {
		t.Fatalf("unexpected status %d: %s", put.Code, put.Body.String())
	}
	if after.JobPosition != nil {
		t.Errorf("accepted omission must clear standalone title, got %q copied from retained Job", ptrToStr(after.JobPosition))
	}
	if afterGet.Code != http.StatusOK || strings.Count(afterGet.Body.String(), "TITLE:Engineer\r\n") != 1 {
		t.Error("accepted update duplicated or lost retained employment")
	}
	for range 3 {
		round := request(http.MethodPut, afterGet.Body.String())
		afterGet = request(http.MethodGet, "")
		if round.Code != http.StatusCreated || afterGet.Code != http.StatusOK || strings.Count(afterGet.Body.String(), "TITLE:Engineer\r\n") != 1 || strings.Contains(afterGet.Body.String(), "TITLE:Artist\r\n") {
			t.Fatal("repeated round trip duplicated or revived employment")
		}
		var stored models.ContactCompany
		if err := db.First(&stored, job.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.ID != originalJob.ID || stored.CompanyID != originalJob.CompanyID || stored.ContactID != originalJob.ContactID || ptrToStr(stored.JobPosition) != ptrToStr(originalJob.JobPosition) {
			t.Fatal("round trip changed retained Job identity/content")
		}
	}
}

func TestCardDAVGroupedRetainedJobDoesNotRetargetSelectedEmployment(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employment")
	jobs := services.NewContactJobService(db)
	for _, entry := range [][2]string{{"Named employer", "Analyst"}, {"   ", "Engineer"}} {
		company, err := services.NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: entry[0]})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: entry[1]}); err != nil {
			t.Fatal(err)
		}
	}
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	request := func(method, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, href, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	var before models.Contact
	var beforeJobs []models.ContactCompany
	if err := db.First(&before, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("contact_id = ?", id).Order("id").Find(&beforeJobs).Error; err != nil {
		t.Fatal(err)
	}
	get := request(http.MethodGet, "")
	if get.Code != 200 || !strings.Contains(get.Body.String(), "employment1.ORG:Named employer\r\n") || !strings.Contains(get.Body.String(), "employment2.TITLE:Engineer\r\n") {
		t.Fatalf("unexpected fixture: %d %s", get.Code, get.Body.String())
	}
	body := strings.Replace(get.Body.String(), "employment1.ORG:Named employer\r\n", "", 1)
	body = strings.Replace(body, "employment1.TITLE:Analyst\r\n", "", 1)
	// Group labels pair fields; arbitrary labels must retain the same identity.
	body = strings.ReplaceAll(body, "employment2.", "retainedRole.")
	put := request(http.MethodPut, body)
	afterGet := request(http.MethodGet, "")
	var after models.Contact
	var afterJobs []models.ContactCompany
	if err := db.First(&after, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("contact_id = ?", id).Order("id").Find(&afterJobs).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("PUT=%d retained-only input=%s resulting GET=%s", put.Code, body, afterGet.Body.String())

	if put.Code != 201 {
		t.Fatalf("unexpected response=%d %s", put.Code, put.Body.String())
	}
	if !reflect.DeepEqual(before.PreferredJobID, after.PreferredJobID) || after.CompanyID == nil || *after.CompanyID != beforeJobs[0].CompanyID {
		t.Error("retained secondary title changed selected employment")
	}
	if len(afterJobs) != 2 {
		t.Fatal("normal jobs unexpectedly removed")
	}
	for i, job := range afterJobs {
		if job.ID != beforeJobs[i].ID || job.CompanyID != beforeJobs[i].CompanyID || ptrToStr(job.JobPosition) != ptrToStr(beforeJobs[i].JobPosition) {
			t.Errorf("unchanged retained secondary TITLE retargeted primary job: before=%q after=%q", ptrToStr(beforeJobs[i].JobPosition), ptrToStr(job.JobPosition))
		}
	}
	for range 3 {
		round := request(http.MethodPut, afterGet.Body.String())
		afterGet = request(http.MethodGet, "")
		if round.Code != http.StatusCreated || afterGet.Code != http.StatusOK || strings.Count(afterGet.Body.String(), "TITLE:Engineer\r\n") != 1 || strings.Count(afterGet.Body.String(), "TITLE:Analyst\r\n") != 1 {
			t.Fatal("repeated round trip lost or duplicated a role")
		}
		var stored []models.ContactCompany
		if err := db.Where("contact_id = ?", id).Order("id").Find(&stored).Error; err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored, beforeJobs) {
			t.Fatal("round trip changed original Jobs")
		}
	}
}

func TestCardDAVGroupedTitleRejectsAmbiguousEmployment(t *testing.T) {
	for _, independent := range []string{"Artist", "Engineer"} {
		t.Run(independent, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			user, email, password := createDAVHTTPTestUser(t, db)
			vault, id := createDAVHTTPTestContact(t, db, user, "Unchanged", "Person")
			jobs := services.NewContactJobService(db)
			if _, err := jobs.LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: independent}); err != nil {
				t.Fatal(err)
			}
			company, err := services.NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "   "})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"}); err != nil {
				t.Fatal(err)
			}
			note := models.Note{ContactID: id, VaultID: vault, Body: "Unrelated note"}
			if err := db.Create(&note).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&note, note.ID).Error; err != nil {
				t.Fatal(err)
			}
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
			request := func(method, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, href, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			beforeGET := request(http.MethodGet, "")
			if beforeGET.Code != http.StatusOK {
				t.Fatal(beforeGET.Code)
			}
			var before models.Contact
			var beforeJobs []models.ContactCompany
			if err := db.First(&before, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("contact_id = ?", id).Order("id").Find(&beforeJobs).Error; err != nil {
				t.Fatal(err)
			}
			title := "Unknown role"
			if independent == "Engineer" {
				title = "Engineer"
			}
			body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Changed Person\r\nN:Person;Changed;;;\r\nclientRole.TITLE:" + title + "\r\nEND:VCARD\r\n"
			for range 2 {
				put := request(http.MethodPut, body)
				if put.Code != http.StatusConflict {
					t.Fatalf("ambiguous title PUT=%d: %s", put.Code, put.Body.String())
				}
				var after models.Contact
				var afterJobs []models.ContactCompany
				var afterNote models.Note
				if err := db.First(&after, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Where("contact_id = ?", id).Order("id").Find(&afterJobs).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&afterNote, note.ID).Error; err != nil {
					t.Fatal(err)
				}
				afterGET := request(http.MethodGet, "")
				if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeJobs, afterJobs) || !reflect.DeepEqual(note, afterNote) || afterGET.Body.String() != beforeGET.Body.String() || afterGET.Header().Get("ETag") != beforeGET.Header().Get("ETag") {
					t.Fatal("ambiguous input escaped transaction rollback")
				}
			}
		})
	}
}
