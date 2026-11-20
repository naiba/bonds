package services

import (
	"log"
	"sync"
	"time"

	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	userTimezone "github.com/naiba/bonds/internal/timezone"
	"gorm.io/gorm"
)

const maxChannelFails = 10

type ReminderSchedulerService struct {
	db     *gorm.DB
	mailer Mailer
	sender NotificationSender
	// processing 保证同一进程内不会有两轮处理同时进行，见 ProcessDueReminders。
	processing sync.Mutex
}

func NewReminderSchedulerService(db *gorm.DB, mailer Mailer, sender NotificationSender) *ReminderSchedulerService {
	return &ReminderSchedulerService{db: db, mailer: mailer, sender: sender}
}

// ProcessDueReminders finds all scheduled reminders that are due and processes them.
func (s *ReminderSchedulerService) ProcessDueReminders() {
	// 这个任务每分钟触发一次，而“已发送”要等发送完成才落库。上一轮还在发
	// （SMTP 变慢、到期提醒很多）时下一轮查到的是同一批未发送的记录，两轮都会发。
	// 上一轮没结束就直接跳过这一轮，到期的提醒下一分钟仍会被处理。
	if !s.processing.TryLock() {
		log.Printf("[reminder-scheduler] Previous run is still sending, skipping this tick")
		return
	}
	defer s.processing.Unlock()

	now := time.Now().UTC().Truncate(time.Minute)

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

// userLocation returns the time.Location for the given user's saved
// timezone preference, falling back to UTC if the user is nil, has no
// timezone set, or the saved string fails to load. Callers should pass
// this to any date math that ends up persisted (scheduled fire times,
// reminder dates) so different users in the same vault don't all fire at
// the server's wall-clock 09:00.
func userLocation(user *models.User) *time.Location {
	if user == nil {
		return userTimezone.Location(nil)
	}
	return userTimezone.Location(user.Timezone)
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
