package services

import (
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

// Clients (API, MCP, imports) have always been allowed to send 0 here, so the
// request keeps working; the meaningless value is stored as "use the default".
func TestReminder_StoresNonPositiveFrequencyAsDefault(t *testing.T) {
	svc, contactID, vaultID := setupReminderTest(t)
	day, month := 15, 6
	zero := 0

	created, err := svc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Call", Day: &day, Month: &month, Type: "recurring_month", FrequencyNumber: &zero,
	})
	if err != nil {
		t.Fatalf("Create with frequency 0 must keep working: %v", err)
	}
	if created.FrequencyNumber != nil {
		t.Fatalf("created frequency_number = %d, want nil (default interval)", *created.FrequencyNumber)
	}

	three, negative := 3, -2
	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateReminderRequest{
		Label: "Call", Day: &day, Month: &month, Type: "recurring_month", FrequencyNumber: &three,
	})
	if err != nil || updated.FrequencyNumber == nil || *updated.FrequencyNumber != 3 {
		t.Fatalf("Update with frequency 3: reminder = %+v, err = %v", updated, err)
	}
	updated, err = svc.Update(created.ID, contactID, vaultID, dto.UpdateReminderRequest{
		Label: "Call", Day: &day, Month: &month, Type: "recurring_month", FrequencyNumber: &negative,
	})
	if err != nil {
		t.Fatalf("Update with a negative frequency must keep working: %v", err)
	}
	if updated.FrequencyNumber != nil {
		t.Fatalf("updated frequency_number = %d, want nil (default interval)", *updated.FrequencyNumber)
	}
}
