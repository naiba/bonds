package services

import (
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
)

// The reminder body is an HTML email. Labels and contact names are free text
// written by any vault member and the mail goes to the other members too, so
// they must not be able to inject markup (links, images, forms) into it.
func TestReminderDeliveryContent_EscapesUserTextInHTMLBody(t *testing.T) {
	reminder := &models.ContactReminder{Label: `<a href="https://phish.example">click</a>`, Type: "one_time"}
	scheduledAt := time.Date(2026, time.March, 10, 9, 0, 0, 0, time.UTC)

	subject, body := reminderDeliveryContent(reminder, scheduledAt, "en", false, `<img src=x onerror=alert(1)> & Co`)

	for _, injected := range []string{"<a href", "<img"} {
		if strings.Contains(body, injected) {
			t.Fatalf("body contains unescaped user markup %q: %s", injected, body)
		}
	}
	for _, escaped := range []string{"&lt;a href=", "&lt;img src=x", "&amp; Co"} {
		if !strings.Contains(body, escaped) {
			t.Fatalf("body does not contain the escaped text %q: %s", escaped, body)
		}
	}
	if !strings.Contains(body, "<strong>") {
		t.Fatalf("the template's own markup must survive: %s", body)
	}
	// The subject is a plain-text header and must stay readable.
	if !strings.Contains(subject, "& Co") {
		t.Fatalf("subject = %q, want the plain contact name", subject)
	}
}
