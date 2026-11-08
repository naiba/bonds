package dav

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/naiba/bonds/pkg/avatar"
)

func TestDAVVersionNegotiationLifecycle(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uploadDir := t.TempDir()
	e := echo.New()
	SetupDAVRoutes(e, db, uploadDir)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, contact := createDAVHTTPTestContact(t, db, user, "Róisín", "Contact")
	book := "/dav/addressbooks/" + user + "/" + vault + "/"
	href := book + contact + ".vcf"
	request := func(method, target, body, etag string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		if method == "REPORT" {
			req.Header.Set("Content-Type", "application/xml")
		} else {
			req.Header.Set("Content-Type", "text/vcard")
		}
		if etag != "" {
			req.Header.Set("If-Match", etag)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", contact).Update("description", "Synthetic note").Error; err != nil {
		t.Fatal(err)
	}
	photo := avatar.GenerateInitials("Synthetic portrait", 32)
	if _, err := services.NewVaultFileService(db, uploadDir).UploadContactAvatar(contact, vault, user, "portrait.png", "image/png", int64(len(photo)), bytes.NewReader(photo)); err != nil {
		t.Fatal(err)
	}
	seed := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Róisín Contact\r\nN:Contact;Róisín;;;\r\nBDAY;X-APPLE-OMIT-YEAR=1604:1604-02-29\r\nTEL;TYPE=CELL;TYPE=pref:+15550101010\r\nNOTE:Synthetic note\r\nEND:VCARD\r\n"
	if rec := request("PUT", href, seed, ""); rec.Code != 201 {
		t.Fatalf("seed PUT: %d %s", rec.Code, rec.Body.String())
	}
	original := request("GET", href, "", "")
	if original.Code != 200 || original.Header().Get("ETag") == "" {
		t.Fatalf("GET: %d %s", original.Code, original.Body.String())
	}
	etag := original.Header().Get("ETag")
	for _, kind := range []string{"addressbook-multiget", "addressbook-query"} {
		for _, version := range []string{"", "3.0", "4.0"} {
			t.Run(kind+"/"+version, func(t *testing.T) {
				attr := ""
				expected := version
				if version != "" {
					attr = ` content-type="text/vcard" version="` + version + `"`
				} else {
					expected = "3.0"
				}
				body := `<C:` + kind + ` xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><D:getetag/><C:address-data` + attr + `/></D:prop>`
				if kind == "addressbook-multiget" {
					body += `<D:href>` + href + `</D:href>`
				} else {
					body += `<C:filter/>`
				}
				body += `</C:` + kind + `>`
				rec := request("REPORT", book, body, "")
				if rec.Code != 207 {
					t.Fatalf("REPORT: %d %s", rec.Code, rec.Body.String())
				}
				var report cardDAVMultigetReport
				if err := xml.Unmarshal(rec.Body.Bytes(), &report); err != nil {
					t.Fatal(err)
				}
				if len(report.Responses) != 1 {
					t.Fatalf("responses: %+v", report)
				}
				response := report.Responses[0]
				if response.Href != href || response.Propstat.Status != "HTTP/1.1 200 OK" || response.Propstat.Prop.ETag != etag {
					t.Fatalf("response: %+v", response)
				}
				card, err := vcard.NewDecoder(strings.NewReader(response.Propstat.Prop.AddressData)).Decode()
				if err != nil {
					t.Fatal(err)
				}
				if card.Value(vcard.FieldVersion) != expected || card.Value(vcard.FieldFormattedName) != "Róisín Contact" || card.Value(vcard.FieldNote) != "Synthetic note" {
					t.Fatalf("incorrect negotiated content: %+v", card)
				}
				if expected == "3.0" {
					assertCardDAVPhoto(t, card, photo, "PNG")
					if card.Value(vcard.FieldBirthday) != "1604-02-29" || card.Get(vcard.FieldBirthday).Params.Get("X-APPLE-OMIT-YEAR") != "1604" || !card.Get(vcard.FieldTelephone).Params.HasType("pref") {
						t.Fatal("lost v3 semantics")
					}
				} else {
					if !strings.Contains(response.Propstat.Prop.AddressData, "PHOTO;VALUE=uri:data:image/png;base64,") {
						t.Fatal("v4 PHOTO must carry a literal data URI separator, not a text escape")
					}
					field := card.Get(vcard.FieldPhoto)
					if field == nil || field.Value != "data:image/png;base64,"+base64.StdEncoding.EncodeToString(photo) || field.Params.Get("ENCODING") != "" || field.Params.Get(vcard.ParamValue) != "uri" {
						t.Fatalf("invalid v4 photo: %+v", field)
					}
					if card.Value(vcard.FieldBirthday) != "--0229" || card.Get(vcard.FieldBirthday).Params.Get("X-APPLE-OMIT-YEAR") != "" || card.Get(vcard.FieldTelephone).Params.Get(vcard.ParamPreferred) != "1" || card.Value(vcard.FieldKind) != "individual" {
						t.Fatal("lost v4 semantics")
					}
				}
			})
		}
	}
	// A rejected version must not change storage or poison a legitimate retry.
	invalid := strings.Replace(seed, "VERSION:3.0", "VERSION:2.1", 1)
	if rec := request("PUT", href, invalid, etag); rec.Code != 409 {
		t.Fatalf("unsupported PUT: %d", rec.Code)
	}
	if rec := request("GET", href, "", ""); rec.Header().Get("ETag") != etag {
		t.Fatal("rejected PUT changed resource")
	}
	updated := strings.ReplaceAll(strings.Replace(seed, "VERSION:3.0", "VERSION:4.0", 1), "Róisín", "Updated")
	updated = strings.Replace(updated, "BDAY;X-APPLE-OMIT-YEAR=1604:1604-02-29", "BDAY:--0229", 1)
	updated = strings.Replace(updated, "TYPE=pref", "PREF=1", 1)
	if rec := request("PUT", href, updated, etag); rec.Code != 201 {
		t.Fatalf("v4 PUT: %d %s", rec.Code, rec.Body.String())
	}
	readback := request("GET", href, "", "")
	if readback.Code != 200 || readback.Header().Get("ETag") == etag || !strings.Contains(readback.Body.String(), "FN:Updated Contact") || !strings.Contains(readback.Body.String(), "VERSION:3.0") {
		t.Fatalf("readback: %d %s", readback.Code, readback.Body.String())
	}
	if rec := request("PUT", href, seed, etag); rec.Code != 412 {
		t.Fatalf("stale PUT: %d %s", rec.Code, rec.Body.String())
	}
	var stored models.Contact
	if err := db.First(&stored, "id = ?", contact).Error; err != nil {
		t.Fatal(err)
	}
	if stored.FirstName == nil || *stored.FirstName != "Updated" || stored.FileID == nil {
		t.Fatalf("persisted state lost: %+v", stored)
	}
	if rec := request("GET", href, "", ""); rec.Header().Get("ETag") != readback.Header().Get("ETag") {
		t.Fatal("stale PUT changed resource")
	}
	// Organization reconciliation precedes birthday validation; a bad v4 birthday
	// must roll back the company as well as the contact name and phone.
	broken := strings.Replace(updated, "BDAY:--0229", "ORG:Synthetic rollback company\r\nBDAY:--0230", 1)
	if rec := request("PUT", href, broken, readback.Header().Get("ETag")); rec.Code != 400 {
		t.Fatalf("invalid v4 birthday: %d %s", rec.Code, rec.Body.String())
	}
	var companies int64
	if err := db.Model(&models.Company{}).Where("vault_id = ? AND name = ?", vault, "Synthetic rollback company").Count(&companies).Error; err != nil || companies != 0 {
		t.Fatalf("company escaped rollback: %d %v", companies, err)
	}
	if rec := request("GET", href, "", ""); rec.Code != 200 || rec.Body.String() != readback.Body.String() || rec.Header().Get("ETag") != readback.Header().Get("ETag") {
		t.Fatal("invalid v4 birthday changed persisted representation")
	}
	if err := db.Model(&models.UserVault{}).Where("user_id = ? AND vault_id = ?", user, vault).Update("permission", models.PermissionViewer).Error; err != nil {
		t.Fatal(err)
	}
	if rec := request("PUT", href, updated, ""); rec.Code != 403 {
		t.Fatalf("viewer v4 write: %d", rec.Code)
	}
	reportBody := `<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><D:getetag/><C:address-data version="4.0"/></D:prop><D:href>` + href + `</D:href></C:addressbook-multiget>`
	viewerRead := request("REPORT", book, reportBody, "")
	var report cardDAVMultigetReport
	if err := xml.Unmarshal(viewerRead.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if viewerRead.Code != 207 || len(report.Responses) != 1 || report.Responses[0].Propstat.Status != "HTTP/1.1 200 OK" || report.Responses[0].Propstat.Prop.ETag != readback.Header().Get("ETag") || !strings.Contains(report.Responses[0].Propstat.Prop.AddressData, "FN:Updated Contact") || !strings.Contains(report.Responses[0].Propstat.Prop.AddressData, "VERSION:4.0") {
		t.Fatalf("viewer v4 read: %d %s", viewerRead.Code, viewerRead.Body.String())
	}
	if err := db.Model(&models.UserVault{}).Where("user_id = ? AND vault_id = ?", user, vault).Update("permission", models.PermissionManager).Error; err != nil {
		t.Fatal(err)
	}
	if rec := request("DELETE", href, "", ""); rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	if rec := request("PUT", href, updated, ""); rec.Code != 410 {
		t.Fatalf("delayed v4 upload resurrected contact: %d", rec.Code)
	}
	var remaining int64
	if err := db.Model(&models.Contact{}).Where("id = ?", contact).Count(&remaining).Error; err != nil || remaining != 0 {
		t.Fatalf("deleted contact restored: %d %v", remaining, err)
	}
	missing := request("REPORT", book, reportBody, "")
	var missingReport cardDAVMultigetErrorReport
	if err := xml.Unmarshal(missing.Body.Bytes(), &missingReport); err != nil {
		t.Fatal(err)
	}
	if missing.Code != 207 || len(missingReport.Response) != 1 || missingReport.Response[0].Status != "HTTP/1.1 404 Not Found" || len(missingReport.Response[0].Propstats) != 0 {
		t.Fatalf("deleted v4 read: %d %s", missing.Code, missing.Body.String())
	}

}

func TestDAVRemotePhotoURIBytes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	e := echo.New()
	SetupDAVRoutes(e, db, "")
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, contact := createDAVHTTPTestContact(t, db, user, "Remote", "Portrait")
	photoURL := "https://example.test/portraits/a,b.png"
	file := models.File{VaultID: vault, UUID: "synthetic-remote-portrait", OriginalURL: &photoURL, MimeType: "image/png", Name: "portrait.png", Type: "image"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", contact).Update("file_id", file.ID).Error; err != nil {
		t.Fatal(err)
	}
	book := "/dav/addressbooks/" + user + "/" + vault + "/"
	exported, err := services.NewVCardService(db).ExportContact(contact, vault)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(exported), ":"+photoURL+"\r\n") {
		t.Errorf("export PHOTO URI bytes changed: %q", exported)
	}
	for _, version := range []string{"3.0", "4.0"} {
		t.Run(version, func(t *testing.T) {
			body := `<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><C:address-data version="` + version + `"/></D:prop><D:href>` + book + contact + `.vcf</D:href></C:addressbook-multiget>`
			req := httptest.NewRequest("REPORT", book, strings.NewReader(body))
			req.SetBasicAuth(email, password)
			req.Header.Set("Content-Type", "application/xml")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			var report cardDAVMultigetReport
			if err := xml.Unmarshal(rec.Body.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 207 || len(report.Responses) != 1 || report.Responses[0].Propstat.Status != "HTTP/1.1 200 OK" {
				t.Fatalf("REPORT: %d %s", rec.Code, rec.Body.String())
			}
			wire := report.Responses[0].Propstat.Prop.AddressData
			if !strings.Contains(wire, ":"+photoURL+"\r\n") {
				t.Fatalf("URI bytes changed in PHOTO: %q", wire)
			}
		})
	}
}
