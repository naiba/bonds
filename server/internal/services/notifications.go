package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
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
			body := fmt.Sprintf("<p>Please verify your notification channel by clicking the link below:</p><p><a href=\"%s\">%s</a></p>", link, link)
			_ = s.mailer.Send(req.Content, "Verify your notification channel", body)
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

	subject := "Test notification"
	body := "<p>This is a test notification from Bonds.</p>"
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
		return fmt.Errorf("no vaultIDs for user %s", userID)
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

		var upcomingDate time.Time
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

		hour, minute := 9, 0
		if channel.PreferredTime != nil {
			fmt.Sscanf(*channel.PreferredTime, "%d:%d", &hour, &minute)
		}
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
