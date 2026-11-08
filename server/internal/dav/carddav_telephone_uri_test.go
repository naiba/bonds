package dav

import (
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestCardDAVTelephoneURILifecycle(t *testing.T) {
	for _, number := range []string{"+15550101010", "+1-555-0101;ext=42", "7042;phone-context=example.test", "+15550101010;isub=desk%2Fwest"} {
		t.Run(number, func(t *testing.T) {
			db := testutil.SetupTestDB(t)
			e := echo.New()
			SetupDAVRoutes(e, db, "")
			user, email, password := createDAVHTTPTestUser(t, db)
			vault, unrelated := createDAVHTTPTestContact(t, db, user, "Unrelated", "Person")
			book := "/dav/addressbooks/" + user + "/" + vault + "/"
			request := func(method, path, body string) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				if method == "REPORT" {
					req.Header.Set("Content-Type", "application/xml")
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			card := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:URI Contact\r\nN:Contact;URI;;;\r\nTEL;VALUE=uri;TYPE=cell;PREF=1:tel:" + number + "\r\nEMAIL;TYPE=home:preserve@example.test\r\nEND:VCARD\r\n"
			rec := request("PUT", book+"telephone-contact.vcf", card)
			if rec.Code != 201 {
				t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
			}
			var contact models.Contact
			if err := db.Where("vault_id = ? AND first_name = ?", vault, "URI").First(&contact).Error; err != nil {
				t.Fatal(err)
			}
			href := book + contact.ID + ".vcf"
			var phone, emailRow models.ContactInformation
			if err := db.Where("contact_id = ? AND data <> ?", contact.ID, "preserve@example.test").First(&phone).Error; err != nil {
				t.Fatal(err)
			}
			if phone.Data != number || phone.Kind == nil || *phone.Kind != "mobile" || !phone.Pref {
				t.Fatalf("URI must store a dialable subscriber, not a second tel scheme: %+v", phone)
			}
			if err := db.Where("contact_id = ? AND data = ?", contact.ID, "preserve@example.test").First(&emailRow).Error; err != nil {
				t.Fatal(err)
			}
			// Existing user aliases and IDs must survive equivalent URI/text updates.
			if err := db.Model(&phone).Update("kind", "cellphone").Error; err != nil {
				t.Fatal(err)
			}
			for _, wire := range []string{card, card, strings.Replace(card, "tel:"+number, "TEL:"+number, 1)} {
				if rec := request("PUT", href, wire); rec.Code != 201 {
					t.Fatalf("repeat PUT: %d %s", rec.Code, rec.Body.String())
				}
			}
			get := request("GET", href, "")
			if get.Code != 200 {
				t.Fatalf("GET: %d", get.Code)
			}
			parsed, err := vcard.NewDecoder(strings.NewReader(get.Body.String())).Decode()
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Value(vcard.FieldTelephone) != number || parsed.Value(vcard.FieldVersion) != "3.0" {
				t.Fatalf("GET telephone/version: %+v", parsed)
			}
			if rec := request("PUT", href, get.Body.String()); rec.Code != 201 {
				t.Fatalf("v3 roundtrip: %d", rec.Code)
			}
			reportBody := `<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><C:address-data version="4.0"/></D:prop><D:href>` + href + `</D:href></C:addressbook-multiget>`
			reportRec := request("REPORT", book, reportBody)
			var report cardDAVMultigetReport
			if err := xml.Unmarshal(reportRec.Body.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if reportRec.Code != 207 || len(report.Responses) != 1 || report.Responses[0].Propstat.Status != "HTTP/1.1 200 OK" {
				t.Fatalf("REPORT: %d %s", reportRec.Code, reportRec.Body.String())
			}
			parsed, err = vcard.NewDecoder(strings.NewReader(report.Responses[0].Propstat.Prop.AddressData)).Decode()
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Value(vcard.FieldTelephone) != number || parsed.Value(vcard.FieldVersion) != "4.0" {
				t.Fatalf("REPORT telephone/version: %+v", parsed)
			}
			var saved models.ContactInformation
			if err := db.First(&saved, phone.ID).Error; err != nil || saved.Data != number || saved.Kind == nil || *saved.Kind != "cellphone" || !saved.Pref {
				t.Fatalf("phone identity/metadata lost: %+v %v", saved, err)
			}
			var savedEmail models.ContactInformation
			if err := db.First(&savedEmail, emailRow.ID).Error; err != nil || savedEmail.Data != emailRow.Data || savedEmail.Pref != emailRow.Pref || savedEmail.Kind == nil || *savedEmail.Kind != *emailRow.Kind {
				t.Fatalf("unrelated email changed: %+v %v", savedEmail, err)
			}
			var count int64
			if err := db.Model(&models.ContactInformation{}).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil || count != 2 {
				t.Fatalf("repeated input duplicated information: %d %v", count, err)
			}
			// Repair an old URI-shaped stored value in place on the next valid upload.
			if err := db.Model(&saved).Update("data", "tel:"+number).Error; err != nil {
				t.Fatal(err)
			}
			legacy := request("GET", href, "")
			if legacy.Code != 200 {
				t.Fatalf("legacy GET: %d", legacy.Code)
			}
			if rec := request("PUT", href, legacy.Body.String()); rec.Code != 201 {
				t.Fatalf("legacy TEXT roundtrip: %d", rec.Code)
			}
			if err := db.First(&saved, phone.ID).Error; err != nil || saved.Data != "tel:"+number || saved.Kind == nil || *saved.Kind != "cellphone" || !saved.Pref {
				t.Fatalf("legacy TEXT identity/metadata lost: %v", err)
			}
			if rec := request("PUT", href, card); rec.Code != 201 {
				t.Fatalf("legacy retry: %d", rec.Code)
			}
			if err := db.First(&saved, phone.ID).Error; err != nil || saved.Data != number || saved.Kind == nil || *saved.Kind != "cellphone" {
				t.Fatalf("legacy identity lost: %+v %v", saved, err)
			}
			before := request("GET", href, "")
			for _, invalid := range []string{"sip:person@example.test", "tel:", "tel:+1555;ext=", "tel:+1555;ext=%0A", "tel:+1555;ext=%ZZ"} {
				bad := strings.Replace(card, "tel:"+number, invalid, 1)
				bad = strings.Replace(bad, "N:Contact;URI;;;", "N:Contact;Rejected;;;\r\nORG:Rollback telephone company", 1)
				for _, target := range []string{href, book + "rejected-telephone.vcf"} {
					if rec := request("PUT", target, bad); rec.Code != 400 {
						t.Fatalf("invalid %q: %d %s", invalid, rec.Code, rec.Body.String())
					}
				}
				after := request("GET", href, "")
				if after.Code != 200 || after.Body.String() != before.Body.String() || after.Header().Get("ETag") != before.Header().Get("ETag") {
					t.Fatal("rejected URI changed existing resource")
				}
				if err := db.Model(&models.Company{}).Where("vault_id = ?", vault).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("company escaped rollback: %d %v", count, err)
				}
				if err := db.Model(&models.Contact{}).Where("vault_id = ?", vault).Count(&count).Error; err != nil || count != 2 {
					t.Fatalf("invalid create escaped rollback: %d %v", count, err)
				}
			}
			if rec := request("PUT", href, card); rec.Code != 201 {
				t.Fatalf("retry after rejection: %d", rec.Code)
			}
			if err := db.First(&saved, phone.ID).Error; err != nil || saved.Data != number {
				t.Fatalf("retry lost phone: %+v %v", saved, err)
			}
			var untouched models.Contact
			if err := db.First(&untouched, "id = ?", unrelated).Error; err != nil || untouched.FirstName == nil || *untouched.FirstName != "Unrelated" {
				t.Fatalf("unrelated contact changed: %+v %v", untouched, err)
			}
		})
	}
}
