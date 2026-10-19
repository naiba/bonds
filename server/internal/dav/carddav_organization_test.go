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
	// Jobs are now the canonical projection. Clearing only a legacy pointer
	// cannot erase these jobs; an ambiguous destructive edit remains a conflict.
	cleared = perform(http.MethodPut, location, "ORG:\r\n", "Person")
	if cleared.Code != http.StatusConflict {
		t.Fatalf("empty ORG must not erase existing jobs: %d", cleared.Code)
	}
	persisted = models.Contact{}
	if err := db.First(&persisted, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.CompanyID == nil {
		t.Fatal("rejected clear changed primary company")
	}
	if err := db.First(&preservedJob, job.ID).Error; err != nil {
		t.Fatal("clear deleted employment", err)
	}
}

func TestCardDAVStructuredOrganizationCompanyNames(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, _ := createDAVHTTPTestContact(t, db, userID, "Existing", "Person")
	company := models.Company{VaultID: vaultID, Name: "Acme"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Organization names", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"organization"}}}); err != nil {
		t.Fatal(err)
	}
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	for _, tc := range []struct{ path, wire, want string }{
		{"department", "Acme;Sales", "Acme"}, {"empty-department", "Acme;", "Acme"},
		{"empty-organization", ";Sales", ""}, {"empty-components", ";", ""},
		{"literal-semicolon", `Acme\;Partners;Sales`, "Acme;Partners"},
		{"literal-backslash", `Acme\\;Sales`, `Acme\`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nORG:" + tc.wire + "\r\nEND:VCARD\r\n"
			req := httptest.NewRequest(http.MethodPut, book+tc.path+".vcf", strings.NewReader(body))
			req.SetBasicAuth(email, password)
			req.Header.Set("Content-Type", "text/vcard")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if tc.want == "" {
				if rec.Code != 409 {
					t.Fatalf("empty organization: %d", rec.Code)
				}
				return
			}
			if rec.Code != 201 {
				t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
			}
			req = httptest.NewRequest(http.MethodGet, rec.Header().Get("Location"), nil)
			req.SetBasicAuth(email, password)
			rec = httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != 200 {
				t.Fatalf("readback: %d", rec.Code)
			}
			card, err := vcard.NewDecoder(strings.NewReader(rec.Body.String())).Decode()
			if err != nil {
				t.Fatal(err)
			}
			parts := card.Get(vcard.FieldOrganization).Organization()
			if len(parts) != 1 || parts[0] != tc.want {
				t.Fatalf("wrong ORG components: %q", parts)
			}
		})
	}
	var companies []models.Company
	if err := db.Where("vault_id = ?", vaultID).Order("id ASC").Find(&companies).Error; err != nil {
		t.Fatal(err)
	}
	if len(companies) != 3 || companies[0].ID != company.ID || companies[0].Name != "Acme" || companies[1].Name != "Acme;Partners" || companies[2].Name != `Acme\` {
		t.Fatalf("incorrect company identities: count %d", len(companies))
	}
	var contacts int64
	if err := db.Model(&models.Contact{}).Where("vault_id = ?", vaultID).Count(&contacts).Error; err != nil {
		t.Fatal(err)
	}
	if contacts != 5 {
		t.Fatalf("empty ORG left contacts behind: %d", contacts)
	}
}
