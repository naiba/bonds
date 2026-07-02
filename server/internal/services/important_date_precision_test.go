package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestCreateImportantDate_StoresDatePrecisionVariants(t *testing.T) {
	ctx := setupImportantDateTest(t)

	tests := []struct {
		name      string
		precision string
		day       *int
		month     *int
		year      *int
	}{
		{name: "full date", precision: "full", day: intPtr(15), month: intPtr(6), year: intPtr(1990)},
		{name: "month and year", precision: "month", month: intPtr(6), year: intPtr(1990)},
		{name: "year only", precision: "year", year: intPtr(1990)},
		{name: "month and day", precision: "month_day", day: intPtr(15), month: intPtr(6)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
				Label:         tt.name,
				DatePrecision: tt.precision,
				Day:           tt.day,
				Month:         tt.month,
				Year:          tt.year,
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			if date.DatePrecision != tt.precision {
				t.Fatalf("Expected date_precision %q, got %q", tt.precision, date.DatePrecision)
			}
			assertOptionalInt(t, "day", date.Day, tt.day)
			assertOptionalInt(t, "month", date.Month, tt.month)
			assertOptionalInt(t, "year", date.Year, tt.year)

			var stored models.ContactImportantDate
			if err := ctx.db.First(&stored, date.ID).Error; err != nil {
				t.Fatalf("Load stored important date failed: %v", err)
			}
			if stored.DatePrecision != tt.precision {
				t.Fatalf("Expected stored date_precision %q, got %q", tt.precision, stored.DatePrecision)
			}
		})
	}
}

func TestCreateImportantDate_RejectsPrecisionFieldMismatch(t *testing.T) {
	ctx := setupImportantDateTest(t)

	tests := []struct {
		name      string
		precision string
		day       *int
		month     *int
		year      *int
	}{
		{name: "full date requires day", precision: "full", month: intPtr(6), year: intPtr(1990)},
		{name: "month precision cannot carry day", precision: "month", day: intPtr(15), month: intPtr(6), year: intPtr(1990)},
		{name: "year precision cannot carry month", precision: "year", month: intPtr(6), year: intPtr(1990)},
		{name: "month-day precision cannot carry year", precision: "month_day", day: intPtr(15), month: intPtr(6), year: intPtr(1990)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
				Label:         tt.name,
				DatePrecision: tt.precision,
				Day:           tt.day,
				Month:         tt.month,
				Year:          tt.year,
			})
			if !errors.Is(err, ErrImportantDateInvalidPrecision) {
				t.Fatalf("Expected ErrImportantDateInvalidPrecision, got %v", err)
			}
		})
	}
}

func TestUpdateImportantDate_ChangesPrecisionAndClearsStaleFields(t *testing.T) {
	ctx := setupImportantDateTest(t)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
		Label:         "Birthday",
		DatePrecision: "full",
		Day:           intPtr(15),
		Month:         intPtr(6),
		Year:          intPtr(1990),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, dto.UpdateImportantDateRequest{
		Label:         "Known birth year",
		DatePrecision: "year",
		Year:          intPtr(1990),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.DatePrecision != "year" {
		t.Fatalf("Expected date_precision year, got %q", updated.DatePrecision)
	}
	if updated.Day != nil || updated.Month != nil {
		t.Fatalf("Expected day/month cleared for year precision, got day=%v month=%v", updated.Day, updated.Month)
	}
	if updated.Year == nil || *updated.Year != 1990 {
		t.Fatalf("Expected year 1990, got %v", updated.Year)
	}
}

func TestImportantDate_RemindMe_DoesNotScheduleWithoutDayAndMonth(t *testing.T) {
	ctx := setupImportantDateTest(t)

	tests := []struct {
		name      string
		precision string
		month     *int
		year      *int
	}{
		{name: "year only", precision: "year", year: intPtr(1990)},
		{name: "month and year", precision: "month", month: intPtr(6), year: intPtr(1990)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remindMe := true
			date, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateImportantDateRequest{
				Label:         tt.name,
				DatePrecision: tt.precision,
				Month:         tt.month,
				Year:          tt.year,
				RemindMe:      &remindMe,
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			if date.RemindMe {
				t.Fatalf("Expected remind_me to stay false when no day is available")
			}

			var count int64
			if err := ctx.db.Model(&models.ContactReminder{}).
				Where("important_date_id = ?", date.ID).
				Count(&count).Error; err != nil {
				t.Fatalf("Count reminders failed: %v", err)
			}
			if count != 0 {
				t.Fatalf("Expected 0 reminders for incomplete date, got %d", count)
			}
		})
	}
}

func assertOptionalInt(t *testing.T, name string, got, want *int) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("Expected %s %v, got %v", name, want, got)
		}
		return
	}
	if *got != *want {
		t.Fatalf("Expected %s %d, got %d", name, *want, *got)
	}
}
