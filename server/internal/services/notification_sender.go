package services

import (
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/nicholas-fedor/shoutrrr"
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

	body := formatShoutrrrMessage(subject, message)

	err := shoutrrr.Send(shoutrrrURL, body)
	if err != nil {
		log.Printf("[notification-sender] shoutrrr send failed for URL prefix %q: %v",
			truncateURL(shoutrrrURL), err)
		return fmt.Errorf("notification send failed: %w", err)
	}
	return nil
}

func formatShoutrrrMessage(subject, message string) string {
	subject = strings.TrimSpace(subject)
	plainMessage := stripHTML(message)
	if subject == "" {
		return plainMessage
	}
	if plainMessage == "" {
		return subject
	}
	return subject + "\n\n" + plainMessage
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

// stripHTML removes HTML tags while retaining boundaries between block-level
// elements so adjacent paragraphs and headings do not run together.
func stripHTML(s string) string {
	var result strings.Builder
	for len(s) > 0 {
		start := strings.IndexByte(s, '<')
		if start < 0 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:start])
		end := strings.IndexByte(s[start:], '>')
		if end < 0 {
			result.WriteString(s[start:])
			break
		}
		tag := s[start+1 : start+end]
		if isHTMLBlockTag(tag) {
			result.WriteByte('\n')
		}
		s = s[start+end+1:]
	}

	lines := strings.Split(html.UnescapeString(result.String()), "\n")
	plainLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			plainLines = append(plainLines, line)
		}
	}
	return strings.Join(plainLines, "\n")
}

func isHTMLBlockTag(tag string) bool {
	tag = strings.TrimSpace(strings.TrimPrefix(tag, "/"))
	if tag == "" {
		return false
	}
	name := strings.ToLower(strings.TrimSuffix(strings.Fields(tag)[0], "/"))
	switch name {
	case "address", "article", "aside", "blockquote", "br", "div", "footer", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr", "li", "main", "nav", "ol", "p", "pre", "section", "table", "tr", "ul":
		return true
	default:
		return false
	}
}
