package services

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type CalendarICSService struct {
	db *gorm.DB
}

func NewCalendarICSService(db *gorm.DB) *CalendarICSService {
	return &CalendarICSService{db: db}
}

// ExportVault renders every dated item in a vault — important dates, reminders,
// tasks and life events — into a single read-only iCalendar feed.
func (s *CalendarICSService) ExportVault(vaultID string) ([]byte, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//Bonds//Calendar Feed//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropCalendarScale, "GREGORIAN")
	cal.Props.SetText("X-WR-CALNAME", "Bonds")

	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
	}

	if len(contactIDs) > 0 {
		var dates []models.ContactImportantDate
		if err := s.db.Where("contact_id IN ?", contactIDs).Find(&dates).Error; err != nil {
			return nil, err
		}
		for i := range dates {
			cal.Children = append(cal.Children, icsImportantDateEvent(&dates[i]))
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

	lifeEvents, err := s.listVaultLifeEvents(vaultID)
	if err != nil {
		return nil, err
	}
	for i := range lifeEvents {
		cal.Children = append(cal.Children, icsLifeEventEvent(&lifeEvents[i]))
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

func (s *CalendarICSService) listVaultLifeEvents(vaultID string) ([]models.LifeEvent, error) {
	var events []models.LifeEvent
	if err := s.db.
		Joins("JOIN timeline_events ON timeline_events.id = life_events.timeline_event_id").
		Where("timeline_events.vault_id = ?", vaultID).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func icsImportantDateEvent(d *models.ContactImportantDate) *ical.Component {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, icsUID(d.UUID, "important-date", d.ID))
	event.Props.SetText(ical.PropSummary, d.Label)
	event.Props.SetDateTime(ical.PropDateTimeStamp, d.UpdatedAt)

	year, month, day := dateParts(d.Year, d.Month, d.Day)
	dtStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	setDateValue(event, ical.PropDateTimeStart, dtStart)

	isAlternative := d.CalendarType != "" && d.CalendarType != "gregorian" && d.OriginalMonth != nil && d.OriginalDay != nil
	if isAlternative {
		if converter, ok := calendarPkg.Get(calendarPkg.CalendarType(d.CalendarType)); ok {
			emitLunarRecurrence(event, converter, d.OriginalDay, d.OriginalMonth, d.OriginalYear, dtStart)
		}
		desc := fmt.Sprintf("Calendar: %s, Original date: %d/%d", d.CalendarType, *d.OriginalMonth, *d.OriginalDay)
		if d.OriginalYear != nil {
			desc = fmt.Sprintf("Calendar: %s, Original date: %d-%d-%d", d.CalendarType, *d.OriginalYear, *d.OriginalMonth, *d.OriginalDay)
		}
		event.Props.SetText(ical.PropDescription, desc)
	} else {
		setYearlyRecurrence(event)
	}

	return event
}

func icsReminderEvent(r *models.ContactReminder) *ical.Component {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, icsUID(nil, "reminder", r.ID))
	event.Props.SetText(ical.PropSummary, r.Label)
	event.Props.SetDateTime(ical.PropDateTimeStamp, r.UpdatedAt)

	year, month, day := dateParts(r.Year, r.Month, r.Day)
	dtStart := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	setDateValue(event, ical.PropDateTimeStart, dtStart)

	isAlternative := r.CalendarType != "" && r.CalendarType != "gregorian" && r.OriginalMonth != nil && r.OriginalDay != nil
	if isAlternative {
		if converter, ok := calendarPkg.Get(calendarPkg.CalendarType(r.CalendarType)); ok {
			emitLunarRecurrence(event, converter, r.OriginalDay, r.OriginalMonth, r.OriginalYear, dtStart)
		}
	} else {
		setYearlyRecurrence(event)
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

func icsLifeEventEvent(e *models.LifeEvent) *ical.Component {
	event := ical.NewComponent(ical.CompEvent)
	event.Props.SetText(ical.PropUID, icsUID(nil, "life-event", e.ID))

	summary := "Life event"
	if e.Summary != nil && *e.Summary != "" {
		summary = *e.Summary
	}
	event.Props.SetText(ical.PropSummary, summary)
	event.Props.SetDateTime(ical.PropDateTimeStamp, e.UpdatedAt)

	if e.Description != nil && *e.Description != "" {
		event.Props.SetText(ical.PropDescription, *e.Description)
	}

	setDateValue(event, ical.PropDateTimeStart, e.HappenedAt.UTC())
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

// emitLunarRecurrence mirrors the CalDAV backend: non-Gregorian dates drift
// against the Gregorian calendar each year, so a plain FREQ=YEARLY would land
// on the wrong day. Instead we project the next several occurrences via the
// calendar converter and emit them as RDATE entries that any client renders.
func emitLunarRecurrence(c *ical.Component, converter calendarPkg.Converter, origDay, origMonth, origYear *int, dtStart time.Time) {
	if origDay == nil || origMonth == nil {
		return
	}
	const horizonYears = 10
	startYear := dtStart.Year()
	if origYear != nil {
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

func icsUID(uuid *string, kind string, id uint) string {
	if uuid != nil && *uuid != "" {
		return *uuid
	}
	return fmt.Sprintf("bonds-%s-%d", kind, id)
}
