package services

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/utils"
	"gorm.io/gorm"
)

type CalendarICSService struct {
	db *gorm.DB
}

func NewCalendarICSService(db *gorm.DB) *CalendarICSService {
	return &CalendarICSService{db: db}
}

// ExportVault renders every dated item in a vault — important dates, reminders,
// tasks and activities — into a single read-only iCalendar feed.
func (s *CalendarICSService) ExportVault(vaultID, userID string) ([]byte, error) {
	nameOrder, err := GetUserNameOrder(s.db, userID)
	if err != nil {
		return nil, err
	}

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//Bonds//Calendar Feed//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropCalendarScale, "GREGORIAN")
	cal.Props.SetText("X-WR-CALNAME", "Bonds")

	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).
		Select("id, first_name, middle_name, last_name, nickname, maiden_name, prefix, suffix").
		Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactIDs := make([]string, len(contacts))
	contactNames := make(map[string]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
		contactNames[c.ID] = utils.FormatContactName(nameOrder, &contacts[i], "")
	}

	if len(contactIDs) > 0 {
		var dates []models.ContactImportantDate
		if err := s.db.Where("contact_id IN ?", contactIDs).Find(&dates).Error; err != nil {
			return nil, err
		}
		for i := range dates {
			cal.Children = append(cal.Children, icsImportantDateEvent(&dates[i], contactNames[dates[i].ContactID]))
		}

		var reminders []models.ContactReminder
		if err := s.db.Where("contact_id IN ?", contactIDs).Find(&reminders).Error; err != nil {
			return nil, err
		}
		for i := range reminders {
			cal.Children = append(cal.Children, icsReminderEvent(&reminders[i]))
		}
	}

	tasks, err := s.listVaultTasks(vaultID, contactIDs)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		cal.Children = append(cal.Children, icsTaskToDo(&tasks[i]))
	}

	activities, err := s.listVaultActivities(vaultID)
	if err != nil {
		return nil, err
	}
	for i := range activities {
		cal.Children = append(cal.Children, icsActivityEvent(&activities[i]))
	}

	if len(cal.Children) == 0 {
		// The go-ical encoder refuses a childless VCALENDAR, but a vault with no
		// dated items must still produce a valid (empty) feed for subscribers.
		return []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Bonds//Calendar Feed//EN\r\nCALSCALE:GREGORIAN\r\nX-WR-CALNAME:Bonds\r\nEND:VCALENDAR\r\n"), nil
	}

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *CalendarICSService) listVaultTasks(vaultID string, contactIDs []string) ([]models.ContactTask, error) {
	var tasks []models.ContactTask
	q := s.db.Model(&models.ContactTask{}).
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

func (s *CalendarICSService) listVaultActivities(vaultID string) ([]models.Activity, error) {
	var events []models.Activity
	if err := s.db.
		Where("activities.vault_id = ?", vaultID).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func icsImportantDateEvent(d *models.ContactImportantDate, contactName string) *ical.Component {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, icsUID(d.UUID, "important-date", d.ID))
	event.Props.SetText(ical.PropSummary, icsImportantDateSummary(d, contactName))
	event.Props.SetDateTime(ical.PropDateTimeStamp, d.UpdatedAt)

	year, month, day := dateParts(d.Year, d.Month, d.Day)
	dtStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	isAlternative := d.CalendarType != "" && d.CalendarType != "gregorian" && d.OriginalMonth != nil && d.OriginalDay != nil
	if isAlternative {
		if converter, ok := calendarPkg.Get(calendarPkg.CalendarType(d.CalendarType)); ok {
			dtStart = projectLunarDtStart(converter, d.OriginalDay, d.OriginalMonth, d.OriginalYear, dtStart)
			setDateValue(event, ical.PropDateTimeStart, dtStart)
			emitLunarRecurrence(event, converter, d.OriginalDay, d.OriginalMonth, d.OriginalYear, dtStart)
		} else {
			setDateValue(event, ical.PropDateTimeStart, dtStart)
		}
		desc := fmt.Sprintf("Calendar: %s, Original date: %d/%d", d.CalendarType, *d.OriginalMonth, *d.OriginalDay)
		if d.OriginalYear != nil {
			desc = fmt.Sprintf("Calendar: %s, Original date: %d-%d-%d", d.CalendarType, *d.OriginalYear, *d.OriginalMonth, *d.OriginalDay)
		}
		event.Props.SetText(ical.PropDescription, desc)
	} else {
		setDateValue(event, ical.PropDateTimeStart, dtStart)
		// Important dates (especially birthdays) repeat every year regardless
		// of the stored year — the year field is only used for display, not
		// filtering (see CalendarService.GetCalendar). Matches CalDAV export.
		setYearlyRecurrence(event)
	}

	return event
}

func icsImportantDateSummary(d *models.ContactImportantDate, contactName string) string {
	// Calendar subscribers only see SUMMARY, so important dates need the contact
	// name there rather than relying on app-only context from the web calendar.
	if contactName == "" {
		return d.Label
	}
	return fmt.Sprintf("%s - %s", contactName, d.Label)
}

func icsReminderEvent(r *models.ContactReminder) *ical.Component {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, icsUID(nil, "reminder", r.ID))
	event.Props.SetText(ical.PropSummary, r.Label)
	event.Props.SetDateTime(ical.PropDateTimeStamp, r.UpdatedAt)

	year, month, day := dateParts(r.Year, r.Month, r.Day)
	dtStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	isAlternative := r.CalendarType != "" && r.CalendarType != "gregorian" && r.OriginalMonth != nil && r.OriginalDay != nil
	if isAlternative {
		if converter, ok := calendarPkg.Get(calendarPkg.CalendarType(r.CalendarType)); ok {
			dtStart = projectLunarDtStart(converter, r.OriginalDay, r.OriginalMonth, r.OriginalYear, dtStart)
			setDateValue(event, ical.PropDateTimeStart, dtStart)
			emitLunarRecurrence(event, converter, r.OriginalDay, r.OriginalMonth, r.OriginalYear, dtStart)
		} else {
			setDateValue(event, ical.PropDateTimeStart, dtStart)
		}
		return event
	}

	setDateValue(event, ical.PropDateTimeStart, dtStart)
	// Map the reminder recurrence type onto an RRULE. one_time reminders are
	// single occurrences and get no recurrence rule at all.
	switch r.Type {
	case "recurring_week":
		setRecurrence(event, "FREQ=WEEKLY", r.FrequencyNumber)
	case "recurring_month":
		setRecurrence(event, "FREQ=MONTHLY", r.FrequencyNumber)
	case "recurring_year":
		setRecurrence(event, "FREQ=YEARLY", r.FrequencyNumber)
	}

	return event
}

func icsTaskToDo(t *models.ContactTask) *ical.Component {
	todo := ical.NewComponent(ical.CompToDo)
	todo.Props.SetText(ical.PropUID, icsUID(t.UUID, "task", t.ID))
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
		todo.Props.SetText(ical.PropPercentComplete, "100")
	} else {
		todo.Props.SetText(ical.PropStatus, "NEEDS-ACTION")
		todo.Props.SetText(ical.PropPercentComplete, "0")
	}
	if t.DueAt != nil {
		todo.Props.SetDateTime(ical.PropDue, *t.DueAt)
	}

	return todo
}

func icsActivityEvent(e *models.Activity) *ical.Component {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, icsUID(nil, "activity", e.ID))

	summary := "Activity"
	if e.Title != "" {
		summary = e.Title
	}
	event.Props.SetText(ical.PropSummary, summary)
	event.Props.SetDateTime(ical.PropDateTimeStamp, e.UpdatedAt)

	if e.Description != nil && *e.Description != "" {
		event.Props.SetText(ical.PropDescription, *e.Description)
	}

	if e.StartDate != nil {
		setDateValue(event, ical.PropDateTimeStart, e.StartDate.UTC())
	}
	if e.EndStatus == "known" && e.EndDate != nil {
		setDateValue(event, ical.PropDateTimeEnd, e.EndDate.UTC())
	}
	return event
}

func dateParts(y, m, d *int) (int, time.Month, int) {
	year := time.Now().Year()
	month := time.January
	day := 1
	if y != nil {
		year = *y
	}
	if m != nil {
		month = time.Month(*m)
	}
	if d != nil {
		day = *d
	}
	return year, month, day
}

func setDateValue(c *ical.Component, name string, t time.Time) {
	prop := ical.NewProp(name)
	prop.SetValueType(ical.ValueDate)
	prop.Value = t.Format("20060102")
	c.Props.Set(prop)
}

func setYearlyRecurrence(c *ical.Component) {
	prop := ical.NewProp(ical.PropRecurrenceRule)
	prop.Value = "FREQ=YEARLY"
	c.Props.Set(prop)
}

func setRecurrence(c *ical.Component, freq string, frequency *int) {
	value := freq
	if frequency != nil && *frequency > 1 {
		value = fmt.Sprintf("%s;INTERVAL=%d", freq, *frequency)
	}
	prop := ical.NewProp(ical.PropRecurrenceRule)
	prop.Value = value
	c.Props.Set(prop)
}

// projectLunarDtStart returns the Gregorian date of the first projected
// occurrence. For month+day-only lunar dates (Year is nil) the Gregorian
// projection is only valid for the current year, so the DTSTART must coincide
// with the first RDATE instead of the stored (nil) fields.
func projectLunarDtStart(converter calendarPkg.Converter, origDay, origMonth, origYear *int, fallback time.Time) time.Time {
	if origDay == nil || origMonth == nil {
		return fallback
	}
	original := calendarPkg.DateInfo{Day: *origDay, Month: *origMonth}
	if origYear == nil {
		// Reminder rows created through applyCalendarFields can retain the fixed
		// 2000 storage projection even though OriginalYear is nil. Never derive
		// DTSTART's recurrence year from that projection; resolve the occurrence
		// that actually lands in the current Gregorian year instead.
		if gd, err := calendarOccurrenceInYear(converter, original, time.Now().UTC().Year()); err == nil {
			return time.Date(gd.Year, time.Month(gd.Month), gd.Day, 0, 0, 0, 0, time.UTC)
		}
		return fallback
	}
	original.Year = *origYear
	if gd, err := converter.ToGregorian(original); err == nil {
		return time.Date(gd.Year, time.Month(gd.Month), gd.Day, 0, 0, 0, 0, time.UTC)
	}
	return fallback
}

// emitLunarRecurrence mirrors the CalDAV backend: non-Gregorian dates drift
// against the Gregorian calendar each year, so a plain FREQ=YEARLY would land
// on the wrong day. Instead we project the next several occurrences via the
// calendar converter and emit them as RDATE entries that any client renders.
func emitLunarRecurrence(c *ical.Component, converter calendarPkg.Converter, origDay, origMonth, origYear *int, dtStart time.Time) {
	if origDay == nil || origMonth == nil {
		return
	}
	const horizonYears = 50
	startYear := calendarYearForGregorianDate(converter, dtStart)
	if current, err := calendarOccurrenceInYear(converter, calendarPkg.DateInfo{
		Day:   *origDay,
		Month: *origMonth,
	}, time.Now().UTC().Year()); err == nil {
		currentDate := time.Date(current.Year, time.Month(current.Month), current.Day, 0, 0, 0, 0, time.UTC)
		startYear = calendarYearForGregorianDate(converter, currentDate)
	}
	if origYear != nil && *origYear > startYear {
		startYear = *origYear
	}

	values := []string{}
	for offset := 0; offset < horizonYears; offset++ {
		gd, err := converter.ToGregorian(calendarPkg.DateInfo{
			Day:   *origDay,
			Month: *origMonth,
			Year:  startYear + offset,
		})
		if err != nil {
			continue
		}
		values = append(values, fmt.Sprintf("%04d%02d%02d", gd.Year, gd.Month, gd.Day))
	}
	if len(values) == 0 {
		return
	}

	prop := ical.NewProp(ical.PropRecurrenceDates)
	prop.SetValueType(ical.ValueDate)
	prop.Value = strings.Join(values, ",")
	c.Props.Set(prop)
}

func calendarYearForGregorianDate(converter calendarPkg.Converter, date time.Time) int {
	converted, err := converter.FromGregorian(calendarPkg.GregorianDate{
		Day:   date.Day(),
		Month: int(date.Month()),
		Year:  date.Year(),
	})
	if err == nil && converted.Year != 0 {
		return converted.Year
	}
	return date.Year()
}

func icsUID(uuid *string, kind string, id uint) string {
	if uuid != nil && *uuid != "" {
		return *uuid
	}
	return fmt.Sprintf("bonds-%s-%d", kind, id)
}
