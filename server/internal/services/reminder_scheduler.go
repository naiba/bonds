package services

import (
	"fmt"
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

const maxChannelFails = 10

type ReminderSchedulerService struct {
	db     *gorm.DB
	mailer Mailer
}

func NewReminderSchedulerService(db *gorm.DB, mailer Mailer) *ReminderSchedulerService {
	return &ReminderSchedulerService{db: db, mailer: mailer}
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

	// Build notification content
	contactName := buildContactName(&reminder.Contact)
	subject := fmt.Sprintf("Reminder: %s", reminder.Label)
	htmlBody := fmt.Sprintf(
		`<h2>Reminder: %s</h2><p>You have a reminder for <strong>%s</strong>.</p><p>%s</p>`,
		reminder.Label, contactName, reminder.Label,
	)

	// Send based on channel type
	var sendErr error
	switch channel.Type {
	case "email":
		sendErr = s.mailer.Send(channel.Content, subject, htmlBody)
	case "telegram":
		log.Printf("[reminder-scheduler] Telegram not yet implemented for channel %d", channel.ID)
		sendErr = nil // Not an error — just not implemented yet
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

	// Reschedule if recurring
	s.rescheduleIfRecurring(reminder, channel.ID, now)
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

func (s *ReminderSchedulerService) rescheduleIfRecurring(reminder *models.ContactReminder, channelID uint, now time.Time) {
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
		if ns, ok := calcNextYearlySchedule(reminder, now); ok {
			nextSchedule = ns
		} else {
			nextSchedule = now.AddDate(freq, 0, 0)
		}
	default:
		return
	}

	s.db.Create(&models.ContactReminderScheduled{
		UserNotificationChannelID: channelID,
		ContactReminderID:         reminder.ID,
		ScheduledAt:               nextSchedule,
	})
}

func calcNextYearlySchedule(reminder *models.ContactReminder, now time.Time) (time.Time, bool) {
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
	return time.Date(gd.Year, time.Month(gd.Month), gd.Day, 9, 0, 0, 0, now.Location()), true
}

func buildContactName(contact *models.Contact) string {
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
		name = "Unknown"
	}
	return name
}
