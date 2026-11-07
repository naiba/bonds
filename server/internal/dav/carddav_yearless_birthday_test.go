package dav

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

func TestCardDAVYearlessBirthdayLifecycle(t *testing.T) {
	for _, entry := range []string{"carddav", "subscription"} {
		t.Run(entry, func(t *testing.T) {
			backend, db, ctx, vault, user := setupCardDAVTest(t)
			contact := createTestContact(t, db, vault, user, "Yearless", "Birthday")
			var kind models.ContactImportantDateType
			if err := db.Where("vault_id = ? AND internal_type = ?", vault, "birthdate").First(&kind).Error; err != nil {
				t.Fatal(err)
			}
			month, day := 2, 29
			birthday := models.ContactImportantDate{ContactID: contact.ID, Label: "Birthdate", ContactImportantDateTypeID: &kind.ID, Month: &month, Day: &day, DatePrecision: "month_day", IsYearUnknown: true}
			ordinary := models.ContactImportantDate{ContactID: contact.ID, Label: "Ordinary date", Month: &month, Day: &day}
			for _, date := range []*models.ContactImportantDate{&birthday, &ordinary} {
				if err := db.Create(date).Error; err != nil {
					t.Fatal(err)
				}
			}
			reminder := models.ContactReminder{ContactID: contact.ID, ImportantDateID: &birthday.ID, Label: "Birthday reminder", Type: "recurring_year", Month: &month, Day: &day}
			if err := db.Create(&reminder).Error; err != nil {
				t.Fatal(err)
			}
			path := "/dav/addressbooks/" + user + "/" + vault + "/" + contact.ID + ".vcf"
			read := func() vcard.Card {
				t.Helper()
				object, err := backend.GetAddressObject(ctx, path, &carddav.AddressDataRequest{AllProp: true})
				if err != nil {
					t.Fatal(err)
				}
				// Exercise the wire codec as clients do; parameters must survive encoding.
				var wire bytes.Buffer
				if err := vcard.NewEncoder(&wire).Encode(object.Card); err != nil {
					t.Fatal(err)
				}
				card, err := vcard.NewDecoder(&wire).Decode()
				if err != nil {
					t.Fatal(err)
				}
				return card
			}
			revision := 0
			write := func(card vcard.Card) error {
				revision++
				if entry == "carddav" {
					_, err := backend.PutAddressObject(ctx, path, card, nil)
					return err
				}
				return db.Transaction(func(tx *gorm.DB) error {
					_, _, err := services.NewVCardService(tx).UpsertContactFromVCard(tx, card, vault, user, AccountIDFromContext(ctx), contact.ID, "/remote/birthday.vcf", fmt.Sprint(revision), nil)
					return err
				})
			}
			assertDate := func(wantDay int) {
				t.Helper()
				var got models.ContactImportantDate
				if err := db.First(&got, birthday.ID).Error; err != nil {
					t.Fatal(err)
				}
				if got.Year != nil || got.Month == nil || *got.Month != 2 || got.Day == nil || *got.Day != wantDay || !got.IsYearUnknown {
					t.Fatalf("birthday changed: %+v", got)
				}
				var retained models.ContactImportantDate
				if err := db.First(&retained, ordinary.ID).Error; err != nil || retained.Label != ordinary.Label || retained.Day == nil || *retained.Day != 29 || retained.ContactImportantDateTypeID != nil {
					t.Fatalf("ordinary date changed: %+v %v", retained, err)
				}
				var linked models.ContactReminder
				if err := db.First(&linked, reminder.ID).Error; err != nil || linked.Year != nil || linked.Day == nil || *linked.Day != wantDay || linked.ImportantDateID == nil || *linked.ImportantDateID != birthday.ID {
					t.Fatalf("reminder changed: %+v %v", linked, err)
				}
			}
			for i := 0; i < 2; i++ {
				card := read()
				if card.Value(vcard.FieldVersion) != "3.0" || card.Value(vcard.FieldBirthday) != "1604-02-29" || card.Get(vcard.FieldBirthday).Params.Get("X-APPLE-OMIT-YEAR") != "1604" {
					t.Fatalf("invalid 3.0 birthday: %+v", card.Get(vcard.FieldBirthday))
				}
				if err := write(card); err != nil {
					t.Fatal(err)
				}
				assertDate(29)
			}
			card := read()
			card.SetValue(vcard.FieldNickname, "Must roll back")
			card.Get(vcard.FieldBirthday).Value = "1604-02-30" // an invalid date must not partially update the contact
			if err := write(card); err == nil {
				t.Fatal("ambiguous birthday accepted")
			}
			assertDate(29)
			var unchanged models.Contact
			if err := db.First(&unchanged, "id = ?", contact.ID).Error; err != nil {
				t.Fatal(err)
			}
			if unchanged.Nickname != nil {
				t.Fatalf("nickname partially committed: %s", *unchanged.Nickname)
			}
			card = read()
			card.Get(vcard.FieldBirthday).Value = "1604-02-27"
			if err := write(card); err != nil {
				t.Fatal(err)
			}
			assertDate(27)
			if err := write(read()); err != nil {
				t.Fatal(err)
			}
			assertDate(27)
			canonical, err := services.NewVCardService(db).ExportContact(contact.ID, vault)
			if err != nil || !strings.Contains(string(canonical), "BDAY:--0227\r\n") {
				t.Fatalf("export: %s %v", canonical, err)
			}
			// A client can enter a real year without removing its old parameter.
			card = read()
			card.Get(vcard.FieldBirthday).Value = "1990-02-27"
			if err := write(card); err != nil {
				t.Fatal(err)
			}
			var known models.ContactImportantDate
			if err := db.First(&known, birthday.ID).Error; err != nil {
				t.Fatal(err)
			}
			if known.Year == nil || *known.Year != 1990 || known.IsYearUnknown || known.DatePrecision != "full" {
				t.Fatalf("explicit year lost: %+v", known)
			}
			card = read()
			if card.Get(vcard.FieldBirthday).Params.Get("X-APPLE-OMIT-YEAR") != "" {
				t.Fatal("known year exported with omit-year marker")
			}
			card.Get(vcard.FieldBirthday).Value = "--0227"
			if err := write(card); err != nil {
				t.Fatal(err)
			}
			assertDate(27)
			card = read()
			delete(card, vcard.FieldBirthday)
			for i := 0; i < 2; i++ {
				if err := write(card); err != nil {
					t.Fatal(err)
				}
			}
			var dates []models.ContactImportantDate
			if err := db.Where("contact_id = ?", contact.ID).Find(&dates).Error; err != nil {
				t.Fatal(err)
			}
			if len(dates) != 1 || dates[0].ID != ordinary.ID {
				t.Fatalf("deletion affected unrelated date: %+v", dates)
			}
		})
	}
}

func TestCardDAVYearlessBirthdayHTTPCreate(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, _ := createDAVHTTPTestContact(t, db, user, "Existing", "Contact")
	path := "/dav/addressbooks/" + user + "/" + vault + "/new-birthday.vcf"
	request := func(method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	body := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:New Birthday\r\nN:Birthday;New;;;\r\nBDAY;X-APPLE-OMIT-YEAR=1604:1604-02-30\r\nEND:VCARD\r\n"
	var before int64
	if err := db.Model(&models.Contact{}).Count(&before).Error; err != nil {
		t.Fatal(err)
	}
	rec := request("PUT", path, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid create: %d %s", rec.Code, rec.Body.String())
	}
	var after int64
	if err := db.Model(&models.Contact{}).Count(&after).Error; err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("rejected create committed contact: %d -> %d", before, after)
	}
	rec = request("PUT", path, strings.Replace(body, "02-30", "02-27", 1))
	if rec.Code != http.StatusCreated {
		t.Fatalf("valid retry: %d %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("missing created object location")
	}
	rec = request("GET", location, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/vcard") {
		t.Fatalf("GET: %d %s", rec.Code, rec.Body.String())
	}
	card, err := vcard.NewDecoder(strings.NewReader(rec.Body.String())).Decode()
	if err != nil {
		t.Fatal(err)
	}
	if card.Value(vcard.FieldVersion) != "3.0" || card.Value(vcard.FieldBirthday) != "1604-02-27" || card.Get(vcard.FieldBirthday).Params.Get("X-APPLE-OMIT-YEAR") != "1604" {
		t.Fatalf("GET birthday: %s", rec.Body.String())
	}
	var date models.ContactImportantDate
	if err := db.Where("contact_id = ?", card.Value(vcard.FieldUID)).First(&date).Error; err != nil {
		t.Fatal(err)
	}
	if date.Year != nil || date.Month == nil || *date.Month != 2 || date.Day == nil || *date.Day != 27 {
		t.Fatalf("stored birthday: %+v", date)
	}
}
