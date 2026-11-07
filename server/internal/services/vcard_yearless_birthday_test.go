package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardYearlessBirthdayImportExport(t *testing.T) {
	for _, scenario := range []struct {
		name, property, canonical string
		year                      int
	}{
		{"apple", "BDAY;X-APPLE-OMIT-YEAR=1604:1604-02-27", "--0227", 0},
		{"apple_leap", "BDAY;X-APPLE-OMIT-YEAR=1604:16040229", "--0229", 0},
		{"standard", "BDAY:--0227", "--0227", 0},
		{"legacy", "BDAY:--02-27", "--0227", 0},
		{"real_1604", "BDAY:1604-02-27", "1604-02-27", 1604},
		{"known", "BDAY:1990-02-27", "1990-02-27", 1990},
		{"edited_known_year", "BDAY;X-APPLE-OMIT-YEAR=1604:1990-02-27", "1990-02-27", 1990},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			svc, _, vault, user := setupVCardTest(t)
			result, err := svc.ImportVCard(vault, user, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Birthday Example\r\n"+scenario.property+"\r\nEND:VCARD\r\n"))
			if err != nil || len(result.Contacts) != 1 {
				t.Fatalf("import: %+v %v", result, err)
			}
			id := result.Contacts[0].ID
			var date models.ContactImportantDate
			if err := svc.db.Where("contact_id = ?", id).First(&date).Error; err != nil {
				t.Fatal(err)
			}
			if (scenario.year == 0 && date.Year != nil) || (scenario.year != 0 && (date.Year == nil || *date.Year != scenario.year)) {
				t.Fatalf("persisted year: %v", date.Year)
			}
			if date.IsYearUnknown != (scenario.year == 0) || (scenario.year == 0 && date.DatePrecision != "month_day") || (scenario.year != 0 && date.DatePrecision != "full") {
				t.Fatalf("birthday precision: %+v", date)
			}
			if date.Month == nil || *date.Month != 2 || date.Day == nil {
				t.Fatalf("persisted birthday: %+v", date)
			}
			for _, bulk := range []bool{false, true} {
				var data []byte
				if bulk {
					data, err = svc.ExportVault(vault)
				} else {
					data, err = svc.ExportContact(id, vault)
				}
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), "BDAY:"+scenario.canonical+"\r\n") || strings.Contains(string(data), "X-APPLE-OMIT-YEAR") {
					t.Fatalf("canonical export: %s", data)
				}
			}
			contact, err := svc.loadContactForVCard(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			card := BuildContactCardDAVV3(contact)
			want := scenario.canonical
			marker := ""
			if scenario.year == 0 {
				want = fmt.Sprintf("1604-%02d-%02d", *date.Month, *date.Day)
				marker = "1604"
			}
			if card.Value(vcard.FieldVersion) != "3.0" || card.Value(vcard.FieldBirthday) != want || card.Get(vcard.FieldBirthday).Params.Get("X-APPLE-OMIT-YEAR") != marker {
				t.Fatalf("CardDAV: %#v", card.Get(vcard.FieldBirthday))
			}
		})
	}
}

func TestVCardYearlessBirthdayInvalidImportRollsBack(t *testing.T) {
	for _, property := range []string{
		"BDAY;X-APPLE-OMIT-YEAR=1604:1604-02-30",
		"BDAY;X-APPLE-OMIT-YEAR=unknown:1604-02-27",
		"BDAY;X-APPLE-OMIT-YEAR=:1604-02-27",
		"BDAY;X-APPLE-OMIT-YEAR=1604: ",
		"BDAY: \r\nBDAY:1604-02-27",
		"BDAY;X-APPLE-OMIT-YEAR=1604,1990:1604-02-27",
		"BDAY:--0227\r\nBDAY:1990-02-27",
	} {
		t.Run(property, func(t *testing.T) {
			svc, _, vault, user := setupVCardTest(t)
			var before int64
			if err := svc.db.Model(&models.Contact{}).Count(&before).Error; err != nil {
				t.Fatal(err)
			}
			_, err := svc.ImportVCard(vault, user, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Rejected Birthday\r\n"+property+"\r\nEND:VCARD\r\n"))
			if !errors.Is(err, ErrVCardInvalidData) {
				t.Fatalf("want invalid birthday, got %v", err)
			}
			var after int64
			if err := svc.db.Model(&models.Contact{}).Count(&after).Error; err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Fatalf("partial contact committed: %d -> %d", before, after)
			}
		})
	}
}

func TestVCardYearlessBirthdaySubscriptionCreate(t *testing.T) {
	svc, _, vault, user := setupVCardTest(t)
	var owner models.User
	if err := svc.db.First(&owner, "id = ?", user).Error; err != nil {
		t.Fatal(err)
	}
	card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Remote Birthday\r\nBDAY;X-APPLE-OMIT-YEAR=1604:1604-02-29\r\nEND:VCARD\r\n")).Decode()
	if err != nil {
		t.Fatal(err)
	}
	var id, action string
	err = svc.db.Transaction(func(tx *gorm.DB) error {
		var err error
		id, action, err = svc.UpsertContactFromVCard(tx, card, vault, user, owner.AccountID, "", "/remote/yearless.vcf", "initial", nil)
		return err
	})
	if err != nil || action != "created" || id == "" {
		t.Fatalf("subscription create: %s %s %v", id, action, err)
	}
	var date models.ContactImportantDate
	if err := svc.db.Where("contact_id = ?", id).First(&date).Error; err != nil {
		t.Fatal(err)
	}
	if date.Year != nil || date.Month == nil || *date.Month != 2 || date.Day == nil || *date.Day != 29 || !date.IsYearUnknown {
		t.Fatalf("persisted subscription birthday: %+v", date)
	}
	data, err := svc.ExportContact(id, vault)
	if err != nil || !strings.Contains(string(data), "BDAY:--0229\r\n") {
		t.Fatalf("subscription export: %s %v", data, err)
	}
}

func TestVCardImportPreservesUnsupportedUnmarkedBirthdayBehavior(t *testing.T) {
	svc, _, vault, user := setupVCardTest(t)
	// Text birthdays cannot be represented by ImportantDate. File import has
	// always kept the contact while omitting that optional, unrepresentable date.
	result, err := svc.ImportVCard(vault, user, strings.NewReader("BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Text Birthday\r\nN:Birthday;Text;;;\r\nBDAY;VALUE=text:Unknown\r\nEND:VCARD\r\n"))
	if err != nil || len(result.Contacts) != 1 {
		t.Fatalf("unmarked optional birthday blocked import: %+v %v", result, err)
	}
	var contact models.Contact
	if err := svc.db.First(&contact, "id = ?", result.Contacts[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if contact.FirstName == nil || *contact.FirstName != "Text" {
		t.Fatalf("contact not preserved: %+v", contact)
	}
	var count int64
	if err := svc.db.Model(&models.ContactImportantDate{}).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invented date for text birthday: %d", count)
	}
}
