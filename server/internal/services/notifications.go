package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrNotificationChannelNotFound = errors.New("notification channel not found")
	ErrInvalidVerificationToken    = errors.New("invalid verification token")
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
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
	ch := models.UserNotificationChannel{
		UserID:        &userID,
		Type:          req.Type,
		Label:         strPtrOrNil(req.Label),
		Content:       req.Content,
		PreferredTime: strPtrOrNil(req.PreferredTime),
	}
	if err := s.db.Create(&ch).Error; err != nil {
		return nil, err
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
	return s.db.Save(&ch).Error
}

func (s *NotificationService) SendTest(id uint, userID string) error {
	var ch models.UserNotificationChannel
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotificationChannelNotFound
		}
		return err
	}
	sent := models.UserNotificationSent{
		UserNotificationChannelID: ch.ID,
		SentAt:                    time.Now(),
		SubjectLine:               "Test notification",
	}
	return s.db.Create(&sent).Error
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
