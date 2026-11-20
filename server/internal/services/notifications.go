package services

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrNotificationChannelNotFound = errors.New("notification channel not found")
	ErrInvalidVerificationToken    = errors.New("invalid verification token")
	// ErrNotificationChannelNotVerified 表示邮件渠道的地址尚未验证，不能启用。
	ErrNotificationChannelNotVerified = errors.New("notification channel is not verified")
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
			locale := s.loadUserLocale(userID)
			subject, body := notificationVerificationContent(s.getAppURL(), ch.ID, token, locale)
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
			locale := s.loadUserLocale(userID)
			subject, body := notificationVerificationContent(s.getAppURL(), ch.ID, *ch.VerificationToken, locale)
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
	if !ch.Active {
		// 邮件渠道必须先证明地址归属才能启用。提醒内容由用户自己填写，
		// 不验证就能启用的话，任何账号都能让本实例向任意邮箱发送任意内容。
		if ch.Type == "email" && ch.VerifiedAt == nil {
			verified, err := s.verifyOwnAccountAddress(&ch, userID)
			if err != nil {
				return nil, err
			}
			if !verified {
				s.resendChannelVerification(&ch, userID)
				return nil, ErrNotificationChannelNotVerified
			}
		}
		// 连续失败被自动停用的渠道，手动重新启用时要清零失败计数，
		// 否则下一次失败就会立刻再次被停用。
		ch.Fails = 0
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
	return scheduleAllContactReminders(s.db, channelID, userID)
}

// verifyOwnAccountAddress 把“地址就是该用户已验证的登录邮箱”视为已证明归属，
// 并把渠道标记为已验证。这也是 SMTP 配置之前创建的渠道（当时发不出验证邮件）
// 仍能启用的途径。
func (s *NotificationService) verifyOwnAccountAddress(ch *models.UserNotificationChannel, userID string) (bool, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}
	if user.EmailVerifiedAt == nil || !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(ch.Content)) {
		return false, nil
	}
	now := time.Now()
	ch.VerifiedAt = &now
	return true, nil
}

// resendChannelVerification 给尚未验证的邮件渠道重发验证邮件，沿用已有的验证令牌。
func (s *NotificationService) resendChannelVerification(ch *models.UserNotificationChannel, userID string) {
	if s.mailer == nil {
		return
	}
	if ch.VerificationToken == nil {
		token := uuid.New().String()
		if err := s.db.Model(ch).Update("verification_token", token).Error; err != nil {
			return
		}
		ch.VerificationToken = &token
	}
	subject, body := notificationVerificationContent(s.getAppURL(), ch.ID, *ch.VerificationToken, s.loadUserLocale(userID))
	_ = s.mailer.Send(ch.Content, subject, body)
}
