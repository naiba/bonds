package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// TelegramService sends messages via the Telegram Bot API.
// If the bot token is empty, NewTelegramService returns nil (telegram disabled).
type TelegramService struct {
	botToken string
	client   *http.Client
	apiBase  string
}

// NewTelegramService creates a TelegramService. Returns nil, nil if botToken is empty.
func NewTelegramService(botToken string) (*TelegramService, error) {
	if botToken == "" {
		return nil, nil
	}
	return &TelegramService{
		botToken: botToken,
		client:   &http.Client{},
		apiBase:  "https://api.telegram.org",
	}, nil
}

// IsAvailable returns true if the bot is configured.
func (s *TelegramService) IsAvailable() bool {
	return s != nil
}

// SendMessage sends a plain text message to a Telegram chat.
func (s *TelegramService) SendMessage(chatID int64, text string) error {
	if s == nil {
		return nil
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", s.apiBase, s.botToken)
	resp, err := s.client.PostForm(endpoint, url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"text":       {text},
		"parse_mode": {"HTML"},
	})
	if err != nil {
		return fmt.Errorf("telegram send failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("telegram API error: %s", result.Description)
	}
	return nil
}

// SendReminder formats and sends a reminder notification.
func (s *TelegramService) SendReminder(chatID int64, contactName, reminderLabel string) error {
	text := FormatReminderMessage(contactName, reminderLabel)
	return s.SendMessage(chatID, text)
}

// FormatReminderMessage builds the reminder notification text.
func FormatReminderMessage(contactName, reminderLabel string) string {
	return fmt.Sprintf("🔔 <b>Reminder</b>\n\n<b>%s</b>: %s", contactName, reminderLabel)
}
