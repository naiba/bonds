package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrInvalidTelegramUpdate   = errors.New("invalid telegram update")
	ErrTelegramChannelNotFound = errors.New("telegram notification channel not found")
)

type TelegramWebhookService struct {
	db *gorm.DB
}

func NewTelegramWebhookService(db *gorm.DB) *TelegramWebhookService {
	return &TelegramWebhookService{db: db}
}

type TelegramUpdate struct {
	Message *TelegramMessage `json:"message"`
}

type TelegramMessage struct {
	Chat TelegramChat `json:"chat"`
	Text string       `json:"text"`
}

type TelegramChat struct {
	ID int64 `json:"id"`
}

func (s *TelegramWebhookService) HandleUpdate(update TelegramUpdate) error {
	if update.Message == nil {
		return ErrInvalidTelegramUpdate
	}

	text := strings.TrimSpace(update.Message.Text)
	if !strings.HasPrefix(text, "/start ") {
		return nil
	}

	token := strings.TrimPrefix(text, "/start ")
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	var ch models.UserNotificationChannel
	if err := s.db.Where("type = ? AND verification_token = ?", "telegram", token).First(&ch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTelegramChannelNotFound
		}
		return err
	}

	chatIDStr := fmt.Sprintf("%d", update.Message.Chat.ID)
	ch.Content = chatIDStr
	now := time.Now()
	ch.VerifiedAt = &now
	ch.Active = true

	return s.db.Save(&ch).Error
}

func (s *TelegramWebhookService) CreateChannel(userID string) (*models.UserNotificationChannel, string, error) {
	token := strconv.FormatInt(time.Now().UnixNano(), 36)
	ch := models.UserNotificationChannel{
		UserID:            &userID,
		Type:              "telegram",
		Content:           "",
		VerificationToken: &token,
	}
	if err := s.db.Create(&ch).Error; err != nil {
		return nil, "", err
	}
	return &ch, token, nil
}
