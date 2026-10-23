package dav

import (
	"fmt"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmploymentAmbiguousTitleOnlyWrite(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employment")
	jobs := services.NewContactJobService(db)
	for _, name := range []string{"Acme", "Beta"} {
		company := models.Company{VaultID: vault, Name: name}
		if err := db.Create(&company).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Original " + name}); err != nil {
			t.Fatal(err)
		}
	}
	before, err := jobs.List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	req := httptest.NewRequest(http.MethodPut, href, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Employment\r\nemployment1.TITLE:Changed Acme\r\nemployment2.TITLE:Changed Beta\r\nEND:VCARD\r\n"))
	req.SetBasicAuth(email, password)
	req.Header.Set("Content-Type", "text/vcard")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	after, err := jobs.List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PUT status=%d before=%+v after=%+v", rec.Code, before, after)
	if rec.Code != http.StatusConflict {
		t.Errorf("ambiguous multiple title-only write returned %d; expected 409 without selecting a job by position", rec.Code)
	}
	if len(after) != len(before) {
		t.Fatal("job count changed")
	}
	for i := range before {
		if before[i].ID != after[i].ID || before[i].JobPosition != after[i].JobPosition {
			t.Error("ambiguous write changed existing employment")
		}
	}
}

func TestEmploymentUnchangedRoundTripPreservesSelectedTitle(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Selection")
	jobs := services.NewContactJobService(db)
	var primary uint
	for i, name := range []string{"Acme", "Beta"} {
		c := models.Company{VaultID: vault, Name: name}
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
		title := ""
		if i == 1 {
			title = "Consultant"
		} else {
			primary = c.ID
		}
		if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: c.ID, JobPosition: title}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": primary, "job_position": nil}).Error; err != nil {
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
	if get.Code != 200 {
		t.Fatal(get.Code)
	}
	put := request(http.MethodPut, get.Body.String())
	if put.Code != 201 {
		t.Fatal(put.Code, put.Body.String())
	}
	var contact models.Contact
	if err := db.First(&contact, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("primary=%d selected=%v stored title=%q", primary, contact.CompanyID, ptrToStr(contact.JobPosition))
	if contact.CompanyID == nil || *contact.CompanyID != primary || ptrToStr(contact.JobPosition) != "" {
		t.Error("unchanged GET/PUT assigned another company's title to the selected company")
	}
	response, err := services.NewContactService(db).GetContact(id, user, vault)
	if err != nil {
		t.Fatal(err)
	}
	if response.CompanyID == nil || *response.CompanyID != primary || response.JobPosition != "" {
		t.Fatal("contact response has the wrong selected title")
	}
}

func TestEmploymentSingleTitleOnlyWriteRemainsEditable(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Title")
	company := models.Company{VaultID: vault, Name: "Acme"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	jobs := services.NewContactJobService(db)
	before, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	req := httptest.NewRequest(http.MethodPut, href, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Title\r\nTITLE:Architect\r\nEND:VCARD\r\n"))
	req.SetBasicAuth(email, password)
	req.Header.Set("Content-Type", "text/vcard")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	after, err := jobs.List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	response, err := services.NewContactService(db).GetContact(id, user, vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].ID != before.ID || after[0].JobPosition != "Architect" || response.CompanyID == nil || *response.CompanyID != company.ID || response.JobPosition != "Architect" {
		t.Fatal("single-title edit or contact response is inconsistent")
	}
}
