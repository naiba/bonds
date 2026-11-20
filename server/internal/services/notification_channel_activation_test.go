package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

type verificationMailRecorder struct {
	recipients []string
}

func (m *verificationMailRecorder) Send(to, _, _ string) error {
	m.recipients = append(m.recipients, to)
	return nil
}

func (m *verificationMailRecorder) Close() {}

// Reminder text is written by the user. If an email channel could be switched
// on without proving ownership of the address, any account could make the
// instance mail arbitrary content to arbitrary people.
func TestNotificationToggle_DoesNotActivateUnverifiedForeignAddress(t *testing.T) {
	svc, userID := setupNotificationTest(t)
	mailer := &verificationMailRecorder{}
	svc.SetMailer(mailer)
	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{Type: "email", Content: "someone-else@example.com"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	mailer.recipients = nil

	_, err = svc.Toggle(created.ID, userID)

	if !errors.Is(err, ErrNotificationChannelNotVerified) {
		t.Fatalf("Toggle err = %v, want ErrNotificationChannelNotVerified", err)
	}
	var stored models.UserNotificationChannel
	if err := svc.db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("reload channel: %v", err)
	}
	if stored.Active || stored.VerifiedAt != nil {
		t.Fatalf("channel active=%v verified_at=%v, want it to stay inactive and unverified", stored.Active, stored.VerifiedAt)
	}
	// The owner still needs a way forward: the verification mail is sent again.
	if len(mailer.recipients) != 1 || mailer.recipients[0] != "someone-else@example.com" {
		t.Fatalf("verification mails sent to %v, want exactly one to the channel address", mailer.recipients)
	}
}

// The account's own verified login address needs no second proof. This is also
// what keeps channels created before SMTP was configured usable.
func TestNotificationToggle_ActivatesTheAccountsOwnVerifiedAddress(t *testing.T) {
	svc, userID := setupNotificationTest(t)
	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{Type: "email", Content: "Notifications-Test@example.com"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	toggled, err := svc.Toggle(created.ID, userID)

	if err != nil {
		t.Fatalf("Toggle failed: %v", err)
	}
	if !toggled.Active {
		t.Fatal("the account's own verified address should activate")
	}
}

// A channel that was switched off after repeated failures must get a fresh
// allowance when its owner turns it back on; otherwise the very next failure
// disables it again.
func TestNotificationToggle_ReenablingClearsTheFailureCount(t *testing.T) {
	svc, userID := setupNotificationTest(t)
	created, err := svc.Create(userID, dto.CreateNotificationChannelRequest{Type: "ntfy", Content: "ntfy://ntfy.example/topic"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := svc.db.Model(&models.UserNotificationChannel{}).Where("id = ?", created.ID).
		Updates(map[string]interface{}{"active": false, "fails": maxChannelFails}).Error; err != nil {
		t.Fatalf("simulate auto-disable: %v", err)
	}

	if _, err := svc.Toggle(created.ID, userID); err != nil {
		t.Fatalf("Toggle failed: %v", err)
	}

	var stored models.UserNotificationChannel
	if err := svc.db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("reload channel: %v", err)
	}
	if !stored.Active || stored.Fails != 0 {
		t.Fatalf("channel active=%v fails=%d, want active with fails reset to 0", stored.Active, stored.Fails)
	}
}
