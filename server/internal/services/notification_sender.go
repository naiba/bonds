package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/containrrr/shoutrrr"
)

// NotificationSender sends notifications through non-email channels (telegram, ntfy, gotify, webhook).
// The destination is a shoutrrr-compatible URL stored in UserNotificationChannel.Content.
type NotificationSender interface {
	Send(shoutrrrURL, subject, message string) error
}

// ShoutrrrSender implements NotificationSender using the shoutrrr library.
type ShoutrrrSender struct{}

// NewShoutrrrSender creates a new ShoutrrrSender.
func NewShoutrrrSender() *ShoutrrrSender {
	return &ShoutrrrSender{}
}

// Send dispatches a notification via the shoutrrr URL.
// The message is formatted as "subject: message" for plain-text channels.
func (s *ShoutrrrSender) Send(shoutrrrURL, subject, message string) error {
	if strings.TrimSpace(shoutrrrURL) == "" {
		return fmt.Errorf("empty shoutrrr URL")
	}

	plainMsg := stripHTML(message)
	body := fmt.Sprintf("%s\n\n%s", subject, plainMsg)

	err := shoutrrr.Send(shoutrrrURL, body)
	if err != nil {
		log.Printf("[notification-sender] shoutrrr send failed for URL prefix %q: %v",
			truncateURL(shoutrrrURL), err)
		return fmt.Errorf("notification send failed: %w", err)
	}
	return nil
}

// NoopSender is a no-op implementation for testing.
type NoopSender struct {
	Calls []NoopSenderCall
}

type NoopSenderCall struct {
	URL, Subject, Message string
}

func (s *NoopSender) Send(shoutrrrURL, subject, message string) error {
	s.Calls = append(s.Calls, NoopSenderCall{URL: shoutrrrURL, Subject: subject, Message: message})
	log.Printf("[NoopSender] Would send via %q subject=%q", truncateURL(shoutrrrURL), subject)
	return nil
}

// truncateURL returns the scheme + first 20 chars for safe logging (no secrets).
func truncateURL(u string) string {
	if len(u) <= 30 {
		return u[:min(len(u), 10)] + "..."
	}
	return u[:30] + "..."
}

// stripHTML removes HTML tags from a string (simple regex-free approach).
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
