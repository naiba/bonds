package dav

import (
	"fmt"
	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestCardDAVEmploymentSelectionSurvivesSecondaryCompanyChange(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Selection")
	companyService := services.NewCompanyService(db)
	a, err := companyService.Create(vault, dto.CreateCompanyRequest{Name: "Earlier employer"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := companyService.Create(vault, dto.CreateCompanyRequest{Name: "Selected employer"})
	if err != nil {
		t.Fatal(err)
	}
	jobs := services.NewContactJobService(db)
	earlier, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: a.ID, JobPosition: "Secondary engineer"})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: b.ID, JobPosition: "Selected consultant"})
	if err != nil {
		t.Fatal(err)
	}
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	request := func(method, fields string) *httptest.ResponseRecorder {
		body := ""
		if method == http.MethodPut {
			body = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Selection\r\n" + fields + "END:VCARD\r\n"
		}
		req := httptest.NewRequest(method, href, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	// Select the later existing employer via the normal DAV update path.
	if rec := request(http.MethodPut, "ORG:Selected employer\r\nTITLE:Selected consultant\r\n"); rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	if _, err = jobs.Update(id, vault, earlier.ID, dto.UpdateContactJobRequest{CompanyID: b.ID, JobPosition: "Secondary engineer"}); err != nil {
		t.Fatal(err)
	}
	response, err := services.NewContactService(db).GetContact(id, user, vault)
	if err != nil {
		t.Fatal(err)
	}
	get := request(http.MethodGet, "")
	if get.Code != 200 {
		t.Fatal(get.Code, get.Body.String())
	}
	card, err := vcard.NewDecoder(strings.NewReader(get.Body.String())).Decode()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("contact response title=%q DAV first TITLE=%q earlierJob=%d selectedJob=%d", response.JobPosition, card.Value(vcard.FieldTitle), earlier.ID, selected.ID)
	if response.JobPosition != "Selected consultant" || card.Value(vcard.FieldTitle) != "Selected consultant" {
		t.Error("secondary company edit changed selected employment projection")
	}
	if rec := request(http.MethodPut, "TITLE:Edited selected consultant\r\n"); rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	var storedEarlier, storedSelected models.ContactCompany
	if err := db.First(&storedEarlier, earlier.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&storedSelected, selected.ID).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("after title-only PUT: earlier=%q selected=%q", ptrToStr(storedEarlier.JobPosition), ptrToStr(storedSelected.JobPosition))
	if ptrToStr(storedEarlier.JobPosition) != "Secondary engineer" || ptrToStr(storedSelected.JobPosition) != "Edited selected consultant" {
		t.Error("TITLE-only PUT modified the wrong persistent employment row")
	}

	// A filter rejects the completed write, including its preferred-job metadata.
	if err := db.Model(&models.Contact{}).Where("id = ?", id).Update("last_name", "Required surname").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := services.NewVaultSettingsService(db).Update(vault, dto.UpdateVaultSettingsRequest{Name: "Selection book", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"last_name"}}}); err != nil {
		t.Fatal(err)
	}
	var beforeContact, afterContact models.Contact
	if err := db.First(&beforeContact, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	var beforeJobs, afterJobs []models.ContactCompany
	if err := db.Where("contact_id = ?", id).Order("id ASC").Find(&beforeJobs).Error; err != nil {
		t.Fatal(err)
	}
	if rec := request(http.MethodPut, "TITLE:Must roll back\r\n"); rec.Code != http.StatusConflict {
		t.Fatalf("filter rejection=%d %s", rec.Code, rec.Body.String())
	}
	if err := db.First(&afterContact, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("contact_id = ?", id).Order("id ASC").Find(&afterJobs).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeContact, afterContact) || !reflect.DeepEqual(beforeJobs, afterJobs) {
		t.Fatal("filter rejection changed selection, jobs or unrelated profile state")
	}
}

func TestCardDAVNewEmploymentRejectsAmbiguousCompanyIdentity(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, _ := createDAVHTTPTestContact(t, db, user, "Synthetic", "Existing")
	companyService := services.NewCompanyService(db)
	for range 2 {
		if _, err := companyService.Create(vault, dto.CreateCompanyRequest{Name: "Same company name"}); err != nil {
			t.Fatal(err)
		}
	}
	count := func(model any) int64 {
		t.Helper()
		var n int64
		if err := db.Model(model).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		return n
	}
	beforeContacts, beforeJobs, beforeCompanies := count(&models.Contact{}), count(&models.ContactCompany{}), count(&models.Company{})
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/ambiguous-employer.vcf", user, vault)
	req := httptest.NewRequest(http.MethodPut, href, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Employee\r\nORG:Same company name\r\nTITLE:Engineer\r\nEND:VCARD\r\n"))
	req.SetBasicAuth(email, password)
	req.Header.Set("Content-Type", "text/vcard")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var jobs []models.ContactCompany
	if err := db.Order("id ASC").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("PUT=%d jobs=%+v", rec.Code, jobs)
	if rec.Code != http.StatusConflict {
		t.Errorf("ambiguous new company association returned %d, want 409", rec.Code)
	}
	if count(&models.Contact{}) != beforeContacts || count(&models.ContactCompany{}) != beforeJobs || count(&models.Company{}) != beforeCompanies {
		t.Error("ambiguous ORG persisted a contact/employment instead of leaving domain state unchanged")
	}
}
