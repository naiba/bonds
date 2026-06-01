package dav

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// CalDAVBackend implements the caldav.Backend interface.
type CalDAVBackend struct {
	db *gorm.DB
}

// NewCalDAVBackend creates a new CalDAV backend.
func NewCalDAVBackend(db *gorm.DB) *CalDAVBackend {
	return &CalDAVBackend{db: db}
}

func (b *CalDAVBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return "", fmt.Errorf("no user in context")
	}
	return "/dav/principals/" + userID + "/", nil
}

func (b *CalDAVBackend) CalendarHomeSetPath(ctx context.Context) (string, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return "", fmt.Errorf("no user in context")
	}
	return "/dav/calendars/" + userID + "/", nil
}

func (b *CalDAVBackend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	var userVaults []models.UserVault
	if err := b.db.Where("user_id = ?", userID).Find(&userVaults).Error; err != nil {
		return nil, err
	}

	var calendars []caldav.Calendar
	for _, uv := range userVaults {
		var vault models.Vault
		if err := b.db.First(&vault, "id = ?", uv.VaultID).Error; err != nil {
			continue
		}
		calendars = append(calendars, caldav.Calendar{
			Path:                  "/dav/calendars/" + userID + "/" + vault.ID + "/",
			Name:                  vault.Name,
			Description:           ptrToStr(vault.Description),
			SupportedComponentSet: []string{ical.CompEvent, ical.CompToDo},
		})
	}
	return calendars, nil
}

func (b *CalDAVBackend) GetCalendar(ctx context.Context, path string) (*caldav.Calendar, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	vaultID := extractVaultIDFromPath(path, "calendars", userID)
	if vaultID == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}

	if err := b.verifyVaultAccess(userID, vaultID); err != nil {
		return nil, err
	}

	var vault models.Vault
	if err := b.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}

	return &caldav.Calendar{
		Path:                  "/dav/calendars/" + userID + "/" + vault.ID + "/",
		Name:                  vault.Name,
		Description:           ptrToStr(vault.Description),
		SupportedComponentSet: []string{ical.CompEvent, ical.CompToDo},
	}, nil
}

func (b *CalDAVBackend) CreateCalendar(_ context.Context, _ *caldav.Calendar) error {
	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("creating calendars is not supported"))
}

func (b *CalDAVBackend) GetCalendarObject(ctx context.Context, path string, _ *caldav.CalendarCompRequest) (*caldav.CalendarObject, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	objectID := extractObjectIDFromPath(path, ".ics")
	if objectID == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar object not found"))
	}

	// Try important dates first
	var importantDate models.ContactImportantDate
	if err := b.db.Preload("Contact").First(&importantDate, "uuid = ?", objectID).Error; err == nil {
		if err := b.verifyVaultAccess(userID, importantDate.Contact.VaultID); err != nil {
			return nil, err
		}
		return importantDateToCalendarObject(&importantDate, userID), nil
	}

	// Try tasks
	var task models.ContactTask
	if err := b.db.First(&task, "uuid = ?", objectID).Error; err == nil {
		if err := b.verifyVaultAccess(userID, task.VaultID); err != nil {
			return nil, err
		}
		return taskToCalendarObject(&task, userID), nil
	}

	return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar object not found"))
}

func (b *CalDAVBackend) ListCalendarObjects(ctx context.Context, path string, _ *caldav.CalendarCompRequest) ([]caldav.CalendarObject, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	vaultID := extractVaultIDFromPath(path, "calendars", userID)
	if vaultID == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}

	if err := b.verifyVaultAccess(userID, vaultID); err != nil {
		return nil, err
	}

	// Get contacts in this vault
	var contacts []models.Contact
	if err := b.db.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		return nil, err
	}

	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
	}

	var objects []caldav.CalendarObject

	if len(contactIDs) > 0 {
		var dates []models.ContactImportantDate
		if err := b.db.Preload("Contact").Where("contact_id IN ?", contactIDs).Find(&dates).Error; err != nil {
			return nil, err
		}

		for i := range dates {
			if dates[i].UUID == nil || *dates[i].UUID == "" {
				uid := uuid.New().String()
				dates[i].UUID = &uid
				b.db.Model(&dates[i]).Update("uuid", uid)
			}
			objects = append(objects, *importantDateToCalendarObject(&dates[i], userID))
		}
	}

	// Tasks are vault-scoped and may have zero assignees (standalone), so
	// list them independently of whether the vault has any contacts.
	tasks, err := b.listVaultTasksForCalendar(vaultID, contactIDs)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		if tasks[i].UUID == nil || *tasks[i].UUID == "" {
			uid := uuid.New().String()
			tasks[i].UUID = &uid
			b.db.Model(&tasks[i]).Update("uuid", uid)
		}
		objects = append(objects, *taskToCalendarObject(&tasks[i], userID))
	}

	return objects, nil
}

// listVaultTasksForCalendar returns every task in the vault that the
// requesting user can see in their calendar feed: any task assigned to one
// of the vault's contacts, plus every standalone (zero-assignee) task. The
// LEFT JOIN with NOT EXISTS keeps standalone tasks regardless of whether
// the contactIDs slice is empty.
func (b *CalDAVBackend) listVaultTasksForCalendar(vaultID string, contactIDs []string) ([]models.ContactTask, error) {
	var tasks []models.ContactTask
	q := b.db.Model(&models.ContactTask{}).
		Distinct().
		Where("contact_tasks.vault_id = ?", vaultID)

	standalone := `NOT EXISTS (
		SELECT 1 FROM task_contacts tc WHERE tc.contact_task_id = contact_tasks.id
	)`
	if len(contactIDs) == 0 {
		q = q.Where(standalone)
	} else {
		assigned := `EXISTS (
			SELECT 1 FROM task_contacts tc
			WHERE tc.contact_task_id = contact_tasks.id AND tc.contact_id IN ?
		)`
		q = q.Where("("+assigned+") OR ("+standalone+")", contactIDs)
	}
	if err := q.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (b *CalDAVBackend) QueryCalendarObjects(ctx context.Context, path string, _ *caldav.CalendarQuery) ([]caldav.CalendarObject, error) {
	// Return all objects — filtering by query is optional
	return b.ListCalendarObjects(ctx, path, nil)
}

func (b *CalDAVBackend) PutCalendarObject(ctx context.Context, path string, calendar *ical.Calendar, _ *caldav.PutCalendarObjectOptions) (*caldav.CalendarObject, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	vaultID := extractVaultIDFromCalendarObjectPath(path, userID)
	if vaultID == "" {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("invalid path"))
	}

	if err := b.verifyVaultAccess(userID, vaultID); err != nil {
		return nil, err
	}

	// Parse the calendar to determine event type
	for _, child := range calendar.Children {
		switch child.Name {
		case ical.CompEvent:
			return b.putEvent(ctx, path, child, vaultID, userID)
		case ical.CompToDo:
			return b.putTodo(ctx, path, child, vaultID, userID)
		}
	}

	return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("no VEVENT or VTODO component found"))
}

func (b *CalDAVBackend) putEvent(_ context.Context, path string, comp *ical.Component, vaultID, userID string) (*caldav.CalendarObject, error) {
	uid, _ := comp.Props.Text(ical.PropUID)
	summary, _ := comp.Props.Text(ical.PropSummary)

	if uid == "" {
		uid = uuid.New().String()
	}

	now := time.Now()

	// Parse date
	var day, month, year *int
	if dtStart := comp.Props.Get(ical.PropDateTimeStart); dtStart != nil {
		dt, err := dtStart.DateTime(time.UTC)
		if err == nil {
			d := dt.Day()
			m := int(dt.Month())
			y := dt.Year()
			day = &d
			month = &m
			year = &y
		}
	}

	// Try to find existing
	var existing models.ContactImportantDate
	err := b.db.Preload("Contact").Where("uuid = ?", uid).First(&existing).Error
	if err == nil {
		// A globally unique event UID still has to belong to the requested DAV collection vault.
		if existing.Contact.VaultID != vaultID {
			return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar object not found"))
		}
		// Update
		existing.Label = summary
		existing.Day = day
		existing.Month = month
		existing.Year = year
		if err := b.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &caldav.CalendarObject{
			Path:    path,
			ModTime: existing.UpdatedAt,
			ETag:    fmt.Sprintf("%d", existing.UpdatedAt.Unix()),
			Data:    buildCalendarFromImportantDate(&existing),
		}, nil
	}

	// Need a contact - find first in vault
	var contact models.Contact
	if err := b.db.Where("vault_id = ?", vaultID).First(&contact).Error; err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("no contacts in vault"))
	}

	importantDate := models.ContactImportantDate{
		ContactID: contact.ID,
		UUID:      &uid,
		Label:     summary,
		Day:       day,
		Month:     month,
		Year:      year,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := b.db.Create(&importantDate).Error; err != nil {
		return nil, err
	}

	return &caldav.CalendarObject{
		Path:    path,
		ModTime: importantDate.UpdatedAt,
		ETag:    fmt.Sprintf("%d", importantDate.UpdatedAt.Unix()),
		Data:    buildCalendarFromImportantDate(&importantDate),
	}, nil
}

func (b *CalDAVBackend) putTodo(_ context.Context, path string, comp *ical.Component, vaultID, userID string) (*caldav.CalendarObject, error) {
	uid, _ := comp.Props.Text(ical.PropUID)
	summary, _ := comp.Props.Text(ical.PropSummary)
	description, _ := comp.Props.Text(ical.PropDescription)

	if uid == "" {
		uid = uuid.New().String()
	}

	now := time.Now()

	// Try to find existing
	var existing models.ContactTask
	err := b.db.Where("uuid = ?", uid).First(&existing).Error
	if err == nil {
		// A globally unique todo UID still has to belong to the requested DAV collection vault.
		if existing.VaultID != vaultID {
			return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar object not found"))
		}
		// Update
		existing.Label = summary
		if description != "" {
			existing.Description = &description
		}
		if err := b.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &caldav.CalendarObject{
			Path:    path,
			ModTime: existing.UpdatedAt,
			ETag:    fmt.Sprintf("%d", existing.UpdatedAt.Unix()),
			Data:    buildCalendarFromTask(&existing),
		}, nil
	}

	var contact models.Contact
	hasContact := b.db.Where("vault_id = ?", vaultID).First(&contact).Error == nil

	task := models.ContactTask{
		VaultID:    vaultID,
		AuthorID:   &userID,
		UUID:       &uid,
		Label:      summary,
		AuthorName: "DAV Client",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if description != "" {
		task.Description = &description
	}
	if err := b.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		if hasContact {
			return tx.Create(&models.TaskContact{ContactTaskID: task.ID, ContactID: contact.ID}).Error
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &caldav.CalendarObject{
		Path:    path,
		ModTime: task.UpdatedAt,
		ETag:    fmt.Sprintf("%d", task.UpdatedAt.Unix()),
		Data:    buildCalendarFromTask(&task),
	}, nil
}

func (b *CalDAVBackend) DeleteCalendarObject(ctx context.Context, path string) error {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return fmt.Errorf("no user in context")
	}

	objectID := extractObjectIDFromPath(path, ".ics")
	if objectID == "" {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar object not found"))
	}

	// Try important date
	var importantDate models.ContactImportantDate
	if err := b.db.Preload("Contact").First(&importantDate, "uuid = ?", objectID).Error; err == nil {
		if err := b.verifyVaultAccess(userID, importantDate.Contact.VaultID); err != nil {
			return err
		}
		return b.db.Delete(&importantDate).Error
	}

	var task models.ContactTask
	if err := b.db.First(&task, "uuid = ?", objectID).Error; err == nil {
		if err := b.verifyVaultAccess(userID, task.VaultID); err != nil {
			return err
		}
		return b.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("contact_task_id = ?", task.ID).Delete(&models.TaskContact{}).Error; err != nil {
				return err
			}
			return tx.Delete(&task).Error
		})
	}

	return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar object not found"))
}

func (b *CalDAVBackend) verifyVaultAccess(userID, vaultID string) error {
	var uv models.UserVault
	if err := b.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err != nil {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("access denied"))
	}
	return nil
}

// importantDateToCalendarObject converts a ContactImportantDate to a CalDAV CalendarObject.
func importantDateToCalendarObject(d *models.ContactImportantDate, userID string) *caldav.CalendarObject {
	uid := ""
	if d.UUID != nil {
		uid = *d.UUID
	}

	cal := buildCalendarFromImportantDate(d)

	return &caldav.CalendarObject{
		Path:    "/dav/calendars/" + userID + "/" + d.Contact.VaultID + "/" + uid + ".ics",
		ModTime: d.UpdatedAt,
		ETag:    fmt.Sprintf("%d", d.UpdatedAt.Unix()),
		Data:    cal,
	}
}

// taskToCalendarObject converts a ContactTask to a CalDAV CalendarObject.
func taskToCalendarObject(t *models.ContactTask, userID string) *caldav.CalendarObject {
	uid := ""
	if t.UUID != nil {
		uid = *t.UUID
	}

	cal := buildCalendarFromTask(t)

	return &caldav.CalendarObject{
		Path:    "/dav/calendars/" + userID + "/" + t.VaultID + "/" + uid + ".ics",
		ModTime: t.UpdatedAt,
		ETag:    fmt.Sprintf("%d", t.UpdatedAt.Unix()),
		Data:    cal,
	}
}

// buildCalendarFromImportantDate creates an iCal VEVENT from a ContactImportantDate.
func buildCalendarFromImportantDate(d *models.ContactImportantDate) *ical.Calendar {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//Bonds//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")

	event := ical.NewComponent(ical.CompEvent)

	uid := ""
	if d.UUID != nil {
		uid = *d.UUID
	}
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetText(ical.PropSummary, d.Label)
	event.Props.SetDateTime(ical.PropDateTimeStamp, d.UpdatedAt)

	// Build DTSTART from day/month/year
	year := time.Now().Year()
	month := time.January
	day := 1

	if d.Year != nil {
		year = *d.Year
	}
	if d.Month != nil {
		month = time.Month(*d.Month)
	}
	if d.Day != nil {
		day = *d.Day
	}

	dtStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	prop := ical.NewProp(ical.PropDateTimeStart)
	prop.SetValueType(ical.ValueDate)
	prop.Value = dtStart.Format("20060102")
	event.Props.Set(prop)

	// Recurrence: for Gregorian we emit a simple RRULE=YEARLY since the
	// same Gregorian day every year is correct. For lunar (and any future
	// non-Gregorian calendars), RRULE=YEARLY would silently drift — lunar
	// dates land on a different Gregorian day each year — so we instead
	// compute the next several Gregorian occurrences via the calendar
	// converter and emit them as RDATE entries. This way Apple Calendar /
	// Thunderbird / Google Calendar render the lunar birthday on the right
	// day without needing lunar-calendar support of their own.
	isAlternative := d.CalendarType != "" && d.CalendarType != "gregorian" && d.OriginalMonth != nil && d.OriginalDay != nil
	if isAlternative {
		ct := calendarPkg.CalendarType(d.CalendarType)
		if converter, ok := calendarPkg.Get(ct); ok {
			emitLunarRDates(event, converter, d, dtStart)
		}
	} else if d.Year != nil {
		rruleProp := ical.NewProp(ical.PropRecurrenceRule)
		rruleProp.Value = "FREQ=YEARLY"
		event.Props.Set(rruleProp)
	}

	if isAlternative {
		desc := fmt.Sprintf("Calendar: %s, Original date: %d/%d", d.CalendarType, *d.OriginalMonth, *d.OriginalDay)
		if d.OriginalYear != nil {
			desc = fmt.Sprintf("Calendar: %s, Original date: %d-%d-%d", d.CalendarType, *d.OriginalYear, *d.OriginalMonth, *d.OriginalDay)
		}
		event.Props.SetText(ical.PropDescription, desc)
	}

	cal.Children = append(cal.Children, event)
	return cal
}

// emitLunarRDates appends RDATE properties to a lunar VEVENT for the next
// several years of Gregorian projections, derived from the original lunar
// date. Without this, downstream CalDAV clients would either not recur at
// all (no RRULE because we removed it) or — under the old code — recur on
// the wrong Gregorian day every year.
//
// The 10-year horizon is a pragmatic balance: long enough for typical
// calendar views (3-5 years) without bloating every VEVENT with decades of
// projections. The DTSTART itself remains the canonical first occurrence;
// RDATE entries supplement it.
func emitLunarRDates(event *ical.Component, converter calendarPkg.Converter, d *models.ContactImportantDate, dtStart time.Time) {
	const horizonYears = 10
	startYear := dtStart.Year()
	if d.OriginalYear != nil {
		startYear = *d.OriginalYear
	}

	values := []string{}
	for offset := 0; offset < horizonYears; offset++ {
		orig := calendarPkg.DateInfo{
			Day:   *d.OriginalDay,
			Month: *d.OriginalMonth,
			Year:  startYear + offset,
		}
		gd, err := converter.ToGregorian(orig)
		if err != nil {
			continue
		}
		values = append(values, fmt.Sprintf("%04d%02d%02d", gd.Year, gd.Month, gd.Day))
	}
	if len(values) == 0 {
		return
	}

	rdateProp := ical.NewProp(ical.PropRecurrenceDates)
	rdateProp.SetValueType(ical.ValueDate)
	rdateProp.Value = strings.Join(values, ",")
	event.Props.Set(rdateProp)
}

// buildCalendarFromTask creates an iCal VTODO from a ContactTask.
func buildCalendarFromTask(t *models.ContactTask) *ical.Calendar {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//Bonds//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")

	todo := ical.NewComponent(ical.CompToDo)

	uid := ""
	if t.UUID != nil {
		uid = *t.UUID
	}
	todo.Props.SetText(ical.PropUID, uid)
	todo.Props.SetText(ical.PropSummary, t.Label)
	todo.Props.SetDateTime(ical.PropDateTimeStamp, t.UpdatedAt)

	if t.Description != nil && *t.Description != "" {
		todo.Props.SetText(ical.PropDescription, *t.Description)
	}

	if t.Completed {
		todo.Props.SetText(ical.PropStatus, "COMPLETED")
		if t.CompletedAt != nil {
			todo.Props.SetDateTime(ical.PropCompleted, *t.CompletedAt)
		}
	} else {
		todo.Props.SetText(ical.PropStatus, "NEEDS-ACTION")
	}

	if t.DueAt != nil {
		todo.Props.SetDateTime(ical.PropDue, *t.DueAt)
	}

	todo.Props.SetText(ical.PropPercentComplete, func() string {
		if t.Completed {
			return "100"
		}
		return "0"
	}())

	cal.Children = append(cal.Children, todo)
	return cal
}

// extractVaultIDFromCalendarObjectPath extracts vault ID from a full object path
// like /dav/calendars/{userID}/{vaultID}/{objectID}.ics
func extractVaultIDFromCalendarObjectPath(path, userID string) string {
	return extractVaultIDFromAddressObjectPath(
		// Reuse the same logic but swap "addressbooks" for "calendars"
		replacePathSegment(path, "calendars", "addressbooks"),
		userID,
	)
}

func replacePathSegment(path, old, replacement string) string {
	// Replace /dav/calendars/ with /dav/addressbooks/ for path parsing reuse
	return "/dav/" + replacement + "/" + path[len("/dav/"+old+"/"):]
}
