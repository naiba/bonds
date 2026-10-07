package services

import (
	"errors"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardLegacyBirthdayClassificationSurvivesRestart(t *testing.T) {
	svc, vaultID, _, accountID := setupContactTest(t)
	db := svc.db
	// Reconstruct a pre-upgrade database without the completed conversion entry.
	if db.Migrator().HasTable("data_migrations") {
		if err := db.Exec("DELETE FROM data_migrations WHERE name = ?", "explicit_birthdate_types").Error; err != nil {
			t.Fatal(err)
		}
	}
	legacy := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Legacy birthday")}
	typed := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Explicit birthday")}
	for _, c := range []*models.Contact{&legacy, &typed} {
		if err := db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	var birthdayType models.ContactImportantDateType
	if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&birthdayType).Error; err != nil {
		t.Fatal(err)
	}
	year, month, day := 1980, 4, 6
	dates := []models.ContactImportantDate{
		{ContactID: legacy.ID, Label: "bIrThDaY", Year: &year, Month: &month, Day: &day, DatePrecision: "full"},
		{ContactID: legacy.ID, Label: "Birthdate", Year: &year, Month: &month, Day: &day, DatePrecision: "full"},
		{ContactID: typed.ID, Label: "Birthday", Year: &year, Month: &month, Day: &day, DatePrecision: "full"},
		{ContactID: typed.ID, Label: "Primary", ContactImportantDateTypeID: &birthdayType.ID, Year: &year, Month: &month, Day: &day, DatePrecision: "full"},
	}
	for i := range dates {
		if err := db.Create(&dates[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Compare database values before/after, including PostgreSQL's timestamp precision.
	for i := range dates {
		if err := db.First(&dates[i], dates[i].ID).Error; err != nil {
			t.Fatal(err)
		}
	}
	reminder := models.ContactReminder{ContactID: legacy.ID, Label: "Legacy reminder", Type: "recurring_year", ImportantDateID: &dates[0].ID, Month: &month, Day: &day}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	for i, original := range dates {
		var current models.ContactImportantDate
		if err := db.First(&current, original.ID).Error; err != nil {
			t.Fatal(err)
		}
		if i == 0 || i == 3 {
			if current.ContactImportantDateTypeID == nil || *current.ContactImportantDateTypeID != birthdayType.ID {
				t.Fatalf("expected explicit birthday: %+v", current)
			}
		} else if current.ContactImportantDateTypeID != nil {
			t.Fatalf("ordinary date promoted: %+v", current)
		}
		if current.Label != original.Label || current.Year == nil || *current.Year != year || !current.UpdatedAt.Equal(original.UpdatedAt) {
			t.Fatalf("migration changed legacy data: %+v", current)
		}
	}
	card, err := NewVCardService(db).ExportContactToVCard(legacy.ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if got := card.Value(vcard.FieldBirthday); got != "1980-04-06" {
		t.Fatalf("legacy birthday not exported: %q", got)
	}
	card.SetValue(vcard.FieldBirthday, "1981-04-06")
	if err := db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, legacy.ID, vaultID, accountID) }); err != nil {
		t.Fatal(err)
	}
	var edited models.ContactImportantDate
	if err := db.First(&edited, dates[0].ID).Error; err != nil || edited.Year == nil || *edited.Year != 1981 {
		t.Fatalf("legacy birthday edit failed: %+v %v", edited, err)
	}
	var retained models.ContactReminder
	if err := db.First(&retained, reminder.ID).Error; err != nil || retained.ImportantDateID == nil || *retained.ImportantDateID != edited.ID {
		t.Fatalf("legacy reminder lost: %+v %v", retained, err)
	}
	delete(card, vcard.FieldBirthday)
	if err := db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, legacy.ID, vaultID, accountID) }); err != nil {
		t.Fatal(err)
	}
	// Restart must not classify the remaining ordinary date as a new birthday.
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, legacy.ID, vaultID, accountID) }); err != nil {
		t.Fatal(err)
	}
	card, err = NewVCardService(db).ExportContactToVCard(legacy.ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if got := card.Value(vcard.FieldBirthday); got != "" {
		t.Fatalf("ordinary date promoted after restart: %q", got)
	}
	var ordinary models.ContactImportantDate
	if err := db.First(&ordinary, dates[1].ID).Error; err != nil || ordinary.ContactImportantDateTypeID != nil {
		t.Fatalf("ordinary legacy date lost: %+v %v", ordinary, err)
	}
	var removed int64
	if err := db.Model(&models.ContactReminder{}).Where("id = ?", reminder.ID).Count(&removed).Error; err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatal("deleted primary reminder retained")
	}
}

func TestVCardLegacyBirthdayMigrationRollsBackAndRetries(t *testing.T) {
	svc, vaultID, _, _ := setupContactTest(t)
	db := svc.db
	if db.Migrator().HasTable("data_migrations") {
		if err := db.Exec("DELETE FROM data_migrations WHERE name = ?", "explicit_birthdate_types").Error; err != nil {
			t.Fatal(err)
		}
	}
	contact := models.Contact{VaultID: vaultID}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatal(err)
	}
	date := models.ContactImportantDate{ContactID: contact.ID, Label: "Birthday"}
	if err := db.Create(&date).Error; err != nil {
		t.Fatal(err)
	}
	injected := errors.New("birthday classification write failed")
	callback := "reject_birthday_classification"
	if err := db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_important_dates" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	err := database.AutoMigrate(db)
	if removeErr := db.Callback().Update().Remove(callback); removeErr != nil {
		t.Fatal(removeErr)
	}
	if !errors.Is(err, injected) {
		t.Fatalf("migration did not reach injected write failure: %v", err)
	}
	var marked int64
	if err := db.Table("data_migrations").Where("name = ?", "explicit_birthdate_types").Count(&marked).Error; err != nil {
		t.Fatal(err)
	}
	if marked != 0 {
		t.Fatal("failed migration was marked complete")
	}
	var unchanged models.ContactImportantDate
	if err := db.First(&unchanged, date.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.ContactImportantDateTypeID != nil {
		t.Fatal("failed migration partially classified date")
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&unchanged, date.ID).Error; err != nil || unchanged.ContactImportantDateTypeID == nil {
		t.Fatalf("retry did not classify date: %+v %v", unchanged, err)
	}
}
