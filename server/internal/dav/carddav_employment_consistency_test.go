package dav

import (
	"fmt"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCardDAVLegacyEmploymentAndAmbiguousTitles(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employment")
	company := models.Company{VaultID: vault, Name: "Legacy Acme"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": company.ID, "job_position": "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	request := func(method, fields string) *httptest.ResponseRecorder {
		body := ""
		if method == http.MethodPut {
			body = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Employment\r\n" + fields + "END:VCARD\r\n"
		}
		req := httptest.NewRequest(method, href, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	for attempt := 0; attempt < 2; attempt++ {
		rec := request(http.MethodPut, "ORG:New Beta\r\nTITLE:Consultant\r\n")
		if rec.Code != 201 {
			t.Fatalf("PUT=%d %s", rec.Code, rec.Body.String())
		}
	}
	jobs, err := services.NewContactJobService(db).List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("employment count=%d", len(jobs))
	}
	foundLegacy, foundNew := false, false
	for _, job := range jobs {
		if job.CompanyID == company.ID && job.JobPosition == "Engineer" {
			foundLegacy = true
		}
		if job.CompanyName == "New Beta" && job.JobPosition == "Consultant" {
			foundNew = true
		}
	}
	if !foundLegacy || !foundNew {
		t.Fatal("DAV replaced the only legacy relationship")
	}
	before := request(http.MethodGet, "")
	if before.Code != 200 {
		t.Fatal(before.Code)
	}
	for _, fields := range []string{
		"ORG:Legacy Acme\r\nORG:New Beta\r\nTITLE:Engineer\r\n",
		"item1.ORG:Legacy Acme\r\nITEM1.ORG:New Beta\r\nitem1.TITLE:Engineer\r\n",
	} {
		rec := request(http.MethodPut, fields)
		if rec.Code != 409 {
			t.Fatalf("ambiguous PUT=%d %s", rec.Code, rec.Body.String())
		}
		after := request(http.MethodGet, "")
		if after.Code != 200 || after.Body.String() != before.Body.String() || after.Header().Get("ETag") != before.Header().Get("ETag") {
			t.Fatal("rejected DAV write changed contact representation or ETag")
		}
	}
}
