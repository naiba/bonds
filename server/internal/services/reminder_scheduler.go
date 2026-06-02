package services

import (
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

const maxChannelFails = 10

type ReminderSchedulerService struct {
	db     *gorm.DB
	mailer Mailer
	sender NotificationSender
}

func NewReminderSchedulerService(db *gorm.DB, mailer Mailer, sender NotificationSender) *ReminderSchedulerService {
	return &ReminderSchedulerService{db: db, mailer: mailer, sender: sender}
}

// ProcessDueReminders finds all scheduled reminders that are due and processes them.
func (s *ReminderSchedulerService) ProcessDueReminders() {
	now := time.Now().Truncate(time.Minute)

	var scheduled []models.ContactReminderScheduled
	err := s.db.
		Where("scheduled_at <= ? AND triggered_at IS NULL", now).
		Preload("ContactReminder.Contact").
		Preload("UserNotificationChannel.User").
		Find(&scheduled).Error
	if err != nil {
		log.Printf("[reminder-scheduler] Failed to query due reminders: %v", err)
		return
	}

	if len(scheduled) == 0 {
		return
	}

	log.Printf("[reminder-scheduler] Found %d due reminders", len(scheduled))

	for i := range scheduled {
		s.processOne(&scheduled[i])
	}
}

func (s *ReminderSchedulerService) processOne(scheduled *models.ContactReminderScheduled) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[reminder-scheduler] Panic processing scheduled reminder %d: %v", scheduled.ID, r)
		}
	}()

	// Load related data if not preloaded
	if scheduled.ContactReminder.ID == 0 {
		if err := s.db.Preload("Contact").First(&scheduled.ContactReminder, scheduled.ContactReminderID).Error; err != nil {
			log.Printf("[reminder-scheduler] Failed to load reminder %d: %v", scheduled.ContactReminderID, err)
			return
		}
	}
	if scheduled.UserNotificationChannel.ID == 0 {
		if err := s.db.Preload("User").First(&scheduled.UserNotificationChannel, scheduled.UserNotificationChannelID).Error; err != nil {
			log.Printf("[reminder-scheduler] Failed to load channel %d: %v", scheduled.UserNotificationChannelID, err)
			return
		}
	}

	channel := &scheduled.UserNotificationChannel
	reminder := &scheduled.ContactReminder

	// Skip inactive channels
	if !channel.Active {
		log.Printf("[reminder-scheduler] Skipping scheduled %d: channel %d is inactive", scheduled.ID, channel.ID)
		return
	}

	// Build notification content. Pull the locale from the channel owner so a
	// user who reads Chinese gets a Chinese reminder regardless of which
	// language the server's HTTP request happened to carry. Falls back to "en"
	// when the user isn't loaded (legacy callers) — i18n.T then returns the
	// English text rather than the raw key.
	locale := "en"
	enableAltCalendar := false
	if channel.User != nil {
		if channel.User.Locale != "" {
			locale = channel.User.Locale
		}
		enableAltCalendar = channel.User.EnableAlternativeCalendar
	}
	contactName := buildContactName(&reminder.Contact, locale)
	dateStr := formatReminderDate(reminder, scheduled.ScheduledAt, enableAltCalendar)
	subject := i18n.Tt(locale, "reminder.subject", map[string]string{"label": reminder.Label})
	htmlBody := i18n.Tt(locale, "reminder.body", map[string]string{
		"label":   reminder.Label,
		"contact": contactName,
		"date":    dateStr,
	})

	var sendErr error
	switch channel.Type {
	case "email":
		sendErr = s.mailer.Send(channel.Content, subject, htmlBody)
	case "shoutrrr", "telegram", "ntfy", "gotify", "webhook":
		if s.sender != nil {
			sendErr = s.sender.Send(channel.Content, subject, htmlBody)
		} else {
			log.Printf("[reminder-scheduler] No notification sender configured for channel %d (type=%s)", channel.ID, channel.Type)
			return
		}
	default:
		log.Printf("[reminder-scheduler] Unknown channel type %q for channel %d", channel.Type, channel.ID)
		return
	}

	now := time.Now()

	if sendErr != nil {
		s.handleFailure(scheduled, channel, subject, htmlBody, sendErr, now)
	} else {
		s.handleSuccess(scheduled, channel, reminder, subject, htmlBody, now)
	}

	loc := userLocation(channel.User)
	base := scheduled.ScheduledAt.In(loc)
	s.rescheduleIfRecurring(reminder, channel.ID, base, loc)
}

func (s *ReminderSchedulerService) handleSuccess(
	scheduled *models.ContactReminderScheduled,
	channel *models.UserNotificationChannel,
	reminder *models.ContactReminder,
	subject, htmlBody string,
	now time.Time,
) {
	// Record successful send
	s.db.Create(&models.UserNotificationSent{
		UserNotificationChannelID: channel.ID,
		SentAt:                    now,
		SubjectLine:               subject,
		Payload:                   &htmlBody,
	})

	// Mark as triggered
	s.db.Model(scheduled).Update("triggered_at", now)

	// Reset channel fails if needed
	if channel.Fails > 0 {
		s.db.Model(channel).Update("fails", 0)
	}

	// Update reminder tracking
	s.db.Model(reminder).Updates(map[string]interface{}{
		"last_triggered_at":      now,
		"number_times_triggered": gorm.Expr("number_times_triggered + 1"),
	})
}

func (s *ReminderSchedulerService) handleFailure(
	scheduled *models.ContactReminderScheduled,
	channel *models.UserNotificationChannel,
	subject, htmlBody string,
	sendErr error,
	now time.Time,
) {
	errMsg := sendErr.Error()

	// Record failed send
	s.db.Create(&models.UserNotificationSent{
		UserNotificationChannelID: channel.ID,
		SentAt:                    now,
		SubjectLine:               subject,
		Payload:                   &htmlBody,
		Error:                     &errMsg,
	})

	// Increment fails
	newFails := channel.Fails + 1
	updates := map[string]interface{}{"fails": newFails}
	if newFails >= maxChannelFails {
		updates["active"] = false
		log.Printf("[reminder-scheduler] Channel %d auto-disabled after %d failures", channel.ID, newFails)
	}
	s.db.Model(channel).Updates(updates)

	// Update in-memory value for reschedule check
	channel.Fails = newFails
	if newFails >= maxChannelFails {
		channel.Active = false
	}
}

func (s *ReminderSchedulerService) rescheduleIfRecurring(reminder *models.ContactReminder, channelID uint, now time.Time, loc *time.Location) {
	if reminder.Type == "one_time" {
		return
	}

	// Don't reschedule if channel was deactivated
	var channel models.UserNotificationChannel
	if err := s.db.First(&channel, channelID).Error; err != nil {
		log.Printf("[reminder-scheduler] Failed to check channel %d for reschedule: %v", channelID, err)
		return
	}
	if !channel.Active {
		return
	}
	hour, minute := parsePreferredNotificationTime(channel.PreferredTime)

	freq := 1
	if reminder.FrequencyNumber != nil {
		freq = *reminder.FrequencyNumber
	}

	var nextSchedule time.Time
	switch reminder.Type {
	case "recurring_week":
		nextSchedule = now.AddDate(0, 0, freq*7)
	case "recurring_month":
		nextSchedule = now.AddDate(0, freq, 0)
	case "recurring_year":
		if ns, ok := calcNextYearlySchedule(reminder, now, loc); ok {
			nextSchedule = ns
		} else {
			nextSchedule = now.AddDate(freq, 0, 0)
		}
	default:
		return
	}
	nextSchedule = time.Date(nextSchedule.Year(), nextSchedule.Month(), nextSchedule.Day(), hour, minute, 0, 0, loc)

	s.db.Create(&models.ContactReminderScheduled{
		UserNotificationChannelID: channelID,
		ContactReminderID:         reminder.ID,
		ScheduledAt:               nextSchedule,
	})
}

func calcNextYearlySchedule(reminder *models.ContactReminder, now time.Time, loc *time.Location) (time.Time, bool) {
	ct := calendarPkg.CalendarType(reminder.CalendarType)
	if ct == "" || ct == calendarPkg.Gregorian {
		return time.Time{}, false
	}
	if reminder.OriginalMonth == nil || reminder.OriginalDay == nil {
		return time.Time{}, false
	}
	converter, ok := calendarPkg.Get(ct)
	if !ok {
		return time.Time{}, false
	}
	origDate := calendarPkg.DateInfo{
		Day:   *reminder.OriginalDay,
		Month: *reminder.OriginalMonth,
	}
	gd, err := converter.NextOccurrence(origDate, now)
	if err != nil {
		log.Printf("[reminder-scheduler] calendar NextOccurrence failed: %v", err)
		return time.Time{}, false
	}
	return time.Date(gd.Year, time.Month(gd.Month), gd.Day, now.Hour(), now.Minute(), 0, 0, loc), true
}

// userLocation returns the time.Location for the given user's saved
// timezone preference, falling back to UTC if the user is nil, has no
// timezone set, or the saved string fails to load. Callers should pass
// this to any date math that ends up persisted (scheduled fire times,
// reminder dates) so different users in the same vault don't all fire at
// the server's wall-clock 09:00.
func userLocation(user *models.User) *time.Location {
	if user == nil || user.Timezone == nil || *user.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(*user.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func buildContactName(contact *models.Contact, locale string) string {
	name := ""
	if contact.FirstName != nil {
		name = *contact.FirstName
	}
	if contact.LastName != nil {
		if name != "" {
			name += " "
		}
		name += *contact.LastName
	}
	if name == "" {
		name = i18n.T(locale, "reminder.unknown_contact")
	}
	return name
}
