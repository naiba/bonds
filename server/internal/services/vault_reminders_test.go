package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultReminderTest(t *testing.T) (*VaultReminderService, *ReminderService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vr-test@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	contactSvc := NewContactService(db)
	contact, _ := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})

	return NewVaultReminderService(db), NewReminderService(db), contact.ID, vault.ID
}

func TestListVaultReminders(t *testing.T) {
	vrSvc, rSvc, contactID, vaultID := setupVaultReminderTest(t)

	day := 15
	month := 6
	_, _ = rSvc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Birthday", Day: &day, Month: &month, Type: "one_time",
	})

	reminders, err := vrSvc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 1 {
		t.Errorf("Expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].Label != "Birthday" {
		t.Errorf("Expected label 'Birthday', got '%s'", reminders[0].Label)
	}
}

func TestListVaultRemindersEmpty(t *testing.T) {
	vrSvc, _, _, vaultID := setupVaultReminderTest(t)

	reminders, err := vrSvc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 0 {
		t.Errorf("Expected 0 reminders, got %d", len(reminders))
	}
}

// Regression for #87: Upcoming Reminders widget previews the next reminders
// chronologically. Service must order by next-occurrence date (ascending),
// not by created_at, so the dashboard glance shows the soonest events first.
func TestListVaultReminders_OrderedByUpcomingDate(t *testing.T) {
	vrSvc, rSvc, contactID, vaultID := setupVaultReminderTest(t)

	now := time.Now()
	// Create three recurring-yearly reminders intentionally inserted in the
	// "wrong" chronological order so created_at != upcoming-date order.
	in60 := now.AddDate(0, 0, 60)
	in10 := now.AddDate(0, 0, 10)
	in30 := now.AddDate(0, 0, 30)

	specs := []struct {
		label string
		t     time.Time
	}{
		{"in60", in60},
		{"in10", in10},
		{"in30", in30},
	}
	for _, s := range specs {
		d, m := s.t.Day(), int(s.t.Month())
		if _, err := rSvc.Create(contactID, vaultID, dto.CreateReminderRequest{
			Label: s.label, Day: &d, Month: &m, Type: "recurring_year",
		}); err != nil {
			t.Fatalf("create %s: %v", s.label, err)
		}
	}

	reminders, err := vrSvc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 3 {
		t.Fatalf("expected 3 reminders, got %d", len(reminders))
	}

	wantOrder := []string{"in10", "in30", "in60"}
	for i, want := range wantOrder {
		if reminders[i].Label != want {
			gotLabels := make([]string, len(reminders))
			for j, r := range reminders {
				gotLabels[j] = r.Label
			}
			t.Fatalf("position %d: want %s, got %s (full order=%v)", i, want, reminders[i].Label, gotLabels)
		}
	}
}
