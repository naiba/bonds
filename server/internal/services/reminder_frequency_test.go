package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

// With a frequency of 0 (or less) the "next" occurrence equals the current
// one, so the scheduler re-sends the same reminder on every tick.
func TestNextRecurringSchedule_NonPositiveFrequencyStillMovesForward(t *testing.T) {
	base := time.Date(2026, time.March, 10, 9, 0, 0, 0, time.UTC)
	for _, reminderType := range []string{"recurring_week", "recurring_month", "recurring_year"} {
		for _, frequency := range []int{0, -3} {
			reminder := &models.ContactReminder{Type: reminderType, FrequencyNumber: &frequency}

			next, ok := nextRecurringSchedule(reminder, nil, base, time.UTC)

			if !ok {
				t.Fatalf("%s frequency %d: no next schedule", reminderType, frequency)
			}
			if !next.After(base.Add(24 * time.Hour)) {
				t.Errorf("%s frequency %d: next = %s, want a later occurrence than %s", reminderType, frequency, next, base)
			}
		}
	}
}

func TestReminder_RejectsNonPositiveFrequency(t *testing.T) {
	svc, contactID, vaultID := setupReminderTest(t)
	day, month := 15, 6
	zero := 0

	_, err := svc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Call", Day: &day, Month: &month, Type: "recurring_month", FrequencyNumber: &zero,
	})
	if !errors.Is(err, ErrReminderInvalidFrequency) {
		t.Fatalf("Create with frequency 0: err = %v, want ErrReminderInvalidFrequency", err)
	}

	one := 1
	created, err := svc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Call", Day: &day, Month: &month, Type: "recurring_month", FrequencyNumber: &one,
	})
	if err != nil {
		t.Fatalf("Create with frequency 1 failed: %v", err)
	}
	_, err = svc.Update(created.ID, contactID, vaultID, dto.UpdateReminderRequest{
		Label: "Call", Day: &day, Month: &month, Type: "recurring_month", FrequencyNumber: &zero,
	})
	if !errors.Is(err, ErrReminderInvalidFrequency) {
		t.Fatalf("Update with frequency 0: err = %v, want ErrReminderInvalidFrequency", err)
	}
}
