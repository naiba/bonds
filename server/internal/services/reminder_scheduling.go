package services

import (
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *ReminderService) reschedulePendingReminder(reminder *models.ContactReminder) {
	if err := reschedulePendingReminder(s.db, reminder); err != nil {
		log.Printf("[reminder] failed to reschedule reminder %d: %v", reminder.ID, err)
	}
}

// BackfillImportantDateReminderSchedules finds important-date reminders missing schedules.
func BackfillImportantDateReminderSchedules(db *gorm.DB) {
	var reminders []models.ContactReminder
	err := db.Where("important_date_id IS NOT NULL").Where("NOT EXISTS (?)", db.Model(&models.ContactReminderScheduled{}).Select("1").Where("contact_reminder_scheduled.contact_reminder_id = contact_reminders.id")).Find(&reminders).Error
	if err != nil {
		log.Printf("[reminder-backfill] query failed: %v", err)
		return
	}
	if len(reminders) == 0 {
		return
	}
	log.Printf("[reminder-backfill] scheduling %d important-date reminders missing schedules", len(reminders))
	for index := range reminders {
		if err := scheduleReminderForVaultUsers(db, &reminders[index]); err != nil {
			log.Printf("[reminder-backfill] schedule reminder %d: %v", reminders[index].ID, err)
		}
	}
}

func reschedulePendingReminder(db *gorm.DB, reminder *models.ContactReminder) error {
	if err := db.Where("contact_reminder_id = ? AND triggered_at IS NULL", reminder.ID).Delete(&models.ContactReminderScheduled{}).Error; err != nil {
		return err
	}
	return scheduleReminderForVaultUsers(db, reminder)
}

func scheduleReminderForVaultUsers(db *gorm.DB, reminder *models.ContactReminder) error {
	var contact models.Contact
	if err := db.First(&contact, "id = ?", reminder.ContactID).Error; err != nil {
		return err
	}
	userIDs, err := reminderAudienceUserIDs(db, reminder, contact.VaultID)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}
	var channels []models.UserNotificationChannel
	if err := db.Preload("User").Where("user_id IN ? AND active = ?", userIDs, true).Find(&channels).Error; err != nil {
		return err
	}
	for index := range channels {
		channel := &channels[index]
		scheduledAt := calcInitialSchedule(reminder, channel.PreferredTime, userLocation(channel.User)).UTC()
		if reminder.Type == "one_time" {
			// 一次性提醒在这个渠道上已经按这个时间发过，就不能再排一次：
			// 编辑提醒（哪怕只改标签）会走到这里，日期没变时再排就是重发。
			var delivered int64
			if err := db.Model(&models.ContactReminderScheduled{}).
				Where("contact_reminder_id = ? AND user_notification_channel_id = ? AND scheduled_at = ? AND triggered_at IS NOT NULL", reminder.ID, channel.ID, scheduledAt).
				Count(&delivered).Error; err != nil {
				return err
			}
			if delivered > 0 {
				continue
			}
		}
		if err := db.Create(&models.ContactReminderScheduled{UserNotificationChannelID: channel.ID, ContactReminderID: reminder.ID, ScheduledAt: scheduledAt}).Error; err != nil {
			return err
		}
	}
	return nil
}

func calcInitialSchedule(reminder *models.ContactReminder, preferredTime *string, location *time.Location) time.Time {
	return calcInitialScheduleAt(reminder, preferredTime, location, time.Now().In(location))
}

func calcInitialScheduleAt(reminder *models.ContactReminder, preferredTime *string, location *time.Location, now time.Time) time.Time {
	hour, minute := parsePreferredNotificationTime(preferredTime)
	calendarType := calendarPkg.CalendarType(reminder.CalendarType)
	if calendarType != "" && calendarType != calendarPkg.Gregorian && reminder.OriginalMonth != nil && reminder.OriginalDay != nil {
		converter, ok := calendarPkg.Get(calendarType)
		if ok {
			originalDate := calendarPkg.DateInfo{}
			reminderOriginalDateForScheduling(&originalDate, *reminder.OriginalMonth, *reminder.OriginalDay, reminder.OriginalYear)
			if reminder.OriginalYear == nil {
				if converted, err := converter.FromGregorian(calendarPkg.GregorianDate{Day: now.Day(), Month: int(now.Month()), Year: now.Year()}); err == nil {
					originalDate.Year = converted.Year
				}
			}
			if gregorianDate, err := converter.NextOccurrence(originalDate, now.AddDate(0, 0, -1)); err == nil {
				return time.Date(gregorianDate.Year, time.Month(gregorianDate.Month), gregorianDate.Day, hour, minute, 0, 0, location)
			}
		}
	}
	year, month, day := now.Year(), time.January, 1
	recurring := reminder.Type != "one_time"
	if reminder.Year != nil && !recurring {
		year = *reminder.Year
	}
	if reminder.Month != nil {
		month = time.Month(*reminder.Month)
	}
	if reminder.Day != nil {
		day = *reminder.Day
	}
	scheduled := time.Date(year, month, day, hour, minute, 0, 0, location)
	if !scheduled.Before(now) {
		return scheduled
	}
	switch reminder.Type {
	case "recurring_week", "recurring_month":
		// 每周 / 每月的提醒要顺着自己的周期推进到下一次。这个函数在每次编辑提醒时
		// 都会被调用：以前这里统一 +1 年，于是锚点日期已过的每周提醒一经编辑，
		// 就要沉默到明年才再触发。
		for scheduled.Before(now) {
			next, ok := nextRecurringSchedule(reminder, preferredTime, scheduled, location)
			if !ok || !next.After(scheduled) {
				break
			}
			scheduled = next
		}
	default:
		if recurring || reminder.Year == nil {
			scheduled = scheduled.AddDate(1, 0, 0)
		}
	}
	return scheduled
}
