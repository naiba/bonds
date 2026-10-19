package dav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
)

func TestCardDAVOrganizationWritesRespectFilterAndVault(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, _ := createDAVHTTPTestContact(t, db, userID, "Existing", "Contact")
	otherVaultID, _ := createDAVHTTPTestContact(t, db, userID, "Other", "Contact")
	foreign := models.Company{VaultID: otherVaultID, Name: "Synthetic Company"}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	settings := services.NewVaultSettingsService(db)
	setFilter := func(fields ...string) {
		t.Helper()
		if _, err := settings.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Company book", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: fields}}); err != nil {
			t.Fatal(err)
		}
	}
	setFilter("organization")
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	perform := func(method, path, organization, surname string) *httptest.ResponseRecorder {
		t.Helper()
		body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Person\r\nN:" + surname + ";Synthetic;;;\r\nTITLE:Engineer\r\n" + organization + "END:VCARD\r\n"
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	created := perform(http.MethodPut, book+"organization-contact.vcf", "ORG:Synthetic Company\r\n", "Person")
	if created.Code != http.StatusCreated {
		t.Fatalf("valid ORG create: want 201, got %d: %s", created.Code, created.Body.String())
	}
	location := created.Header().Get("Location")
	if location == "" {
		t.Fatal("missing created object location")
	}
	var contact models.Contact
	if err := db.Preload("Company").Where("vault_id = ? AND first_name = ?", vaultID, "Synthetic").First(&contact).Error; err != nil {
		t.Fatal(err)
	}
	if contact.Company == nil || contact.Company.Name != "Synthetic Company" || contact.Company.VaultID != vaultID || contact.Company.ID == foreign.ID {
		t.Fatalf("wrong scoped company: %+v", contact.Company)
	}
	var job models.ContactCompany
	if err := db.Where("contact_id = ? AND company_id = ?", contact.ID, *contact.CompanyID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(job.JobPosition) != "Engineer" {
		t.Fatalf("job title not imported: %+v", job)
	}
	read := perform(http.MethodGet, location, "", "")
	if read.Code != http.StatusOK {
		t.Fatalf("created object not readable: %d %s", read.Code, read.Body.String())
	}
	card, err := vcard.NewDecoder(strings.NewReader(read.Body.String())).Decode()
	if err != nil {
		t.Fatal(err)
	}
	if card.Value(vcard.FieldOrganization) != "Synthetic Company" {
		t.Fatalf("wrong ORG readback: %q", card.Value(vcard.FieldOrganization))
	}
	// Reuse the local company, never the same-named company in another Vault.
	again := perform(http.MethodPut, book+"organization-colleague.vcf", "ORG:Synthetic Company\r\n", "Colleague")
	if again.Code != http.StatusCreated {
		t.Fatalf("second create: %d %s", again.Code, again.Body.String())
	}
	var companyCount int64
	if err := db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&companyCount).Error; err != nil {
		t.Fatal(err)
	}
	if companyCount != 1 {
		t.Fatalf("expected one reused company, got %d", companyCount)
	}
	// A rejected create must roll back all newly imported rows, including the company.
	setFilter("organization", "last_name")
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
	rejected := perform(http.MethodPut, book+"organization-rejected.vcf", "ORG:Rejected Company\r\n", "")
	if rejected.Code != http.StatusConflict {
		t.Fatalf("nonmatching create: want 409, got %d", rejected.Code)
	}
	if after := countRows(); !slices.Equal(after, before) {
		t.Fatalf("rejected create left rows: before %v after %v", before, after)
	}
	// Company, job and profile writes must share the filter's rollback transaction.
	rejected = perform(http.MethodPut, location, "ORG:Rejected Update\r\n", "")
	if rejected.Code != http.StatusConflict {
		t.Fatalf("nonmatching update: want 409, got %d", rejected.Code)
	}
	if after := countRows(); !slices.Equal(after, before) {
		t.Fatalf("rejected update left rows: before %v after %v", before, after)
	}
	var persisted models.Contact
	if err := db.First(&persisted, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.CompanyID == nil || *persisted.CompanyID != *contact.CompanyID || ptrToStr(persisted.LastName) != "Person" {
		t.Fatal("rejected update changed company or surname")
	}
	// A changed ORG selects a company without renaming the shared company or deleting jobs.
	// The current go-webdav PUT handler returns 201 for both creates and updates.
	updated := perform(http.MethodPut, location, "ORG:New Company\r\n", "Person")
	if updated.Code != http.StatusCreated {
		t.Fatalf("valid company update: %d %s", updated.Code, updated.Body.String())
	}
	read = perform(http.MethodGet, location, "", "")
	card, err = vcard.NewDecoder(strings.NewReader(read.Body.String())).Decode()
	if read.Code != http.StatusOK || err != nil || card.Value(vcard.FieldOrganization) != "New Company" {
		t.Fatalf("updated ORG not readable: %d %v", read.Code, err)
	}
	var oldCompany models.Company
	if err := db.First(&oldCompany, *contact.CompanyID).Error; err != nil {
		t.Fatal(err)
	}
	if oldCompany.Name != "Synthetic Company" {
		t.Fatal("renamed shared company")
	}
	var preservedJob models.ContactCompany
	if err := db.First(&preservedJob, job.ID).Error; err != nil {
		t.Fatal("lost previous employment", err)
	}
	if preservedJob.CompanyID != job.CompanyID || ptrToStr(preservedJob.JobPosition) != ptrToStr(job.JobPosition) {
		t.Fatal("changed previous employment")
	}
	omitted := perform(http.MethodPut, location, "", "Person")
	if omitted.Code != http.StatusCreated {
		t.Fatalf("omitted ORG should preserve company: %d", omitted.Code)
	}
	cleared := perform(http.MethodPut, location, "ORG:   \r\n", "Person")
	if cleared.Code != http.StatusConflict {
		t.Fatalf("explicit clear must obey filter: %d", cleared.Code)
	}
	setFilter()
	cleared = perform(http.MethodPut, location, "ORG:\r\n", "Person")
	if cleared.Code != http.StatusCreated {
		t.Fatalf("unfiltered explicit clear: %d %s", cleared.Code, cleared.Body.String())
	}
	persisted = models.Contact{}
	if err := db.First(&persisted, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.CompanyID != nil {
		t.Fatal("explicit ORG clear did not clear primary company")
	}
	if err := db.First(&preservedJob, job.ID).Error; err != nil {
		t.Fatal("clear deleted employment", err)
	}
}
