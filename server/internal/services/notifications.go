package services

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrNotificationChannelNotFound = errors.New("notification channel not found")
	ErrInvalidVerificationToken    = errors.New("invalid verification token")
)

type NotificationService struct {
	db       *gorm.DB
	mailer   Mailer
	sender   NotificationSender
	settings *SystemSettingService
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

func (s *NotificationService) SetMailer(mailer Mailer) {
	s.mailer = mailer
}

func (s *NotificationService) SetSender(sender NotificationSender) {
	s.sender = sender
}

func (s *NotificationService) SetSystemSettings(settings *SystemSettingService) {
	s.settings = settings
}

func (s *NotificationService) getAppURL() string {
	if s.settings != nil {
		return s.settings.GetWithDefault("app.url", "http://localhost:8080")
	}
	return "http://localhost:8080"
}

// loadUserLocale returns the user's persisted locale, falling back to "en"
// when the user row is missing or the column is empty. Used to localize
// notification emails the user themselves will read (verify-channel,
// send-test). The locale is loaded fresh on every call rather than threaded
// through the call chain so changes to Preferences.locale take effect on
// the next email without needing every caller plumbing.
func (s *NotificationService) loadUserLocale(userID string) string {
	var user models.User
	if err := s.db.Select("locale").Where("id = ?", userID).First(&user).Error; err != nil {
		return "en"
	}
	if user.Locale == "" {
		return "en"
	}
	return user.Locale
}

func (s *NotificationService) List(userID string) ([]dto.NotificationChannelResponse, error) {
	var channels []models.UserNotificationChannel
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&channels).Error; err != nil {
		return nil, err
	}
	result := make([]dto.NotificationChannelResponse, len(channels))
	for i, ch := range channels {
		result[i] = toNotificationChannelResponse(&ch)
	}
	return result, nil
}

func (s *NotificationService) Create(userID string, req dto.CreateNotificationChannelRequest) (*dto.NotificationChannelResponse, error) {
	token := uuid.New().String()
	ch := models.UserNotificationChannel{
		UserID:            &userID,
		Type:              req.Type,
		Label:             strPtrOrNil(req.Label),
		Content:           req.Content,
		PreferredTime:     strPtrOrNil(req.PreferredTime),
		VerificationToken: &token,
	}
	if err := s.db.Create(&ch).Error; err != nil {
		return nil, err
	}

	if req.Type == "email" {
		if s.mailer != nil {
			link := fmt.Sprintf("%s/settings/notifications/%d/verify/%s", s.getAppURL(), ch.ID, token)
			locale := s.loadUserLocale(userID)
			subject := i18n.T(locale, "notification.channel.verify.subject")
			body := i18n.Tt(locale, "notification.channel.verify.body", map[string]string{"link": link})
			_ = s.mailer.Send(req.Content, subject, body)
		}
	} else {
		now := time.Now()
		ch.VerifiedAt = &now
		ch.Active = true
		if err := s.db.Model(&ch).Updates(map[string]interface{}{
			"verified_at": now,
			"active":      true,
		}).Error; err != nil {
			return nil, err
		}
		if err := s.ScheduleAllContactReminders(ch.ID, userID); err != nil {
			return nil, err
		}
	}

	resp := toNotificationChannelResponse(&ch)
	return &resp, nil
}

func (s *NotificationService) Update(id uint, userID string, req dto.UpdateNotificationChannelRequest) (*dto.NotificationChannelResponse, error) {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationChannelNotFound
		}
		return nil, err
	}

	contentChanged := ch.Content != req.Content
	preferredTimeChanged := ptrToStr(ch.PreferredTime) != req.PreferredTime

	ch.Label = strPtrOrNil(req.Label)
	ch.Content = req.Content
	ch.PreferredTime = strPtrOrNil(req.PreferredTime)

	if contentChanged && ch.Type == "email" {
		token := uuid.New().String()
		ch.VerificationToken = &token
		ch.VerifiedAt = nil
		ch.Active = false
		ch.Fails = 0
	}

	if err := s.db.Save(&ch).Error; err != nil {
		return nil, err
	}

	if contentChanged && ch.Type == "email" {
		if s.mailer != nil {
			link := fmt.Sprintf("%s/settings/notifications/%d/verify/%s", s.getAppURL(), ch.ID, *ch.VerificationToken)
			locale := s.loadUserLocale(userID)
			subject := i18n.T(locale, "notification.channel.verify.subject")
			body := i18n.Tt(locale, "notification.channel.verify.body", map[string]string{"link": link})
			_ = s.mailer.Send(req.Content, subject, body)
		}
	}
	if ch.Active && preferredTimeChanged {
		if err := s.db.Where("user_notification_channel_id = ? AND triggered_at IS NULL", ch.ID).
			Delete(&models.ContactReminderScheduled{}).Error; err != nil {
			return nil, err
		}
		if err := s.ScheduleAllContactReminders(ch.ID, userID); err != nil {
			return nil, err
		}
	}

	resp := toNotificationChannelResponse(&ch)
	return &resp, nil
}

func (s *NotificationService) Toggle(id uint, userID string) (*dto.NotificationChannelResponse, error) {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationChannelNotFound
		}
		return nil, err
	}
	ch.Active = !ch.Active
	if err := s.db.Save(&ch).Error; err != nil {
		return nil, err
	}
	if ch.Active {
		_ = s.ScheduleAllContactReminders(ch.ID, userID)
	}
	resp := toNotificationChannelResponse(&ch)
	return &resp, nil
}

func (s *NotificationService) Delete(id uint, userID string) error {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotificationChannelNotFound
		}
		return err
	}
	return s.db.Delete(&ch).Error
}

func (s *NotificationService) Verify(id uint, userID, token string) error {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotificationChannelNotFound
		}
		return err
	}
	if ch.VerificationToken == nil || *ch.VerificationToken != token {
		return ErrInvalidVerificationToken
	}
	now := time.Now()
	ch.VerifiedAt = &now
	ch.Active = true
	if err := s.db.Save(&ch).Error; err != nil {
		return err
	}
	return s.ScheduleAllContactReminders(ch.ID, *ch.UserID)
}

func (s *NotificationService) SendTest(id uint, userID string) error {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotificationChannelNotFound
		}
		return err
	}

	locale := s.loadUserLocale(userID)
	subject := i18n.T(locale, "notification.channel.test.subject")
	body := i18n.T(locale, "notification.channel.test.body")
	var sendErr error

	switch ch.Type {
	case "email":
		if s.mailer != nil {
			sendErr = s.mailer.Send(ch.Content, subject, body)
		}
	case "shoutrrr", "telegram", "ntfy", "gotify", "webhook":
		if s.sender != nil {
			sendErr = s.sender.Send(ch.Content, subject, body)
		}
	}

	now := time.Now()
	sent := models.UserNotificationSent{
		UserNotificationChannelID: ch.ID,
		SentAt:                    now,
		SubjectLine:               subject,
		Payload:                   &body,
	}
	if sendErr != nil {
		errMsg := sendErr.Error()
		sent.Error = &errMsg
	}
	if err := s.db.Create(&sent).Error; err != nil {
		return err
	}
	return sendErr
}

func (s *NotificationService) ListLogs(id uint, userID string) ([]dto.NotificationLogResponse, error) {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationChannelNotFound
		}
		return nil, err
	}
	var logs []models.UserNotificationSent
	if err := s.db.Where("user_notification_channel_id = ?", id).Order("sent_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	result := make([]dto.NotificationLogResponse, len(logs))
	for i, l := range logs {
		result[i] = dto.NotificationLogResponse{
			ID:          l.ID,
			SentAt:      l.SentAt,
			SubjectLine: l.SubjectLine,
			Payload:     ptrToStr(l.Payload),
			Error:       ptrToStr(l.Error),
			CreatedAt:   l.CreatedAt,
		}
	}
	return result, nil
}

func (s *NotificationService) ScheduleAllContactReminders(channelID uint, userID string) error {
	var channel models.UserNotificationChannel
	if err := s.db.Preload("User").First(&channel, channelID).Error; err != nil {
		return fmt.Errorf("channel load: %w", err)
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user load: %w", err)
	}

	var vaultIDs []string
	if err := s.db.Model(&models.UserVault{}).
		Where("user_id = ?", userID).
		Pluck("vault_id", &vaultIDs).Error; err != nil {
		return fmt.Errorf("vault pluck: %w", err)
	}
	if len(vaultIDs) == 0 {
		return nil
	}

	var contactIDs []string
	if err := s.db.Model(&models.Contact{}).
		Where("vault_id IN ?", vaultIDs).
		Pluck("id", &contactIDs).Error; err != nil {
		return err
	}
	if len(contactIDs) == 0 {
		return nil
	}

	var reminders []models.ContactReminder
	if err := s.db.Where("contact_id IN ?", contactIDs).Find(&reminders).Error; err != nil {
		return err
	}

	now := time.Now()
	tz := "UTC"
	if user.Timezone != nil && *user.Timezone != "" {
		tz = *user.Timezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	for _, r := range reminders {
		var existing models.ContactReminderScheduled
		err := s.db.Where("user_notification_channel_id = ? AND contact_reminder_id = ? AND triggered_at IS NULL",
			channelID, r.ID).First(&existing).Error
		if err == nil {
			continue
		}

		// nextLunarOccurrence returns a zero Time when the reminder isn't
		// lunar or the converter is unavailable — fall back to the cached
		// Gregorian fields. Keeping the two paths separated rather than
		// merged so a future-me reading this can see "lunar uses converter,
		// gregorian uses cached fields" without unwinding shared variables.
		var upcomingDate time.Time
		if lunarDate, ok := nextLunarOccurrence(&r, now, loc); ok {
			upcomingDate = lunarDate
		} else {
			month := 1
			day := 1
			if r.Month != nil {
				month = *r.Month
			}
			if r.Day != nil {
				day = *r.Day
			}
			if r.Year == nil || *r.Year == 0 {
				upcomingDate = time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, loc)
			} else {
				upcomingDate = time.Date(*r.Year, time.Month(month), day, 0, 0, 0, 0, loc)
			}

			if upcomingDate.Before(now) {
				upcomingDate = time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, loc)
				if upcomingDate.Before(now) {
					upcomingDate = upcomingDate.AddDate(1, 0, 0)
				}
			}
		}

		hour, minute := parsePreferredNotificationTime(channel.PreferredTime)
		upcomingDate = time.Date(upcomingDate.Year(), upcomingDate.Month(), upcomingDate.Day(), hour, minute, 0, 0, loc)

		scheduled := models.ContactReminderScheduled{
			UserNotificationChannelID: channelID,
			ContactReminderID:         r.ID,
			ScheduledAt:               upcomingDate.UTC(),
		}
		if err := s.db.Create(&scheduled).Error; err != nil {
			return fmt.Errorf("create scheduled reminder: %w", err)
		}
	}

	return nil
}

func parsePreferredNotificationTime(preferredTime *string) (int, int) {
	if preferredTime == nil {
		return 9, 0
	}
	raw := strings.TrimSpace(*preferredTime)
	if len(raw) != 5 || raw[2] != ':' {
		return 9, 0
	}
	hour, hourErr := strconv.Atoi(raw[:2])
	minute, minuteErr := strconv.Atoi(raw[3:])
	// Avoid fmt.Sscanf partial parses: invalid HH:MM must never overflow time.Date.
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 9, 0
	}
	return hour, minute
}

// nextLunarOccurrence resolves the next Gregorian fire date for a lunar
// (or other non-Gregorian) reminder using its OriginalMonth/OriginalDay
// metadata. Without this, ScheduleAllContactReminders would naively reuse
// the cached Gregorian projection from whichever year the reminder was
// first created — and lunar dates drift year over year, so the channel
// would fire on the wrong day each time it was reactivated.
//
// Returns ok=false for Gregorian reminders and when the converter cannot
// resolve the next date — callers should fall back to the cached fields.
func nextLunarOccurrence(r *models.ContactReminder, now time.Time, loc *time.Location) (time.Time, bool) {
	ct := calendarPkg.CalendarType(r.CalendarType)
	if ct == "" || ct == calendarPkg.Gregorian {
		return time.Time{}, false
	}
	if r.OriginalMonth == nil || r.OriginalDay == nil {
		return time.Time{}, false
	}
	converter, ok := calendarPkg.Get(ct)
	if !ok {
		return time.Time{}, false
	}
	orig := calendarPkg.DateInfo{Day: *r.OriginalDay, Month: *r.OriginalMonth}
	if r.OriginalYear != nil {
		orig.Year = *r.OriginalYear
	}
	gd, err := converter.NextOccurrence(orig, now.AddDate(0, 0, -1))
	if err != nil {
		log.Printf("[notifications] lunar NextOccurrence failed for reminder %d: %v", r.ID, err)
		return time.Time{}, false
	}
	return time.Date(gd.Year, time.Month(gd.Month), gd.Day, 0, 0, 0, 0, loc), true
}

func toNotificationChannelResponse(ch *models.UserNotificationChannel) dto.NotificationChannelResponse {
	return dto.NotificationChannelResponse{
		ID:            ch.ID,
		Type:          ch.Type,
		Label:         ptrToStr(ch.Label),
		Content:       ch.Content,
		PreferredTime: ptrToStr(ch.PreferredTime),
		Active:        ch.Active,
		VerifiedAt:    ch.VerifiedAt,
		CreatedAt:     ch.CreatedAt,
		UpdatedAt:     ch.UpdatedAt,
	}
}
