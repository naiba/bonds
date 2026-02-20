package services

import (
	"strings"
	"testing"
)

func TestNoopSender_RecordsCalls(t *testing.T) {
	sender := &NoopSender{}

	err := sender.Send("ntfy://ntfy.sh/test-topic", "Test Subject", "Test body")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sender.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(sender.Calls))
	}
	if sender.Calls[0].URL != "ntfy://ntfy.sh/test-topic" {
		t.Errorf("expected URL 'ntfy://ntfy.sh/test-topic', got %q", sender.Calls[0].URL)
	}
	if sender.Calls[0].Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got %q", sender.Calls[0].Subject)
	}
}

func TestShoutrrrSender_EmptyURL(t *testing.T) {
	sender := NewShoutrrrSender()

	err := sender.Send("", "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !strings.Contains(err.Error(), "empty shoutrrr URL") {
		t.Errorf("expected 'empty shoutrrr URL' error, got %q", err.Error())
	}
}

func TestShoutrrrSender_WhitespaceURL(t *testing.T) {
	sender := NewShoutrrrSender()

	err := sender.Send("   ", "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for whitespace-only URL")
	}
}

func TestShoutrrrSender_InvalidURL(t *testing.T) {
	sender := NewShoutrrrSender()

	err := sender.Send("not-a-valid-url", "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for invalid shoutrrr URL")
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tags",
			input:    "<h2>Title</h2><p>Body text</p>",
			expected: "TitleBody text",
		},
		{
			name:     "nested tags",
			input:    "<p>Hello <strong>World</strong></p>",
			expected: "Hello World",
		},
		{
			name:     "no tags",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "attributes",
			input:    `<a href="https://example.com">Link</a>`,
			expected: "Link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateURL(t *testing.T) {
	short := "ntfy://x"
	result := truncateURL(short)
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncated URL to contain '...', got %q", result)
	}

	long := "telegram://1234567890:ABCDEFghijklmnop@telegram?channels=123456789"
	result = truncateURL(long)
	if len(result) > 40 {
		t.Errorf("expected truncated URL to be short, got %q (len=%d)", result, len(result))
	}
}

func TestShoutrrrSender_InterfaceCompliance(t *testing.T) {
	var _ NotificationSender = &ShoutrrrSender{}
	var _ NotificationSender = &NoopSender{}
}

func TestNoopSender_MultipleCalls(t *testing.T) {
	sender := &NoopSender{}

	urls := []string{
		"ntfy://ntfy.sh/topic1",
		"gotify://server/token",
		"telegram://bot@telegram?channels=123",
		"generic+https://hooks.example.com/webhook",
	}

	for _, u := range urls {
		if err := sender.Send(u, "Subject", "Message"); err != nil {
			t.Fatalf("unexpected error for URL %q: %v", u, err)
		}
	}

	if len(sender.Calls) != 4 {
		t.Errorf("expected 4 calls, got %d", len(sender.Calls))
	}

	for i, u := range urls {
		if sender.Calls[i].URL != u {
			t.Errorf("call %d: expected URL %q, got %q", i, u, sender.Calls[i].URL)
		}
	}
}
