package dav

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

func TestCalDAVWritesRejectMergedOwnership(t *testing.T) {
	for _, scenario := range []string{"update_event", "create_event", "create_todo"} {
		t.Run(scenario, func(t *testing.T) {
			backend, db, ctx, vault, user := setupCalDAVTest(t)
			contacts := []*models.Contact{createTestContact(t, db, vault, user, "Duplicate", ""), createTestContact(t, db, vault, user, "Duplicate", "")}
			sort.Slice(contacts, func(i, j int) bool { return contacts[i].ID < contacts[j].ID })
			source, target := contacts[0], contacts[1]
			service := services.NewContactService(db)
			uid := "synthetic-merge-calendar-event"
			var dateID uint
			if scenario == "update_event" {
				day, month, year, remind := 15, 6, 2027, true
				date, err := services.NewImportantDateService(db).Create(source.ID, vault, dto.CreateImportantDateRequest{Label: "Annual gathering", Day: &day, Month: &month, Year: &year, DatePrecision: "full", RemindMe: &remind})
				if err != nil {
					t.Fatal(err)
				}
				dateID = date.ID
				if err := db.Model(&models.ContactImportantDate{}).Where("id = ?", dateID).Update("uuid", uid).Error; err != nil {
					t.Fatal(err)
				}
			}
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			review, err := service.PreviewContactMerge(vault, user, req)
			if err != nil {
				t.Fatal(err)
			}
			if len(review.Blockers) > 0 {
				t.Fatalf("invalid merge fixture: %+v", review)
			}
			req.ReviewToken = review.ReviewToken
			loaded, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "acceptance:calendar_owner_loaded"
			if err := db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				if tx.Error != nil {
					return
				}
				matches := false
				if scenario == "update_event" {
					row, ok := tx.Statement.Dest.(*models.ContactImportantDate)
					matches = ok && row.ID == dateID && row.Contact.ID == source.ID
				} else {
					row, ok := tx.Statement.Dest.(*models.Contact)
					matches = ok && row.ID == source.ID
				}
				if matches && paused.CompareAndSwap(false, true) {
					close(loaded)
					select {
					case <-resume:
					case <-time.After(10 * time.Second):
						tx.AddError(errors.New("calendar interleaving timeout"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Query().Remove(hook)
			calendar := ical.NewCalendar()
			calendar.Props.SetText(ical.PropProductID, "-//Bonds Acceptance//EN")
			calendar.Props.SetText(ical.PropVersion, "2.0")
			kind := ical.CompEvent
			if scenario == "create_todo" {
				kind = ical.CompToDo
			}
			event := ical.NewComponent(kind)
			event.Props.SetText(ical.PropUID, uid)
			event.Props.SetText(ical.PropSummary, "Edited annual gathering")
			event.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2027, 6, 16, 0, 0, 0, 0, time.UTC))
			calendar.Children = append(calendar.Children, event)
			finished := make(chan error, 1)
			go func() {
				_, e := backend.PutCalendarObject(ctx, "/dav/calendars/"+user+"/"+vault+"/"+uid+".ics", calendar, nil)
				finished <- e
			}()
			select {
			case <-loaded:
			case e := <-finished:
				t.Fatalf("write returned before owner read: %v", e)
			case <-time.After(10 * time.Second):
				close(resume)
				t.Fatal("owner read not reached")
			}
			_, mergeErr := service.MergeContacts(vault, user, req)
			close(resume)
			if mergeErr != nil {
				t.Fatal(mergeErr)
			}
			select {
			case err = <-finished:
			case <-time.After(10 * time.Second):
				t.Fatal("CalDAV write did not finish")
			}
			if err == nil || err.Error() != webdav.NewHTTPError(http.StatusNotFound, services.ErrContactNotFound).Error() {
				t.Errorf("stale write must return 404, got %v", err)
			}
			if scenario != "update_event" {
				var count int64
				table := "contact_important_dates"
				if scenario == "create_todo" {
					table = "contact_tasks"
				}
				if e := db.Table(table).Where("uuid = ?", uid).Count(&count).Error; e != nil {
					t.Fatal(e)
				}
				if count != 0 {
					t.Errorf("rejected creation persisted %d rows", count)
				}
				var pivots int64
				if e := db.Model(&models.TaskContact{}).Where("contact_id = ?", source.ID).Count(&pivots).Error; e != nil {
					t.Fatal(e)
				}
				if pivots != 0 {
					t.Errorf("rejected creation left %d source task assignments", pivots)
				}
			}
			var tombstone models.Contact
			if e := db.Unscoped().First(&tombstone, "id = ?", source.ID).Error; e != nil {
				t.Fatal(e)
			}
			if !tombstone.DeletedAt.Valid {
				t.Fatal("CalDAV resurrected merged source")
			}

			if scenario == "update_event" {
				var stored models.ContactImportantDate
				if e := db.First(&stored, dateID).Error; e != nil {
					t.Fatal(e)
				}
				if stored.ContactID != target.ID || stored.Label != "Annual gathering" || stored.Day == nil || *stored.Day != 15 {
					t.Fatalf("rejected write changed migrated date: %+v", stored)
				}
				assertCalendarReminder(t, db, dateID, target.ID, 15)
			}
			// A refreshed request resolves the surviving owner and remains writable.
			if _, e := backend.PutCalendarObject(ctx, "/dav/calendars/"+user+"/"+vault+"/"+uid+".ics", calendar, nil); e != nil {
				t.Fatal(e)
			}
			if scenario == "create_todo" {
				var task models.ContactTask
				if e := db.Where("uuid = ?", uid).First(&task).Error; e != nil {
					t.Fatal(e)
				}
				var assignments []models.TaskContact
				if e := db.Where("contact_task_id = ?", task.ID).Find(&assignments).Error; e != nil {
					t.Fatal(e)
				}
				if len(assignments) != 1 || assignments[0].ContactID != target.ID {
					t.Fatalf("incorrect retry assignments: %+v", assignments)
				}
			} else {
				var stored models.ContactImportantDate
				if e := db.Where("uuid = ?", uid).First(&stored).Error; e != nil {
					t.Fatal(e)
				}
				if stored.ContactID != target.ID || stored.Label != "Edited annual gathering" || stored.Day == nil || *stored.Day != 16 {
					t.Fatalf("incorrect retry date: %+v", stored)
				}
				if scenario == "update_event" {
					if stored.ID != dateID {
						t.Fatal("retry replaced original date")
					}
					assertCalendarReminder(t, db, dateID, target.ID, 16)
				}
			}
			objects, e := backend.ListCalendarObjects(ctx, "/dav/calendars/"+user+"/"+vault+"/", nil)
			if e != nil {
				t.Fatal(e)
			}
			if len(objects) != 1 {
				t.Fatalf("expected one visible calendar object, got %d", len(objects))
			}
		})
	}
}

func assertCalendarReminder(t *testing.T, db *gorm.DB, dateID uint, ownerID string, day int) {
	t.Helper()
	var reminders []models.ContactReminder
	if err := db.Where("important_date_id = ?", dateID).Find(&reminders).Error; err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 1 {
		t.Fatalf("expected one retained reminder, got %d", len(reminders))
	}
	reminder := reminders[0]
	if reminder.ContactID != ownerID || reminder.Day == nil || *reminder.Day != day {
		t.Fatalf("incorrect reminder owner/day: %+v", reminder)
	}
	var schedules []models.ContactReminderScheduled
	if err := db.Where("contact_reminder_id = ? AND triggered_at IS NULL", reminder.ID).Find(&schedules).Error; err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected one pending schedule, got %d", len(schedules))
	}
	if schedules[0].ScheduledAt.Day() != day || schedules[0].ScheduledAt.Month() != 6 {
		t.Fatalf("incorrect pending schedule: %+v", schedules[0])
	}
}

func TestCalDAVReminderEditRollsBackFailedSchedule(t *testing.T) {
	backend, db, ctx, vault, user := setupCalDAVTest(t)
	contact := createTestContact(t, db, vault, user, "Calendar owner", "")
	day, month, year, enabled := 15, 6, 2027, true
	date, err := services.NewImportantDateService(db).Create(contact.ID, vault, dto.CreateImportantDateRequest{Label: "Annual gathering", Day: &day, Month: &month, Year: &year, DatePrecision: "full", RemindMe: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	uid := "calendar-reminder-rollback"
	if err := db.Model(&models.ContactImportantDate{}).Where("id = ?", date.ID).Update("uuid", uid).Error; err != nil {
		t.Fatal(err)
	}
	var reminder models.ContactReminder
	if err := db.Where("important_date_id = ?", date.ID).First(&reminder).Error; err != nil {
		t.Fatal(err)
	}
	var original models.ContactReminderScheduled
	if err := db.Where("contact_reminder_id = ?", reminder.ID).First(&original).Error; err != nil {
		t.Fatal(err)
	}
	delivered := original
	delivered.ID = 0
	now := time.Now()
	delivered.TriggeredAt = &now
	if err := db.Create(&delivered).Error; err != nil {
		t.Fatal(err)
	}
	fail := errors.New("schedule persistence failed")
	hook := "caldav:reject_schedule"
	if err := db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_reminder_scheduled" {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Create().Remove(hook)
	calendar := calendarOwnershipPayload(uid, ical.CompEvent)
	path := "/dav/calendars/" + user + "/" + vault + "/" + uid + ".ics"
	if _, err := backend.PutCalendarObject(ctx, path, calendar, nil); !errors.Is(err, fail) {
		t.Fatalf("expected schedule failure, got %v", err)
	}
	var stored models.ContactImportantDate
	if err := db.First(&stored, date.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Label != "Annual gathering" || stored.Day == nil || *stored.Day != 15 {
		t.Fatalf("date was not rolled back: %+v", stored)
	}
	assertCalendarReminder(t, db, date.ID, contact.ID, 15)
	var pending models.ContactReminderScheduled
	if err := db.First(&pending, original.ID).Error; err != nil {
		t.Fatal("original pending schedule lost:", err)
	}
	if !pending.ScheduledAt.Equal(original.ScheduledAt) {
		t.Fatal("rollback changed original schedule")
	}
	if err := db.Callback().Create().Remove(hook); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.PutCalendarObject(ctx, path, calendar, nil); err != nil {
		t.Fatal(err)
	}
	assertCalendarReminder(t, db, date.ID, contact.ID, 16)
	var retained models.ContactReminder
	if err := db.First(&retained, reminder.ID).Error; err != nil {
		t.Fatal(err)
	}
	if retained.Label != "Edited annual gathering" || retained.ImportantDateID == nil || *retained.ImportantDateID != date.ID {
		t.Fatalf("reminder identity changed: %+v", retained)
	}
	var history models.ContactReminderScheduled
	if err := db.First(&history, delivered.ID).Error; err != nil {
		t.Fatal(err)
	}
	if history.TriggeredAt == nil || !history.ScheduledAt.Equal(delivered.ScheduledAt) || history.ContactReminderID != reminder.ID {
		t.Fatal("delivered history changed")
	}
}

func calendarOwnershipPayload(uid, kind string) *ical.Calendar {
	calendar := ical.NewCalendar()
	calendar.Props.SetText(ical.PropProductID, "-//Bonds Calendar Test//EN")
	calendar.Props.SetText(ical.PropVersion, "2.0")
	event := ical.NewComponent(kind)
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetText(ical.PropSummary, "Edited annual gathering")
	event.Props.SetDateTime(ical.PropDateTimeStart, time.Date(2027, 6, 16, 0, 0, 0, 0, time.UTC))
	calendar.Children = append(calendar.Children, event)
	return calendar
}

func TestCalDAVTaskEditPreservesConcurrentMergeAndCompletion(t *testing.T) {
	backend, db, ctx, vault, user := setupCalDAVTest(t)
	source := createTestContact(t, db, vault, user, "Duplicate", "")
	target := createTestContact(t, db, vault, user, "Duplicate", "")
	uid := "calendar-task-concurrent-completion"
	task := models.ContactTask{VaultID: vault, UUID: &uid, Label: "Original task"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TaskContact{ContactTaskID: task.ID, ContactID: source.ID}).Error; err != nil {
		t.Fatal(err)
	}
	svc := services.NewContactService(db)
	req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
	preview, err := svc.PreviewContactMerge(vault, user, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Blockers) != 0 {
		t.Fatalf("invalid merge fixture: %+v", preview.Blockers)
	}
	req.ReviewToken = preview.ReviewToken
	completedAt := time.Now().UTC().Truncate(time.Second)
	var interleaved atomic.Bool
	hook := "caldav:merge_and_complete_loaded_task"
	if err := db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		loaded, ok := tx.Statement.Dest.(*models.ContactTask)
		if tx.Error != nil || !ok || loaded.ID != task.ID || !interleaved.CompareAndSwap(false, true) {
			return
		}
		if _, err := svc.MergeContacts(vault, user, req); err != nil {
			tx.AddError(err)
			return
		}
		// Another writer finishes the task after CalDAV has loaded its old state.
		tx.AddError(db.Model(&models.ContactTask{}).Where("id = ?", task.ID).Updates(map[string]any{
			"status": models.TaskStatusDone, "completed": true, "completed_at": completedAt,
		}).Error)
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Query().Remove(hook)
	path := "/dav/calendars/" + user + "/" + vault + "/" + uid + ".ics"
	if _, err := backend.PutCalendarObject(ctx, path, calendarOwnershipPayload(uid, ical.CompToDo), nil); err != nil {
		t.Fatal(err)
	}
	if !interleaved.Load() {
		t.Fatal("task read did not reach concurrent merge")
	}
	var stored models.ContactTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Label != "Edited annual gathering" || stored.VaultID != vault || stored.Status != models.TaskStatusDone || !stored.Completed || stored.CompletedAt == nil || !stored.CompletedAt.Equal(completedAt) {
		t.Fatalf("calendar edit overwrote concurrent task state: %+v", stored)
	}
	var assignments []models.TaskContact
	if err := db.Where("contact_task_id = ?", task.ID).Find(&assignments).Error; err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 || assignments[0].ContactID != target.ID {
		t.Fatalf("calendar edit lost merged assignment: %+v", assignments)
	}
}

func TestCalDAVWritesHoldOwnerBeforeMerge(t *testing.T) {
	for _, scenario := range []string{"update_event", "create_event", "create_todo"} {
		t.Run(scenario, func(t *testing.T) {
			backend, db, ctx, vault, user := setupCalDAVTest(t)
			if db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL parent row lock interleaving")
			}
			source := createTestContact(t, db, vault, user, "Duplicate", "")
			target := createTestContact(t, db, vault, user, "Duplicate", "")
			// Creation chooses the first contact, so make its identity deterministic.
			if target.ID < source.ID {
				source, target = target, source
			}
			uid := "calendar-write-before-merge"
			if scenario == "update_event" {
				day, month, year, enabled := 15, 6, 2027, true
				date, err := services.NewImportantDateService(db).Create(source.ID, vault, dto.CreateImportantDateRequest{Label: "Annual gathering", Day: &day, Month: &month, Year: &year, DatePrecision: "full", RemindMe: &enabled})
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Model(&models.ContactImportantDate{}).Where("id = ?", date.ID).Update("uuid", uid).Error; err != nil {
					t.Fatal(err)
				}
			}
			svc := services.NewContactService(db)
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			preview, err := svc.PreviewContactMerge(vault, user, req)
			if err != nil {
				t.Fatal(err)
			}
			if len(preview.Blockers) != 0 {
				t.Fatalf("invalid fixture: %+v", preview.Blockers)
			}
			req.ReviewToken = preview.ReviewToken
			written, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "caldav:pause_child_write"
			callback := func(tx *gorm.DB) {
				if tx.Error != nil {
					return
				}
				table := "contact_important_dates"
				if scenario == "create_todo" {
					table = "contact_tasks"
				}
				if tx.Statement.Table == table && paused.CompareAndSwap(false, true) {
					close(written)
					select {
					case <-resume:
					case <-time.After(10 * time.Second):
						tx.AddError(errors.New("calendar write timeout"))
					}
				}
			}
			if scenario == "update_event" {
				if err := db.Callback().Update().After("gorm:update").Register(hook, callback); err != nil {
					t.Fatal(err)
				}
				defer db.Callback().Update().Remove(hook)
			} else {
				if err := db.Callback().Create().After("gorm:create").Register(hook, callback); err != nil {
					t.Fatal(err)
				}
				defer db.Callback().Create().Remove(hook)
			}
			ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			backend = NewCalDAVBackend(db.WithContext(ctx))
			kind := ical.CompEvent
			if scenario == "create_todo" {
				kind = ical.CompToDo
			}
			finished := make(chan error, 1)
			go func() {
				_, err := backend.PutCalendarObject(ctx, "/dav/calendars/"+user+"/"+vault+"/"+uid+".ics", calendarOwnershipPayload(uid, kind), nil)
				finished <- err
			}()
			select {
			case <-written:
			case err := <-finished:
				close(resume)
				t.Fatalf("write ended before barrier: %v", err)
			case <-time.After(10 * time.Second):
				close(resume)
				t.Fatal("write barrier not reached")
			}
			merged := make(chan error, 1)
			go func() {
				_, err := services.NewContactService(db.WithContext(ctx)).MergeContacts(vault, user, req)
				merged <- err
			}()
			waited := false
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				var count int64
				err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE '%contacts%FOR UPDATE%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='contacts'::regclass)`).Scan(&count).Error
				if err != nil {
					close(resume)
					t.Fatal(err)
				}
				if count > 0 {
					waited = true
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			close(resume)
			select {
			case err := <-finished:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("write did not finish")
			}
			select {
			case err := <-merged:
				if !errors.Is(err, services.ErrContactMergeReviewChanged) {
					t.Fatalf("expected refreshed review, got %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("merge did not finish")
			}
			if !waited {
				t.Fatal("merge did not wait for calendar owner lock")
			}
			preview, err = svc.PreviewContactMerge(vault, user, req)
			if err != nil {
				t.Fatal(err)
			}
			req.ReviewToken = preview.ReviewToken
			if _, err := svc.MergeContacts(vault, user, req); err != nil {
				t.Fatal(err)
			}
			if scenario == "create_todo" {
				var task models.ContactTask
				if err := db.Where("uuid = ?", uid).First(&task).Error; err != nil {
					t.Fatal(err)
				}
				var owners []models.TaskContact
				if err := db.Where("contact_task_id = ?", task.ID).Find(&owners).Error; err != nil {
					t.Fatal(err)
				}
				if len(owners) != 1 || owners[0].ContactID != target.ID {
					t.Fatalf("wrong merged owners: %+v", owners)
				}
			} else {
				var date models.ContactImportantDate
				if err := db.Where("uuid = ?", uid).First(&date).Error; err != nil {
					t.Fatal(err)
				}
				if date.ContactID != target.ID || date.Day == nil || *date.Day != 16 {
					t.Fatalf("wrong merged date: %+v", date)
				}
				if scenario == "update_event" {
					assertCalendarReminder(t, db, date.ID, target.ID, 16)
				}
			}
		})
	}
}
