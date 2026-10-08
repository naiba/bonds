package services

import (
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardUpdateOrdinaryEditsPreserveUnrelatedRecords(t *testing.T) {
	for _, scenario := range []string{"add_birthday", "edit_birthday", "remove_birthday", "birthday_wire_format", "edit_email_with_mobile", "reorder_phone_parameters", "add_address_with_history"} {
		t.Run(scenario, func(t *testing.T) {
			svc, vaultID, _, accountID := setupContactTest(t)
			db := svc.db
			contact := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Synthetic")}
			if err := db.Create(&contact).Error; err != nil {
				t.Fatal(err)
			}
			year, month, day := 1990, 3, 2
			graduation := models.ContactImportantDate{ContactID: contact.ID, Label: "Graduation", Year: &year, Month: &month, Day: &day}
			var birthdayType models.ContactImportantDateType
			if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&birthdayType).Error; err != nil {
				t.Fatal(err)
			}
			birthday := models.ContactImportantDate{ContactID: contact.ID, Label: "Birthdate", ContactImportantDateTypeID: &birthdayType.ID, Year: &year, Month: &month, Day: &day, DatePrecision: "full"}
			var phoneType, emailType models.ContactInformationType
			if err := db.Where("account_id = ? AND type = ?", accountID, "phone").First(&phoneType).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("account_id = ? AND type = ?", accountID, "email").First(&emailType).Error; err != nil {
				t.Fatal(err)
			}
			phone := models.ContactInformation{ContactID: contact.ID, TypeID: phoneType.ID, Data: "+15550101010", Kind: strPtrOrNil("mobile"), Pref: true}
			email := models.ContactInformation{ContactID: contact.ID, TypeID: emailType.ID, Data: "before@example.test"}
			address := models.Address{VaultID: vaultID, Line1: strPtrOrNil("Garden street"), Line2: strPtrOrNil("Apartment A")}
			for _, row := range []any{&graduation, &phone, &email, &address} {
				if err := db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			moved := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			pivot := models.ContactAddress{ContactID: contact.ID, AddressID: address.ID, DateFrom: &moved}
			if err := db.Create(&pivot).Error; err != nil {
				t.Fatal(err)
			}
			if scenario != "add_birthday" {
				if err := db.Create(&birthday).Error; err != nil {
					t.Fatal(err)
				}
			}
			card, err := NewVCardService(db).ExportContactToVCard(contact.ID, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "add_birthday", "edit_birthday":
				card.SetValue(vcard.FieldBirthday, "2000-04-05")
			case "remove_birthday":
				delete(card, vcard.FieldBirthday)
			case "birthday_wire_format":
				card.SetValue(vcard.FieldBirthday, "19900302")
			case "edit_email_with_mobile":
				card.SetValue(vcard.FieldEmail, "after@example.test")
			case "reorder_phone_parameters":
				card[vcard.FieldTelephone][0].Params[vcard.ParamType] = []string{"VOICE", "CELL"}
			case "add_address_with_history":
				card.AddAddress(&vcard.Address{StreetAddress: "Second street"})
			}
			err = db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, contact.ID, vaultID, accountID) })
			if err != nil {
				t.Fatalf("ordinary DAV edit rejected: %v", err)
			}
			if err := db.First(&graduation, graduation.ID).Error; err != nil || graduation.Label != "Graduation" || graduation.Day == nil || *graduation.Day != 2 {
				t.Fatalf("unrelated date lost: %+v, %v", graduation, err)
			}
			if err := db.First(&phone, phone.ID).Error; err != nil || ptrToStr(phone.Kind) != "mobile" || !phone.Pref {
				t.Fatalf("unchanged mobile lost: %+v, %v", phone, err)
			}
			if err := db.First(&pivot, pivot.ID).Error; err != nil || pivot.DateFrom == nil || !pivot.DateFrom.Equal(moved) {
				t.Fatalf("address history lost: %+v, %v", pivot, err)
			}
			if err := db.First(&address, address.ID).Error; err != nil || ptrToStr(address.Line2) != "Apartment A" {
				t.Fatalf("address detail lost: %+v, %v", address, err)
			}
			switch scenario {
			case "add_birthday", "edit_birthday", "birthday_wire_format":
				var stored models.ContactImportantDate
				if err := db.Where("contact_id = ? AND contact_important_date_type_id = ?", contact.ID, birthdayType.ID).First(&stored).Error; err != nil {
					t.Fatal(err)
				}
				wantYear, wantDay := 2000, 5
				if scenario == "birthday_wire_format" {
					wantYear, wantDay = 1990, 2
				}
				if stored.Year == nil || *stored.Year != wantYear || stored.Day == nil || *stored.Day != wantDay || (birthday.ID != 0 && stored.ID != birthday.ID) {
					t.Fatalf("birthday edit not persisted in place: %+v", stored)
				}
			case "remove_birthday":
				var count int64
				if err := db.Model(&models.ContactImportantDate{}).Where("id = ?", birthday.ID).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("birthday not removed: %d, %v", count, err)
				}
			case "edit_email_with_mobile":
				var stored []models.ContactInformation
				if err := db.Where("contact_id = ? AND type_id = ?", contact.ID, emailType.ID).Find(&stored).Error; err != nil || len(stored) != 1 || stored[0].Data != "after@example.test" {
					t.Fatalf("email edit missing: %+v, %v", stored, err)
				}
			case "add_address_with_history":
				var count int64
				if err := db.Model(&models.ContactAddress{}).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil || count != 2 {
					t.Fatalf("new address missing: %d, %v", count, err)
				}
			}
		})
	}
}

func TestVCardBirthdayUpdateReschedulesLinkedReminders(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)
	db := svc.db
	contact := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Synthetic")}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	var kind models.ContactImportantDateType
	if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	year, month, day := 1990, 3, 2
	birthday := models.ContactImportantDate{ContactID: contact.ID, Label: "Birthday", ContactImportantDateTypeID: &kind.ID, Year: &year, Month: &month, Day: &day, RemindMe: true}
	if err := db.Create(&birthday).Error; err != nil {
		t.Fatal(err)
	}
	reminder := models.ContactReminder{ContactID: contact.ID, ImportantDateID: &birthday.ID, Label: "Remember birthday", Type: "recurring_year", Month: &month, Day: &day}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}
	var channel models.UserNotificationChannel
	if err := db.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		t.Fatal(err)
	}
	scheduled := models.ContactReminderScheduled{ContactReminderID: reminder.ID, UserNotificationChannelID: channel.ID, ScheduledAt: time.Date(2030, 3, 2, 9, 0, 0, 0, time.UTC)}
	if err := db.Create(&scheduled).Error; err != nil {
		t.Fatal(err)
	}
	deliveredAt := time.Date(2020, 3, 2, 9, 0, 0, 0, time.UTC)
	delivered := models.ContactReminderScheduled{ContactReminderID: reminder.ID, UserNotificationChannelID: channel.ID, ScheduledAt: deliveredAt, TriggeredAt: &deliveredAt}
	if err := db.Create(&delivered).Error; err != nil {
		t.Fatal(err)
	}
	card := vcard.Card{}
	card.SetValue(vcard.FieldBirthday, "--0405")
	if err := db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, contact.ID, vaultID, accountID) }); err != nil {
		t.Fatal(err)
	}
	var saved models.ContactReminder
	if err := db.First(&saved, reminder.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.ImportantDateID == nil || *saved.ImportantDateID != birthday.ID || saved.Day == nil || *saved.Day != 5 || saved.Month == nil || *saved.Month != 4 || saved.Year != nil || saved.Label != "Remember birthday" {
		t.Fatalf("linked reminder changed incorrectly: %+v", saved)
	}
	var pending []models.ContactReminderScheduled
	if err := db.Where("contact_reminder_id = ? AND triggered_at IS NULL", reminder.ID).Find(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID == scheduled.ID || pending[0].ScheduledAt.Month() != time.April || pending[0].ScheduledAt.Day() != 5 {
		t.Fatalf("pending schedule did not follow edited birthday: %+v", pending)
	}
	if err := db.First(&delivered, delivered.ID).Error; err != nil || delivered.TriggeredAt == nil || !delivered.TriggeredAt.Equal(deliveredAt) {
		t.Fatalf("delivery history lost: %v", err)
	}
	exported, err := NewVCardService(db).ExportContactToVCard(contact.ID, vaultID)
	if err != nil || exported.Value(vcard.FieldBirthday) != "--04-05" {
		t.Fatalf("yearless birthday roundtrip: %v, %v", exported, err)
	}
}

func TestVCardUpdatePreservesUnprojectedBirthday(t *testing.T) {
	svc, vaultID, _, accountID := setupContactTest(t)
	contact := models.Contact{VaultID: vaultID}
	if err := svc.db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	var kind models.ContactImportantDateType
	if err := svc.db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	year := 1990
	birthday := models.ContactImportantDate{ContactID: contact.ID, Label: "Birthday", ContactImportantDateTypeID: &kind.ID, Year: &year, DatePrecision: "year"}
	if err := svc.db.Create(&birthday).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Transaction(func(tx *gorm.DB) error {
		return ReplaceContactVCardFields(tx, vcard.Card{}, contact.ID, vaultID, accountID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.First(&birthday, birthday.ID).Error; err != nil || birthday.Year == nil || *birthday.Year != year || birthday.ContactImportantDateTypeID == nil || *birthday.ContactImportantDateTypeID != kind.ID || birthday.DatePrecision != "year" || birthday.ContactID != contact.ID {
		t.Fatalf("unprojected birthday deleted: %v", err)
	}
}

func TestVCardInformationEditsAndDeletions(t *testing.T) {
	for _, version := range []string{"3.0", "4.0"} {
		t.Run(version, func(t *testing.T) {
			svc, vaultID, _, accountID := setupContactTest(t)
			contact := models.Contact{VaultID: vaultID}
			if err := svc.db.Create(&contact).Error; err != nil {
				t.Fatal(err)
			}
			card := vcard.Card{}
			card.SetValue(vcard.FieldVersion, version)
			card.Add(vcard.FieldTelephone, &vcard.Field{Value: "+15550101010", Params: vcard.Params{vcard.ParamType: []string{"CELL", "VOICE", "PREF"}}})
			card.Add(vcard.FieldTelephone, &vcard.Field{Value: "+15550101011", Params: vcard.Params{vcard.ParamType: []string{"WORK", "FAX"}}})
			apply := func() {
				t.Helper()
				if err := svc.db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, contact.ID, vaultID, accountID) }); err != nil {
					t.Fatal(err)
				}
			}
			apply()
			var phones []models.ContactInformation
			if err := svc.db.Where("contact_id = ?", contact.ID).Order("id").Find(&phones).Error; err != nil {
				t.Fatal(err)
			}
			if len(phones) != 2 || ptrToStr(phones[0].Kind) != "mobile" || !phones[0].Pref || ptrToStr(phones[1].Kind) != "work fax" || phones[1].Pref {
				t.Fatalf("metadata not imported: %+v", phones)
			}
			card[vcard.FieldTelephone][0].Value = "+15550101999"
			card[vcard.FieldTelephone] = card[vcard.FieldTelephone][:1]
			apply()
			var remaining []models.ContactInformation
			if err := svc.db.Where("contact_id = ?", contact.ID).Find(&remaining).Error; err != nil || len(remaining) != 1 || remaining[0].Data != "+15550101999" || ptrToStr(remaining[0].Kind) != "mobile" || !remaining[0].Pref {
				t.Fatalf("phone edit/removal failed: %+v, %v", remaining, err)
			}
		})
	}
}

func TestVCardAddressRemovalOnlyDetachesSharedContact(t *testing.T) {
	svc, vaultID, _, accountID := setupContactTest(t)
	contact, neighbor := models.Contact{VaultID: vaultID}, models.Contact{VaultID: vaultID}
	address := models.Address{VaultID: vaultID, Line1: strPtrOrNil("Shared street")}
	for _, row := range []any{&contact, &neighbor, &address} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{contact.ID, neighbor.ID} {
		if err := svc.db.Create(&models.ContactAddress{ContactID: id, AddressID: address.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.db.Transaction(func(tx *gorm.DB) error {
		return ReplaceContactVCardFields(tx, vcard.Card{}, contact.ID, vaultID, accountID)
	}); err != nil {
		t.Fatal(err)
	}
	var links []models.ContactAddress
	if err := svc.db.Where("address_id = ?", address.ID).Find(&links).Error; err != nil || len(links) != 1 || links[0].ContactID != neighbor.ID {
		t.Fatalf("shared address ownership changed: %+v, %v", links, err)
	}
	if err := svc.db.First(&address, address.ID).Error; err != nil || ptrToStr(address.Line1) != "Shared street" {
		t.Fatalf("neighbor's address lost: %v", err)
	}
}
