package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *ReminderSchedulerService) processOne(scheduled *models.ContactReminderScheduled) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[reminder-scheduler] Panic processing scheduled reminder %d: %v", scheduled.ID, recovered)
		}
	}()

	current, eligible, err := s.loadEligibleSchedule(scheduled.ID)
	if err != nil {
		log.Printf("[reminder-scheduler] Load delivery eligibility for scheduled reminder %d: %v", scheduled.ID, err)
		return
	}
	if current == nil {
		return
	}
	if !eligible {
		// Materialized schedules are not authorization: membership and audience may change after scheduling.
		if err := s.discardIneligibleSchedule(current.ID); err != nil {
			log.Printf("[reminder-scheduler] Discard ineligible scheduled reminder %d: %v", current.ID, err)
		}
		return
	}

	channel := &current.UserNotificationChannel
	reminder := &current.ContactReminder
	locale, enableAltCalendar := reminderDeliveryLocale(channel.User)
	contactName, err := s.reminderContactName(reminder, channel.UserID, locale)
	if err != nil {
		log.Printf("[reminder-scheduler] Format contact name for reminder %d: %v", reminder.ID, err)
		return
	}
	subject, htmlBody := reminderDeliveryContent(reminder, current.ScheduledAt, locale, enableAltCalendar, contactName)

	sendErr := s.sendReminder(channel, subject, htmlBody)
	if sendErr != nil {
		if err := s.handleFailure(current, channel, subject, htmlBody, sendErr, time.Now()); err != nil {
			log.Printf("[reminder-scheduler] Record failed delivery for scheduled reminder %d: %v", current.ID, err)
		}
		return
	}
	if err := s.handleSuccess(current, channel, reminder, subject, htmlBody, time.Now()); err != nil {
		log.Printf("[reminder-scheduler] Record successful delivery for scheduled reminder %d: %v", current.ID, err)
	}
}

func (s *ReminderSchedulerService) loadEligibleSchedule(scheduleID uint) (*models.ContactReminderScheduled, bool, error) {
	var scheduled models.ContactReminderScheduled
	err := s.db.Preload("ContactReminder.Contact").Preload("UserNotificationChannel.User").First(&scheduled, scheduleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("load schedule: %w", err)
	}
	if scheduled.TriggeredAt != nil || !scheduled.UserNotificationChannel.Active || scheduled.UserNotificationChannel.UserID == nil {
		return &scheduled, false, nil
	}

	userID := *scheduled.UserNotificationChannel.UserID
	var membershipCount int64
	if err := s.db.Model(&models.UserVault{}).
		Where("vault_id = ? AND user_id = ?", scheduled.ContactReminder.Contact.VaultID, userID).
		Count(&membershipCount).Error; err != nil {
		return nil, false, fmt.Errorf("check vault membership: %w", err)
	}
	if membershipCount == 0 {
		return &scheduled, false, nil
	}
	audience := reminderAudienceOrDefault(scheduled.ContactReminder.Audience)
	if audience == models.ReminderAudienceAllVaultUsers {
		return &scheduled, true, nil
	}
	if audience != models.ReminderAudienceSelectedUsers {
		return &scheduled, false, nil
	}

	var recipientCount int64
	if err := s.db.Model(&models.ContactReminderSelectedUser{}).
		Where("contact_reminder_id = ? AND user_id = ?", scheduled.ContactReminderID, userID).
		Count(&recipientCount).Error; err != nil {
		return nil, false, fmt.Errorf("check selected recipient: %w", err)
	}
	return &scheduled, recipientCount > 0, nil
}

func (s *ReminderSchedulerService) discardIneligibleSchedule(scheduleID uint) error {
	return s.db.Where("id = ? AND triggered_at IS NULL", scheduleID).Delete(&models.ContactReminderScheduled{}).Error
}

func reminderDeliveryLocale(user *models.User) (string, bool) {
	if user == nil {
		return i18n.Default, false
	}
	locale := user.Locale
	if locale == "" {
		locale = i18n.Default
	}
	return locale, user.EnableAlternativeCalendar
}

func (s *ReminderSchedulerService) reminderContactName(reminder *models.ContactReminder, userID *string, locale string) (string, error) {
	contactName := buildContactName(&reminder.Contact, locale)
	if userID == nil {
		return contactName, nil
	}
	formatter, err := newContactNameFormatter(s.db, *userID)
	if err != nil {
		return "", fmt.Errorf("load formatter: %w", err)
	}
	contactName, err = formatter.format(&reminder.Contact, i18n.T(locale, "reminder.unknown_contact"))
	if err != nil {
		return "", fmt.Errorf("format contact: %w", err)
	}
	return contactName, nil
}

func reminderDeliveryContent(reminder *models.ContactReminder, scheduledAt time.Time, locale string, enableAltCalendar bool, contactName string) (string, string) {
	date := formatReminderDate(reminder, scheduledAt, enableAltCalendar)
	params := map[string]string{"label": reminder.Label, "contact": contactName, "date": date}
	subject := i18n.Tt(locale, "reminder.subject", params)
	body := i18n.Tt(locale, "reminder.body", params)
	return subject, body
}

func (s *ReminderSchedulerService) sendReminder(channel *models.UserNotificationChannel, subject, body string) error {
	switch channel.Type {
	case "email":
		return s.mailer.Send(channel.Content, subject, body)
	case "shoutrrr", "telegram", "ntfy", "gotify", "webhook":
		if s.sender == nil {
			return fmt.Errorf("notification sender is not configured for channel %d", channel.ID)
		}
		return s.sender.Send(channel.Content, subject, body)
	default:
		return fmt.Errorf("unknown notification channel type %q", channel.Type)
	}
}

func (s *ReminderSchedulerService) handleSuccess(scheduled *models.ContactReminderScheduled, channel *models.UserNotificationChannel, reminder *models.ContactReminder, subject, body string, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.UserNotificationSent{UserNotificationChannelID: channel.ID, SentAt: now, SubjectLine: subject, Payload: &body}).Error; err != nil {
			return fmt.Errorf("create success log: %w", err)
		}
		if err := tx.Model(scheduled).Where("triggered_at IS NULL").Update("triggered_at", now).Error; err != nil {
			return fmt.Errorf("mark schedule triggered: %w", err)
		}
		if channel.Fails > 0 {
			if err := tx.Model(channel).Update("fails", 0).Error; err != nil {
				return fmt.Errorf("reset channel failures: %w", err)
			}
		}
		if err := tx.Model(reminder).Updates(map[string]interface{}{"last_triggered_at": now, "number_times_triggered": gorm.Expr("number_times_triggered + 1")}).Error; err != nil {
			return fmt.Errorf("update reminder tracking: %w", err)
		}
		return rescheduleRecurringReminder(tx, scheduled, channel, reminder)
	})
}

func (s *ReminderSchedulerService) handleFailure(scheduled *models.ContactReminderScheduled, channel *models.UserNotificationChannel, subject, body string, sendErr error, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		errMsg := sendErr.Error()
		if err := tx.Create(&models.UserNotificationSent{UserNotificationChannelID: channel.ID, SentAt: now, SubjectLine: subject, Payload: &body, Error: &errMsg}).Error; err != nil {
			return fmt.Errorf("create failure log: %w", err)
		}
		newFails := channel.Fails + 1
		updates := map[string]interface{}{"fails": newFails}
		if newFails >= maxChannelFails {
			updates["active"] = false
		}
		if err := tx.Model(channel).Updates(updates).Error; err != nil {
			return fmt.Errorf("increment channel failures: %w", err)
		}
		if newFails >= maxChannelFails {
			log.Printf("[reminder-scheduler] Channel %d auto-disabled after %d failures", channel.ID, newFails)
		}
		return nil
	})
}

func rescheduleRecurringReminder(db *gorm.DB, scheduled *models.ContactReminderScheduled, channel *models.UserNotificationChannel, reminder *models.ContactReminder) error {
	if reminder.Type == "one_time" {
		return nil
	}
	location := userLocation(channel.User)
	base := scheduled.ScheduledAt.In(location)
	nextSchedule, ok := nextRecurringSchedule(reminder, channel.PreferredTime, base, location)
	if !ok {
		return nil
	}
	if err := db.Create(&models.ContactReminderScheduled{UserNotificationChannelID: channel.ID, ContactReminderID: reminder.ID, ScheduledAt: nextSchedule.UTC()}).Error; err != nil {
		return fmt.Errorf("create next schedule: %w", err)
	}
	return nil
}

func nextRecurringSchedule(reminder *models.ContactReminder, preferredTime *string, base time.Time, location *time.Location) (time.Time, bool) {
	frequency := 1
	if reminder.FrequencyNumber != nil {
		frequency = *reminder.FrequencyNumber
	}
	var next time.Time
	switch reminder.Type {
	case "recurring_week":
		next = base.AddDate(0, 0, frequency*7)
	case "recurring_month":
		next = base.AddDate(0, frequency, 0)
	case "recurring_year":
		if yearly, ok := calcNextYearlySchedule(reminder, base, location); ok {
			next = yearly
		} else {
			next = base.AddDate(frequency, 0, 0)
		}
	default:
		return time.Time{}, false
	}
	hour, minute := parsePreferredNotificationTime(preferredTime)
	return time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, location), true
}

func calcNextYearlySchedule(reminder *models.ContactReminder, now time.Time, location *time.Location) (time.Time, bool) {
	calendarType := calendarPkg.CalendarType(reminder.CalendarType)
	if calendarType == "" || calendarType == calendarPkg.Gregorian || reminder.OriginalMonth == nil || reminder.OriginalDay == nil {
		return time.Time{}, false
	}
	converter, ok := calendarPkg.Get(calendarType)
	if !ok {
		return time.Time{}, false
	}
	next, err := converter.NextOccurrence(calendarPkg.DateInfo{Day: *reminder.OriginalDay, Month: *reminder.OriginalMonth}, now)
	if err != nil {
		log.Printf("[reminder-scheduler] calendar NextOccurrence failed: %v", err)
		return time.Time{}, false
	}
	return time.Date(next.Year, time.Month(next.Month), next.Day, now.Hour(), now.Minute(), 0, 0, location), true
}
