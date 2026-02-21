package dav

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupCalDAVTest(t *testing.T) (*CalDAVBackend, *gorm.DB, context.Context, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := services.NewAuthService(db, cfg)
	vaultSvc := services.NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "CalUser",
		Email:     "caldav-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Cal Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	backend := NewCalDAVBackend(db)
	ctx := WithUserID(context.Background(), resp.User.ID)
	ctx = WithAccountID(ctx, resp.User.AccountID)

	return backend, db, ctx, vault.ID, resp.User.ID
}

// Verify the CalDAVBackend implements the caldav.Backend interface at compile time.
var _ caldav.Backend = (*CalDAVBackend)(nil)

func TestListCalendars(t *testing.T) {
	backend, _, ctx, _, _ := setupCalDAVTest(t)

	cals, err := backend.ListCalendars(ctx)
	if err != nil {
		t.Fatalf("ListCalendars failed: %v", err)
	}
	if len(cals) != 1 {
		t.Fatalf("Expected 1 calendar, got %d", len(cals))
	}
	if cals[0].Name != "Cal Vault" {
		t.Errorf("Expected name 'Cal Vault', got '%s'", cals[0].Name)
	}
}

func TestListCalendarObjects(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	// Create a contact first
	contact := createTestContact(t, db, vaultID, userID, "John", "Doe")

	// Create important dates with UUIDs
	uid1 := uuid.New().String()
	uid2 := uuid.New().String()
	day1 := 15
	month1 := 3
	year1 := 1990
	day2 := 25
	month2 := 12

	date1 := models.ContactImportantDate{
		ContactID: contact.ID,
		UUID:      &uid1,
		Label:     "Birthday",
		Day:       &day1,
		Month:     &month1,
		Year:      &year1,
	}
	if err := db.Create(&date1).Error; err != nil {
		t.Fatalf("Create date1 failed: %v", err)
	}

	date2 := models.ContactImportantDate{
		ContactID: contact.ID,
		UUID:      &uid2,
		Label:     "Christmas",
		Day:       &day2,
		Month:     &month2,
	}
	if err := db.Create(&date2).Error; err != nil {
		t.Fatalf("Create date2 failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListCalendarObjects(ctx, path, nil)
	if err != nil {
		t.Fatalf("ListCalendarObjects failed: %v", err)
	}

	// Should have at least 2 important dates (plus possibly seed data dates)
	dateCount := 0
	for _, obj := range objects {
		if obj.Data == nil {
			t.Error("Expected non-nil Data")
			continue
		}
		for _, child := range obj.Data.Children {
			if child.Name == ical.CompEvent {
				dateCount++
			}
		}
	}
	if dateCount < 2 {
		t.Errorf("Expected at least 2 events, got %d (total objects: %d)", dateCount, len(objects))
	}

	// Verify iCal content
	foundBirthday := false
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for _, child := range obj.Data.Children {
			summary, _ := child.Props.Text(ical.PropSummary)
			if summary == "Birthday" {
				foundBirthday = true
				// Check RRULE exists (birthday has year set)
				rrule := child.Props.Get(ical.PropRecurrenceRule)
				if rrule == nil {
					t.Error("Expected RRULE for birthday event")
				}
			}
		}
	}
	if !foundBirthday {
		t.Error("Expected to find Birthday event")
	}
}

func TestGetCalendarObject(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Jane", "Doe")

	uid := uuid.New().String()
	day := 14
	month := 2
	year := 2000
	date := models.ContactImportantDate{
		ContactID: contact.ID,
		UUID:      &uid,
		Label:     "Anniversary",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}
	if err := db.Create(&date).Error; err != nil {
		t.Fatalf("Create date failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/" + uid + ".ics"
	obj, err := backend.GetCalendarObject(ctx, path, nil)
	if err != nil {
		t.Fatalf("GetCalendarObject failed: %v", err)
	}

	if obj.Data == nil {
		t.Fatal("Expected non-nil Data")
	}

	foundEvent := false
	for _, child := range obj.Data.Children {
		if child.Name == ical.CompEvent {
			foundEvent = true
			summary, _ := child.Props.Text(ical.PropSummary)
			if summary != "Anniversary" {
				t.Errorf("Expected summary 'Anniversary', got '%s'", summary)
			}
			gotUID, _ := child.Props.Text(ical.PropUID)
			if gotUID != uid {
				t.Errorf("Expected UID '%s', got '%s'", uid, gotUID)
			}
		}
	}
	if !foundEvent {
		t.Error("Expected VEVENT in calendar object")
	}
}

func TestGetCalendarObjectNotFound(t *testing.T) {
	backend, _, ctx, _, _ := setupCalDAVTest(t)

	path := "/dav/calendars/x/y/nonexistent.ics"
	_, err := backend.GetCalendarObject(ctx, path, nil)
	if err == nil {
		t.Error("Expected error for nonexistent calendar object")
	}
}

func TestListCalendarObjectsWithTasks(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Task", "User")

	uid := uuid.New().String()
	task := models.ContactTask{
		ContactID:  contact.ID,
		AuthorID:   &userID,
		UUID:       &uid,
		AuthorName: "Test",
		Label:      "Buy groceries",
		Completed:  false,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListCalendarObjects(ctx, path, nil)
	if err != nil {
		t.Fatalf("ListCalendarObjects failed: %v", err)
	}

	foundTodo := false
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for _, child := range obj.Data.Children {
			if child.Name == ical.CompToDo {
				summary, _ := child.Props.Text(ical.PropSummary)
				if summary == "Buy groceries" {
					foundTodo = true
					status, _ := child.Props.Text(ical.PropStatus)
					if status != "NEEDS-ACTION" {
						t.Errorf("Expected status 'NEEDS-ACTION', got '%s'", status)
					}
				}
			}
		}
	}
	if !foundTodo {
		t.Error("Expected to find VTODO 'Buy groceries'")
	}
}

func TestGetCalendarObjectTask(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Task", "Person")

	uid := uuid.New().String()
	desc := "A detailed description"
	dueAt := time.Now().Add(24 * time.Hour)
	task := models.ContactTask{
		ContactID:   contact.ID,
		AuthorID:    &userID,
		UUID:        &uid,
		AuthorName:  "Test",
		Label:       "Finish report",
		Description: &desc,
		DueAt:       &dueAt,
		Completed:   true,
		CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/" + uid + ".ics"
	obj, err := backend.GetCalendarObject(ctx, path, nil)
	if err != nil {
		t.Fatalf("GetCalendarObject failed: %v", err)
	}

	if obj.Data == nil {
		t.Fatal("Expected non-nil Data")
	}

	foundTodo := false
	for _, child := range obj.Data.Children {
		if child.Name == ical.CompToDo {
			foundTodo = true
			summary, _ := child.Props.Text(ical.PropSummary)
			if summary != "Finish report" {
				t.Errorf("Expected summary 'Finish report', got '%s'", summary)
			}
			status, _ := child.Props.Text(ical.PropStatus)
			if status != "COMPLETED" {
				t.Errorf("Expected status 'COMPLETED', got '%s'", status)
			}
			gotDesc, _ := child.Props.Text(ical.PropDescription)
			if gotDesc != desc {
				t.Errorf("Expected description '%s', got '%s'", desc, gotDesc)
			}
		}
	}
	if !foundTodo {
		t.Error("Expected VTODO in calendar object")
	}
}

func TestCalendarHomeSetPath(t *testing.T) {
	backend, _, ctx, _, userID := setupCalDAVTest(t)

	path, err := backend.CalendarHomeSetPath(ctx)
	if err != nil {
		t.Fatalf("CalendarHomeSetPath failed: %v", err)
	}
	expected := "/dav/calendars/" + userID + "/"
	if path != expected {
		t.Errorf("Expected '%s', got '%s'", expected, path)
	}
}

func TestListCalendarObjectsLunarDate(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Lunar", "Person")

	uid := uuid.New().String()
	day := 12
	month := 2
	year := 2025
	origDay := 15
	origMonth := 1
	origYear := 2025
	calType := "lunar"

	date := models.ContactImportantDate{
		ContactID:     contact.ID,
		UUID:          &uid,
		Label:         "Lunar Birthday",
		Day:           &day,
		Month:         &month,
		Year:          &year,
		CalendarType:  calType,
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	}
	if err := db.Create(&date).Error; err != nil {
		t.Fatalf("Create lunar date failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListCalendarObjects(ctx, path, nil)
	if err != nil {
		t.Fatalf("ListCalendarObjects failed: %v", err)
	}

	foundLunar := false
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for _, child := range obj.Data.Children {
			summary, _ := child.Props.Text(ical.PropSummary)
			if summary == "Lunar Birthday" {
				foundLunar = true
				desc, _ := child.Props.Text(ical.PropDescription)
				if desc == "" {
					t.Error("Expected DESCRIPTION for lunar calendar date")
				}
				if len(desc) > 0 && !(strings.Contains(desc, "lunar") || strings.Contains(desc, "Calendar")) {
					t.Errorf("Expected description to mention 'lunar' or 'Calendar', got '%s'", desc)
				}
			}
		}
	}
	if !foundLunar {
		t.Error("Expected to find 'Lunar Birthday' event")
	}
}

func TestCreateCalendarNotSupported(t *testing.T) {
	backend, _, ctx, _, _ := setupCalDAVTest(t)

	err := backend.CreateCalendar(ctx, &caldav.Calendar{})
	if err == nil {
		t.Error("Expected error for CreateCalendar")
	}
}
