package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type importantDateTestCtx struct {
	svc       *ImportantDateService
	contactID string
	vaultID   string
	db        *gorm.DB
}

func setupImportantDateTest(t *testing.T) importantDateTestCtx {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "important-date-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return importantDateTestCtx{
		svc:       NewImportantDateService(db),
		contactID: contact.ID,
		vaultID:   vault.ID,
		db:        db,
	}
}

func TestCreateImportantDate(t *testing.T) {
	ctx := setupImportantDateTest(t)

	day := 15
	month := 6
	year := 1990
	date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Label: "Birthday",
		Day:   &day,
		Month: &month,
		Year:  &year,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.Label != "Birthday" {
		t.Errorf("Expected label 'Birthday', got '%s'", date.Label)
	}
	if date.ContactID != ctx.contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", ctx.contactID, date.ContactID)
	}
	if date.Day == nil || *date.Day != 15 {
		t.Errorf("Expected day 15, got %v", date.Day)
	}
	if date.Month == nil || *date.Month != 6 {
		t.Errorf("Expected month 6, got %v", date.Month)
	}
	if date.Year == nil || *date.Year != 1990 {
		t.Errorf("Expected year 1990, got %v", date.Year)
	}
	if date.ID == 0 {
		t.Error("Expected important date ID to be non-zero")
	}
}

func TestListImportantDates(t *testing.T) {
	ctx := setupImportantDateTest(t)

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{Label: "Birthday"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{Label: "Anniversary"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dates, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(dates) != 2 {
		t.Errorf("Expected 2 important dates, got %d", len(dates))
	}
}

func TestUpdateImportantDate(t *testing.T) {
	ctx := setupImportantDateTest(t)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{Label: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	day := 25
	month := 12
	updated, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, dto.UpdateImportantDateRequest{
		Label: "Updated",
		Day:   &day,
		Month: &month,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Updated" {
		t.Errorf("Expected label 'Updated', got '%s'", updated.Label)
	}
	if updated.Day == nil || *updated.Day != 25 {
		t.Errorf("Expected day 25, got %v", updated.Day)
	}
	if updated.Month == nil || *updated.Month != 12 {
		t.Errorf("Expected month 12, got %v", updated.Month)
	}
}

func TestDeleteImportantDate(t *testing.T) {
	ctx := setupImportantDateTest(t)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{Label: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := ctx.svc.Delete(created.ID, ctx.contactID, ctx.vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	dates, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(dates) != 0 {
		t.Errorf("Expected 0 important dates after delete, got %d", len(dates))
	}
}

func TestDeleteImportantDateNotFound(t *testing.T) {
	ctx := setupImportantDateTest(t)

	err := ctx.svc.Delete(9999, ctx.contactID, ctx.vaultID)
	if err != ErrImportantDateNotFound {
		t.Errorf("Expected ErrImportantDateNotFound, got %v", err)
	}
}

func TestCreateImportantDateLunar(t *testing.T) {
	ctx := setupImportantDateTest(t)

	origDay := 15
	origMonth := 1
	origYear := 2025
	date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Label:         "Lunar New Year",
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.Label != "Lunar New Year" {
		t.Errorf("Expected label 'Lunar New Year', got '%s'", date.Label)
	}
	if date.CalendarType != "lunar" {
		t.Errorf("Expected calendar_type 'lunar', got '%s'", date.CalendarType)
	}
	if date.OriginalDay == nil || *date.OriginalDay != 15 {
		t.Errorf("Expected original_day 15, got %v", date.OriginalDay)
	}
	if date.OriginalMonth == nil || *date.OriginalMonth != 1 {
		t.Errorf("Expected original_month 1, got %v", date.OriginalMonth)
	}
	if date.OriginalYear == nil || *date.OriginalYear != 2025 {
		t.Errorf("Expected original_year 2025, got %v", date.OriginalYear)
	}
	if date.Day == nil || date.Month == nil || date.Year == nil {
		t.Fatal("Expected gregorian day/month/year to be set after conversion")
	}
	if *date.Year != 2025 {
		t.Errorf("Expected gregorian year 2025, got %d", *date.Year)
	}
	if *date.Month != 2 {
		t.Errorf("Expected gregorian month 2 (Feb), got %d", *date.Month)
	}
}

func TestUpdateImportantDateLunar(t *testing.T) {
	ctx := setupImportantDateTest(t)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{Label: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	origDay := 8
	origMonth := 8
	origYear := 2025
	updated, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, dto.UpdateImportantDateRequest{
		Label:         "Mid-Autumn",
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.CalendarType != "lunar" {
		t.Errorf("Expected calendar_type 'lunar', got '%s'", updated.CalendarType)
	}
	if updated.OriginalDay == nil || *updated.OriginalDay != 8 {
		t.Errorf("Expected original_day 8, got %v", updated.OriginalDay)
	}
	if updated.Day == nil || updated.Month == nil {
		t.Fatal("Expected gregorian day/month to be set")
	}
}

func TestCreateImportantDate_LabelAutoFilledFromType(t *testing.T) {
	ctx := setupImportantDateTest(t)

	var birthdateType models.ContactImportantDateType
	if err := ctx.db.Where("vault_id = ? AND label = ?", ctx.vaultID, "Birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("Failed to find Birthdate type: %v", err)
	}

	day := 1
	month := 1
	date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Day:                        &day,
		Month:                      &month,
		ContactImportantDateTypeID: &birthdateType.ID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.Label != "Birthdate" {
		t.Errorf("Expected label 'Birthdate' (auto-filled from type), got '%s'", date.Label)
	}
	if date.ContactImportantDateTypeID == nil || *date.ContactImportantDateTypeID != birthdateType.ID {
		t.Errorf("Expected type ID %d, got %v", birthdateType.ID, date.ContactImportantDateTypeID)
	}
}

func TestCreateImportantDate_LabelRequiredWithoutType(t *testing.T) {
	ctx := setupImportantDateTest(t)

	day := 1
	month := 1
	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Day:   &day,
		Month: &month,
	})
	if !errors.Is(err, ErrImportantDateLabelRequired) {
		t.Errorf("Expected ErrImportantDateLabelRequired, got %v", err)
	}
}

func TestCreateImportantDate_ExplicitLabelOverridesType(t *testing.T) {
	ctx := setupImportantDateTest(t)

	var birthdateType models.ContactImportantDateType
	if err := ctx.db.Where("vault_id = ? AND label = ?", ctx.vaultID, "Birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("Failed to find Birthdate type: %v", err)
	}

	day := 1
	month := 1
	date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Label:                      "My Custom Birthday",
		Day:                        &day,
		Month:                      &month,
		ContactImportantDateTypeID: &birthdateType.ID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.Label != "My Custom Birthday" {
		t.Errorf("Expected label 'My Custom Birthday', got '%s'", date.Label)
	}
}

func TestImportantDate_RemindMe(t *testing.T) {
	ctx := setupImportantDateTest(t)

	var birthdateType models.ContactImportantDateType
	if err := ctx.db.Where("vault_id = ? AND label = ?", ctx.vaultID, "Birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("Failed to find Birthdate type: %v", err)
	}

	day, month, year := 15, 6, 1990
	remindTrue := true
	date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Label:                      "Birthday",
		Day:                        &day,
		Month:                      &month,
		Year:                       &year,
		ContactImportantDateTypeID: &birthdateType.ID,
		RemindMe:                   &remindTrue,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !date.RemindMe {
		t.Error("Expected RemindMe to be true after create")
	}

	var reminder models.ContactReminder
	if err := ctx.db.Where("contact_id = ? AND important_date_id = ?", ctx.contactID, date.ID).First(&reminder).Error; err != nil {
		t.Fatalf("Expected reminder to exist: %v", err)
	}
	if reminder.Type != "recurring_year" {
		t.Errorf("Expected reminder type 'recurring_year', got '%s'", reminder.Type)
	}
	if reminder.Label != "Birthday" {
		t.Errorf("Expected reminder label 'Birthday', got '%s'", reminder.Label)
	}

	remindFalse := false
	updated, err := ctx.svc.Update(date.ID, ctx.contactID, ctx.vaultID, dto.UpdateImportantDateRequest{
		Label:                      "Birthday",
		Day:                        &day,
		Month:                      &month,
		Year:                       &year,
		ContactImportantDateTypeID: &birthdateType.ID,
		RemindMe:                   &remindFalse,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.RemindMe {
		t.Error("Expected RemindMe to be false after update")
	}

	var count int64
	ctx.db.Model(&models.ContactReminder{}).Where("contact_id = ? AND important_date_id = ?", ctx.contactID, date.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 reminders after disabling remind_me, got %d", count)
	}

	updated2, err := ctx.svc.Update(date.ID, ctx.contactID, ctx.vaultID, dto.UpdateImportantDateRequest{
		Label:                      "Birthday",
		Day:                        &day,
		Month:                      &month,
		Year:                       &year,
		ContactImportantDateTypeID: &birthdateType.ID,
		RemindMe:                   &remindTrue,
	})
	if err != nil {
		t.Fatalf("Update (re-enable) failed: %v", err)
	}
	if !updated2.RemindMe {
		t.Error("Expected RemindMe to be true after re-enable")
	}

	ctx.db.Model(&models.ContactReminder{}).Where("contact_id = ? AND important_date_id = ?", ctx.contactID, date.ID).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 reminder after re-enabling remind_me, got %d", count)
	}
}

func TestImportantDate_DeleteRemovesReminder(t *testing.T) {
	ctx := setupImportantDateTest(t)

	var birthdateType models.ContactImportantDateType
	if err := ctx.db.Where("vault_id = ? AND label = ?", ctx.vaultID, "Birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("Failed to find Birthdate type: %v", err)
	}

	day, month, year := 25, 12, 1985
	remindTrue := true
	date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Label:                      "Christmas Birthday",
		Day:                        &day,
		Month:                      &month,
		Year:                       &year,
		ContactImportantDateTypeID: &birthdateType.ID,
		RemindMe:                   &remindTrue,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var count int64
	ctx.db.Model(&models.ContactReminder{}).Where("contact_id = ? AND important_date_id = ?", ctx.contactID, date.ID).Count(&count)
	if count != 1 {
		t.Fatalf("Expected 1 reminder before delete, got %d", count)
	}

	if err := ctx.svc.Delete(date.ID, ctx.contactID, ctx.vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	ctx.db.Model(&models.ContactReminder{}).Where("contact_id = ? AND important_date_id = ?", ctx.contactID, date.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 reminders after deleting date, got %d", count)
	}
}
