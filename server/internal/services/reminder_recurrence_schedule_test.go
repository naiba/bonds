package services

import (
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
)

func reminderWithAnchor(reminderType string, month, day int) *models.ContactReminder {
	return &models.ContactReminder{Type: reminderType, Month: &month, Day: &day}
}

// Editing a reminder recomputes its first schedule from the anchor date. For a
// weekly or monthly reminder whose anchor already passed this year that used
// to jump a whole year ahead, silencing the reminder until then.
func TestCalcInitialSchedule_RecurringReminderContinuesAtItsNextOccurrence(t *testing.T) {
	now := time.Date(2026, time.November, 20, 12, 0, 0, 0, time.UTC)
	two := 2
	cases := []struct {
		name     string
		reminder *models.ContactReminder
		want     time.Time
	}{
		// Anchor Tue 2026-03-10; weekly occurrences fall on Tuesdays.
		{"weekly", reminderWithAnchor("recurring_week", 3, 10), time.Date(2026, time.November, 24, 9, 0, 0, 0, time.UTC)},
		{"monthly", reminderWithAnchor("recurring_month", 3, 10), time.Date(2026, time.December, 10, 9, 0, 0, 0, time.UTC)},
		{"monthly on the 31st clamps to short months", reminderWithAnchor("recurring_month", 1, 31), time.Date(2026, time.November, 30, 9, 0, 0, 0, time.UTC)},
		{"yearly still waits for next year", reminderWithAnchor("recurring_year", 3, 10), time.Date(2027, time.March, 10, 9, 0, 0, 0, time.UTC)},
		{"future anchor is used as is", reminderWithAnchor("recurring_week", 12, 25), time.Date(2026, time.December, 25, 9, 0, 0, 0, time.UTC)},
	}
	everyTwoWeeks := reminderWithAnchor("recurring_week", 3, 10)
	everyTwoWeeks.FrequencyNumber = &two
	cases = append(cases, struct {
		name     string
		reminder *models.ContactReminder
		want     time.Time
	}{"every two weeks keeps its phase", everyTwoWeeks, time.Date(2026, time.December, 1, 9, 0, 0, 0, time.UTC)})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calcInitialScheduleAt(tc.reminder, nil, time.UTC, now)
			if !got.Equal(tc.want) {
				t.Fatalf("first schedule = %s, want %s", got, tc.want)
			}
		})
	}
}

// A weekly or monthly reminder that stores its start year has to continue the
// series that began on that date. Rebuilding the anchor from "this year's"
// month and day silenced a monthly reminder started last December until this
// December, and moved a weekly reminder to a different weekday every year.
func TestCalcInitialSchedule_RecurringReminderContinuesFromItsStartYear(t *testing.T) {
	withStartYear := func(reminderType string, year, month, day int) *models.ContactReminder {
		reminder := reminderWithAnchor(reminderType, month, day)
		reminder.Year = &year
		return reminder
	}
	cases := []struct {
		name     string
		reminder *models.ContactReminder
		now      time.Time
		want     time.Time
	}{
		{
			"monthly started last December, edited in January",
			withStartYear("recurring_month", 2025, 12, 15),
			time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC),
			time.Date(2026, time.January, 15, 9, 0, 0, 0, time.UTC),
		},
		{
			// 2025-03-10 is a Monday; 2026-03-10 is a Tuesday.
			"weekly keeps the weekday it started on",
			withStartYear("recurring_week", 2025, 3, 10),
			time.Date(2026, time.November, 20, 12, 0, 0, 0, time.UTC),
			time.Date(2026, time.November, 23, 9, 0, 0, 0, time.UTC),
		},
		{
			"start date in a later year is used as is",
			withStartYear("recurring_month", 2027, 2, 10),
			time.Date(2026, time.November, 20, 12, 0, 0, 0, time.UTC),
			time.Date(2027, time.February, 10, 9, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calcInitialScheduleAt(tc.reminder, nil, time.UTC, tc.now)
			if !got.Equal(tc.want) {
				t.Fatalf("first schedule = %s (%s), want %s (%s)", got, got.Weekday(), tc.want, tc.want.Weekday())
			}
		})
	}
}

// Stepping from the previous occurrence lets Go normalise Jan 31 + 1 month to
// Mar 3 and the reminder then stays on the 3rd for good. The day has to come
// from the reminder's own anchor, clamped to the length of each month.
func TestNextRecurringSchedule_KeepsTheAnchorDayAcrossShortMonths(t *testing.T) {
	monthEnd := reminderWithAnchor("recurring_month", 1, 31)
	fromJanuary := time.Date(2026, time.January, 31, 9, 0, 0, 0, time.UTC)

	february, ok := nextRecurringSchedule(monthEnd, nil, fromJanuary, time.UTC)
	if !ok || !february.Equal(time.Date(2026, time.February, 28, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("after Jan 31 = %s (ok=%v), want 2026-02-28", february, ok)
	}
	march, ok := nextRecurringSchedule(monthEnd, nil, february, time.UTC)
	if !ok || !march.Equal(time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("after Feb 28 = %s (ok=%v), want 2026-03-31", march, ok)
	}

	leapDay := reminderWithAnchor("recurring_year", 2, 29)
	nonLeap, ok := nextRecurringSchedule(leapDay, nil, time.Date(2024, time.February, 29, 9, 0, 0, 0, time.UTC), time.UTC)
	if !ok || !nonLeap.Equal(time.Date(2025, time.February, 28, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("year after Feb 29 = %s (ok=%v), want 2025-02-28", nonLeap, ok)
	}
	backOnLeapDay, ok := nextRecurringSchedule(leapDay, nil, time.Date(2027, time.February, 28, 9, 0, 0, 0, time.UTC), time.UTC)
	if !ok || !backOnLeapDay.Equal(time.Date(2028, time.February, 29, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("leap year after Feb 28 = %s (ok=%v), want 2028-02-29", backOnLeapDay, ok)
	}
}

// A one-time reminder that was already delivered must not go out again just
// because someone edited its label.
func TestReschedulePendingReminder_DoesNotResendDeliveredOneTimeReminder(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	channel := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "one_time", nil)
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	year, month, day := yesterday.Year(), int(yesterday.Month()), yesterday.Day()
	reminder.Year, reminder.Month, reminder.Day = &year, &month, &day
	if err := ctx.db.Save(reminder).Error; err != nil {
		t.Fatalf("save reminder date: %v", err)
	}
	if err := reschedulePendingReminder(ctx.db, reminder); err != nil {
		t.Fatalf("initial schedule: %v", err)
	}
	ctx.svc.ProcessDueReminders()
	// One delivery per active channel of the user (registration creates one too).
	deliveredBeforeEdit := len(ctx.mailer.calls)
	if deliveredBeforeEdit == 0 {
		t.Fatal("test setup: expected the past-dated reminder to be delivered")
	}

	reminder.Label = "Edited label"
	if err := reschedulePendingReminder(ctx.db, reminder); err != nil {
		t.Fatalf("reschedule after edit: %v", err)
	}
	ctx.svc.ProcessDueReminders()

	if len(ctx.mailer.calls) != deliveredBeforeEdit {
		t.Fatalf("deliveries after editing a delivered one-time reminder = %d, want still %d", len(ctx.mailer.calls), deliveredBeforeEdit)
	}
	var pending int64
	if err := ctx.db.Model(&models.ContactReminderScheduled{}).
		Where("contact_reminder_id = ? AND user_notification_channel_id = ? AND triggered_at IS NULL", reminder.ID, channel.ID).
		Count(&pending).Error; err != nil {
		t.Fatalf("count pending schedules: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending schedules = %d, want 0", pending)
	}
}

// Importing old data or moving a contact to another vault schedules reminders
// again. A one-time reminder dated years ago would be due immediately and go
// out as if it were new.
func TestReschedulePendingReminder_SkipsLongExpiredOneTimeReminder(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "one_time", nil)
	year, month, day := 2020, 5, 17
	reminder.Year, reminder.Month, reminder.Day = &year, &month, &day
	if err := ctx.db.Save(reminder).Error; err != nil {
		t.Fatalf("save reminder date: %v", err)
	}
	if err := ctx.db.Where("contact_reminder_id = ?", reminder.ID).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
		t.Fatalf("clear schedules created with the reminder: %v", err)
	}
	deliveredBefore := len(ctx.mailer.calls)

	if err := reschedulePendingReminder(ctx.db, reminder); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	ctx.svc.ProcessDueReminders()

	var scheduled int64
	if err := ctx.db.Model(&models.ContactReminderScheduled{}).Where("contact_reminder_id = ?", reminder.ID).Count(&scheduled).Error; err != nil {
		t.Fatalf("count schedules: %v", err)
	}
	if scheduled != 0 {
		t.Errorf("schedules for a reminder dated 2020 = %d, want 0", scheduled)
	}
	if len(ctx.mailer.calls) != deliveredBefore {
		t.Errorf("deliveries = %d, want still %d", len(ctx.mailer.calls), deliveredBefore)
	}
}

type blockingMailer struct {
	mu       sync.Mutex
	sends    int
	entered  chan struct{}
	release  chan struct{}
	enterOne sync.Once
}

func (m *blockingMailer) Send(_, _, _ string) error {
	m.mu.Lock()
	m.sends++
	m.mu.Unlock()
	m.enterOne.Do(func() { close(m.entered) })
	<-m.release
	return nil
}

func (m *blockingMailer) Close() {}

// The job fires every minute. When a run is still sending (slow SMTP, many
// reminders) the next tick finds the same undelivered rows; without a guard
// both runs send them.
func TestProcessDueReminders_OverlappingRunsDoNotSendTwice(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	channel := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "one_time", nil)
	ctx.createScheduled(t, reminder.ID, channel.ID, time.Now().Add(-time.Minute).UTC(), nil)
	mailer := &blockingMailer{entered: make(chan struct{}), release: make(chan struct{})}
	svc := NewReminderSchedulerService(ctx.db, mailer, &NoopSender{})

	firstRunDone := make(chan struct{})
	go func() {
		defer close(firstRunDone)
		svc.ProcessDueReminders()
	}()
	select {
	case <-mailer.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first run never started sending")
	}

	overlappingRunDone := make(chan struct{})
	go func() {
		defer close(overlappingRunDone)
		svc.ProcessDueReminders()
	}()
	select {
	case <-overlappingRunDone:
	case <-time.After(2 * time.Second):
		close(mailer.release)
		<-firstRunDone
		t.Fatal("the overlapping run did not return: it is sending the same reminder again")
	}
	close(mailer.release)
	<-firstRunDone

	mailer.mu.Lock()
	sends := mailer.sends
	mailer.mu.Unlock()
	if sends != 1 {
		t.Fatalf("reminder sent %d times, want 1", sends)
	}
}
