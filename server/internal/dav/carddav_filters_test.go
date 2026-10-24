package dav

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/naiba/bonds/internal/models"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
)

func TestCardDAVFiltersExcludeWithoutArchiving(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	included := createTestContact(t, db, vaultID, userID, "Has", "Surname")
	excluded := createTestContact(t, db, vaultID, userID, "Name only", " ")
	var req dto.UpdateVaultSettingsRequest
	if err := json.Unmarshal([]byte(`{"name":"Filtered book","carddav_filter":{"match":"all","fields":["last_name"]}}`), &req); err != nil {
		t.Fatal(err)
	}
	if _, err := services.NewVaultSettingsService(db).Update(vaultID, req); err != nil {
		t.Fatal(err)
	}
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	for name, read := range map[string]func() ([]carddav.AddressObject, error){
		"list": func() ([]carddav.AddressObject, error) { return backend.ListAddressObjects(ctx, book, nil) },
		"query": func() ([]carddav.AddressObject, error) {
			return backend.QueryAddressObjects(ctx, book, &carddav.AddressBookQuery{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			objects, err := read()
			if err != nil {
				t.Fatal(err)
			}
			if len(objects) != 1 || objects[0].Path != book+included.ID+".vcf" {
				t.Fatalf("expected only matching contact, got %+v", objects)
			}
		})
	}
	if _, err := backend.GetAddressObject(ctx, book+excluded.ID+".vcf", nil); err == nil {
		t.Error("GET exposed excluded contact")
	}
	if err := db.First(excluded, "id = ?", excluded.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !excluded.Listed || excluded.DeletedAt.Valid {
		t.Fatal("filter changed contact lifecycle")
	}
}

func TestCardDAVFilterPresenceAndCombinations(t *testing.T) {
	value, blank, phone, email := " value ", "\t \n", "phone", "email"
	populated := models.Contact{Listed: true, FirstName: &value, LastName: &value, Nickname: &value, JobPosition: &value,
		Company: &models.Company{Name: value}, Addresses: []models.Address{{City: &value}},
		ContactInformations: []models.ContactInformation{
			{Data: value, ContactInformationType: models.ContactInformationType{Type: &phone}},
			{Data: value, ContactInformationType: models.ContactInformationType{Type: &email}},
		},
	}
	empty := models.Contact{Listed: true, FirstName: &blank, LastName: &blank, Nickname: &blank, JobPosition: &blank, Company: &models.Company{Name: blank}, Addresses: []models.Address{{City: &blank}}, ContactInformations: []models.ContactInformation{{Data: blank, ContactInformationType: models.ContactInformationType{Type: &phone}}}}
	fields := []string{"phone", "email", "first_name", "last_name", "nickname", "organization", "job_title", "address"}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			filter := &models.CardDAVFilter{Match: "all", Fields: []string{field}}
			if !matchesCardDAVFilter(&populated, filter) {
				t.Error("populated field rejected")
			}
			if matchesCardDAVFilter(&empty, filter) {
				t.Error("blank field included")
			}
			if matchesCardDAVFilter(&models.Contact{Listed: true}, filter) {
				t.Error("missing field included")
			}
		})
	}
	filter := &models.CardDAVFilter{Match: "all", Fields: []string{"phone", "last_name"}}
	partial := models.Contact{Listed: true, LastName: &value}
	if matchesCardDAVFilter(&partial, filter) {
		t.Error("all accepted partial match")
	}
	filter.Match = "any"
	if !matchesCardDAVFilter(&partial, filter) || matchesCardDAVFilter(&empty, filter) {
		t.Error("any semantics incorrect")
	}
	if !matchesCardDAVFilter(&empty, nil) || !matchesCardDAVFilter(&empty, &models.CardDAVFilter{Match: "all"}) {
		t.Error("default should include active contacts")
	}
	populated.Listed = false
	if matchesCardDAVFilter(&populated, nil) || matchesCardDAVFilter(&populated, filter) {
		t.Error("archived contact included")
	}
}

func TestCardDAVFilterWritesAndRecovery(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Kept", "Surname")
	hidden := createTestContact(t, db, vaultID, userID, "Hidden", "")
	service := services.NewVaultSettingsService(db)
	if _, err := service.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Book", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"last_name"}}}); err != nil {
		t.Fatal(err)
	}
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	card := vcard.Card{}
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetName(&vcard.Name{GivenName: "Changed"})
	assertStatus := func(err error, status int) {
		t.Helper()
		if err == nil || !strings.HasPrefix(err.Error(), webdav.NewHTTPError(status, nil).Error()+":") {
			t.Fatalf("expected HTTP %d, got %v", status, err)
		}
	}
	if _, err := backend.PutAddressObject(ctx, book+contact.ID+".vcf", card, nil); err != nil {
		assertStatus(err, 409)
	} else {
		t.Fatal("PUT removing required field succeeded")
	}
	var persisted models.Contact
	if err := db.First(&persisted, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(persisted.FirstName) != "Kept" || ptrToStr(persisted.LastName) != "Surname" {
		t.Fatal("rejected PUT changed contact")
	}
	var before, after int64
	if err := db.Model(&models.Contact{}).Count(&before).Error; err != nil {
		t.Fatal(err)
	}
	_, err := backend.PutAddressObject(ctx, book+"new-contact.vcf", card, nil)
	assertStatus(err, 409)
	if err := db.Model(&models.Contact{}).Count(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatal("rejected create left a contact")
	}
	card.SetName(&vcard.Name{GivenName: "Changed", FamilyName: "Surname"})
	_, err = backend.PutAddressObject(ctx, book+hidden.ID+".vcf", card, nil)
	assertStatus(err, 404)
	assertStatus(backend.DeleteAddressObject(ctx, book+hidden.ID+".vcf"), 404)
	if _, err = backend.PutAddressObject(ctx, book+contact.ID+".vcf", card, nil); err != nil {
		t.Fatalf("matching update failed: %v", err)
	}
	created, err := backend.PutAddressObject(ctx, book+"new-matching-contact.vcf", card, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = backend.GetAddressObject(ctx, created.Path, nil); err != nil {
		t.Fatal("successful PUT was not readable", err)
	}
	if _, err = service.Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Book", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{}}}); err != nil {
		t.Fatal(err)
	}
	object, err := backend.GetAddressObject(ctx, book+hidden.ID+".vcf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if object.Card.Value(vcard.FieldFormattedName) != "Hidden" {
		t.Fatal("hidden contact mutated by stale client")
	}
	if err = backend.DeleteAddressObject(ctx, created.Path); err != nil {
		t.Fatal("matching deletion failed", err)
	}
	if _, err = backend.GetAddressObject(ctx, created.Path, nil); err == nil {
		t.Fatal("deleted contact still readable")
	}
}

func TestCardDAVFilterHTTPMultigetAndPhoneData(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Phone", "Contact")
	var vault models.Vault
	if err := db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatal(err)
	}
	phoneType := models.ContactInformationType{AccountID: vault.AccountID, Type: strPtrOrNil("phone")}
	if err := db.Create(&phoneType).Error; err != nil {
		t.Fatal(err)
	}
	info := models.ContactInformation{ContactID: contactID, TypeID: phoneType.ID, Data: "  "}
	if err := db.Create(&info).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := services.NewVaultSettingsService(db).Update(vaultID, dto.UpdateVaultSettingsRequest{Name: "Phones", CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"phone"}}}); err != nil {
		t.Fatal(err)
	}
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	href := book + contactID + ".vcf"
	report := `<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><D:getetag/><C:address-data/></D:prop><D:href>` + href + `</D:href></C:addressbook-multiget>`
	perform := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "application/xml")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	hidden := perform("REPORT", book, report)
	if hidden.Code != 207 {
		t.Fatalf("multiget failed: %d %s", hidden.Code, hidden.Body.String())
	}
	var multistatus struct {
		Responses []struct {
			Href   string `xml:"href"`
			Status string `xml:"status"`
		} `xml:"response"`
	}
	if err := xml.Unmarshal(hidden.Body.Bytes(), &multistatus); err != nil {
		t.Fatal(err)
	}
	if len(multistatus.Responses) != 1 || multistatus.Responses[0].Href != href || !strings.Contains(multistatus.Responses[0].Status, "404") {
		t.Fatalf("multiget exposed filtered contact: %s", hidden.Body.String())
	}
	if rec := perform("DELETE", href, ""); rec.Code != 404 {
		t.Fatalf("stale delete status %d", rec.Code)
	}
	if err := db.Model(&info).Update("data", "+12025550123").Error; err != nil {
		t.Fatal(err)
	}
	visible := perform("REPORT", book, report)
	var decoded cardDAVMultigetReport
	if err := xml.Unmarshal(visible.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if visible.Code != 207 || len(decoded.Responses) != 1 || decoded.Responses[0].Propstat.Status != "HTTP/1.1 200 OK" || !strings.Contains(decoded.Responses[0].Propstat.Prop.AddressData, "+12025550123") {
		t.Fatalf("phone update did not restore visibility: %s", visible.Body.String())
	}
	if rec := perform("GET", href, ""); rec.Code != 200 {
		t.Fatalf("GET did not restore contact: %d", rec.Code)
	}
}
