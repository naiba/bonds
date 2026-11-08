package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
)

func TestVCardTelephoneURIImport(t *testing.T) {
	svc, _, vault, user := setupVCardTest(t)
	for _, tc := range []struct{ label, field, want string }{
		{"uri", "TEL;VALUE=uri;TYPE=cell;PREF=1:tel:+15550101010", "+15550101010"},
		{"parameters", "TEL;VALUE=uri;TYPE=cell;PREF=1:tel:+15550101010;ext=42", "+15550101010;ext=42"},
		{"local-context", "TEL;VALUE=URI;TYPE=cell;PREF=1:TEL:7042;phone-context=example.test", "7042;phone-context=example.test"},
		{"text", "TEL;VALUE=text;TYPE=cell;PREF=1:+1 (555) 010 1010", "+1 (555) 010 1010"},
		{"default-text", "TEL;TYPE=cell;PREF=1:+1 (555) 010 1010", "+1 (555) 010 1010"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			wire := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Imported Phone\r\nN:Phone;Imported;;;\r\n" + tc.field + "\r\nEND:VCARD\r\n"
			result, err := svc.ImportVCard(vault, user, strings.NewReader(wire))
			if err != nil || result.ImportedCount != 1 {
				t.Fatalf("import: %+v %v", result, err)
			}
			id := result.Contacts[0].ID
			var phone models.ContactInformation
			if err := svc.db.Where("contact_id = ?", id).First(&phone).Error; err != nil || phone.Data != tc.want || phone.Kind == nil || *phone.Kind != "mobile" || !phone.Pref {
				t.Fatalf("saved data=%q kind=%v pref=%v err=%v", phone.Data, phone.Kind, phone.Pref, err)
			}
			wireOut, err := svc.ExportContact(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			card, err := vcard.NewDecoder(strings.NewReader(string(wireOut))).Decode()
			if err != nil {
				t.Fatal(err)
			}
			if card.Value(vcard.FieldTelephone) != tc.want {
				t.Fatalf("export telephone=%q", card.Value(vcard.FieldTelephone))
			}
			again, err := svc.ImportVCard(vault, user, strings.NewReader(string(wireOut)))
			if err != nil || again.ImportedCount != 1 {
				t.Fatalf("reimport: %+v %v", again, err)
			}
			var reread models.ContactInformation
			if err := svc.db.Where("contact_id = ?", again.Contacts[0].ID).First(&reread).Error; err != nil || reread.Data != tc.want {
				t.Fatalf("parameter loss: %q %v", reread.Data, err)
			}
		})
	}
	var beforeTypes int64
	if err := svc.db.Model(&models.Contact{}).Where("vault_id = ?", vault).Count(&beforeTypes).Error; err != nil {
		t.Fatal(err)
	}
	// Ambiguous or unsupported value types cannot silently select a phone representation.
	for _, field := range []string{"TEL;VALUE=uri;VALUE=text:tel:+15550101010", "TEL;VALUE=:tel:+15550101010", "TEL;VALUE=binary:tel:+15550101010"} {
		wire := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Ambiguous Phone\r\n" + field + "\r\nEND:VCARD\r\n"
		if _, err := svc.ImportVCard(vault, user, strings.NewReader(wire)); !errors.Is(err, ErrVCardInvalidData) {
			t.Fatalf("ambiguous value type %q: %v", field, err)
		}
		var afterTypes int64
		if err := svc.db.Model(&models.Contact{}).Where("vault_id = ?", vault).Count(&afterTypes).Error; err != nil || afterTypes != beforeTypes {
			t.Fatalf("ambiguous input escaped rollback: %d %v", afterTypes, err)
		}
	}
	// Invalid data later in a multi-card import must roll back the whole batch.
	var before int64
	if err := svc.db.Model(&models.Contact{}).Where("vault_id = ?", vault).Count(&before).Error; err != nil {
		t.Fatal(err)
	}
	good := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Batch Phone\r\nTEL;VALUE=uri:tel:+15550101010\r\nEND:VCARD\r\n"
	for _, bad := range []string{"sip:person@example.test", "tel:", "tel:+1555;ext=%ZZ", "tel:+1555;ext=1;ext=2", "tel:7042"} {
		rejected := strings.Replace(good, "TEL;VALUE=uri:tel:+15550101010", "ORG:Rejected import company\r\nTEL;VALUE=uri:"+bad, 1)
		if _, err := svc.ImportVCard(vault, user, strings.NewReader(good+rejected)); !errors.Is(err, ErrVCardInvalidData) {
			t.Fatalf("%q: %v", bad, err)
		}
		var count int64
		if err := svc.db.Model(&models.Contact{}).Where("vault_id = ?", vault).Count(&count).Error; err != nil || count != before {
			t.Fatalf("batch rollback: %d want %d %v", count, before, err)
		}
		if err := svc.db.Model(&models.Company{}).Where("vault_id = ? AND name = ?", vault, "Rejected import company").Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("company rollback: %d %v", count, err)
		}
	}
	if result, err := svc.ImportVCard(vault, user, strings.NewReader(good)); err != nil || result.ImportedCount != 1 {
		t.Fatalf("retry: %+v %v", result, err)
	}
}
