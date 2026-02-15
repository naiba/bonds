package calendar

import (
	"testing"
	"time"
)

func TestDateInfoIsLeapMonth(t *testing.T) {
	tests := []struct {
		name   string
		month  int
		expect bool
	}{
		{"positive month", 4, false},
		{"negative month (leap)", -4, true},
		{"month 1", 1, false},
		{"month -1", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DateInfo{Month: tt.month}
			if d.IsLeapMonth() != tt.expect {
				t.Errorf("IsLeapMonth() = %v, want %v", d.IsLeapMonth(), tt.expect)
			}
		})
	}
}

func TestDateInfoAbsMonth(t *testing.T) {
	tests := []struct {
		name   string
		month  int
		expect int
	}{
		{"positive", 4, 4},
		{"negative", -4, 4},
		{"one", 1, 1},
		{"negative one", -1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DateInfo{Month: tt.month}
			if d.AbsMonth() != tt.expect {
				t.Errorf("AbsMonth() = %d, want %d", d.AbsMonth(), tt.expect)
			}
		})
	}
}

func TestRegistryAndSupported(t *testing.T) {
	// Gregorian and Lunar are registered via init()
	if !IsSupported(Gregorian) {
		t.Error("Expected Gregorian to be supported")
	}
	if !IsSupported(Lunar) {
		t.Error("Expected Lunar to be supported")
	}
	if IsSupported("buddhist") {
		t.Error("Expected 'buddhist' to not be supported")
	}

	types := SupportedTypes()
	if len(types) < 2 {
		t.Errorf("Expected at least 2 supported types, got %d", len(types))
	}

	c, ok := Get(Gregorian)
	if !ok || c == nil {
		t.Error("Expected to get Gregorian converter")
	}
	c, ok = Get(Lunar)
	if !ok || c == nil {
		t.Error("Expected to get Lunar converter")
	}
	_, ok = Get("unknown")
	if ok {
		t.Error("Expected 'unknown' to not be found")
	}
}

// --- Gregorian Converter Tests ---

func TestGregorianToGregorian(t *testing.T) {
	c, _ := Get(Gregorian)
	gd, err := c.ToGregorian(DateInfo{Day: 15, Month: 6, Year: 2025})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd.Day != 15 || gd.Month != 6 || gd.Year != 2025 {
		t.Errorf("Expected 2025-06-15, got %d-%02d-%02d", gd.Year, gd.Month, gd.Day)
	}
}

func TestGregorianFromGregorian(t *testing.T) {
	c, _ := Get(Gregorian)
	di, err := c.FromGregorian(GregorianDate{Day: 15, Month: 6, Year: 2025})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if di.Day != 15 || di.Month != 6 || di.Year != 2025 {
		t.Errorf("Expected 2025-06-15, got %d-%02d-%02d", di.Year, di.Month, di.Day)
	}
}

func TestGregorianNextOccurrence(t *testing.T) {
	c, _ := Get(Gregorian)

	// Feb 14 recurring, after Jan 1 2026 → should be Feb 14 2026
	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	gd, err := c.NextOccurrence(DateInfo{Day: 14, Month: 2}, after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd.Year != 2026 || gd.Month != 2 || gd.Day != 14 {
		t.Errorf("Expected 2026-02-14, got %d-%02d-%02d", gd.Year, gd.Month, gd.Day)
	}

	// Feb 14 recurring, after Mar 1 2026 → should be Feb 14 2027
	after2 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	gd2, err := c.NextOccurrence(DateInfo{Day: 14, Month: 2}, after2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd2.Year != 2027 || gd2.Month != 2 || gd2.Day != 14 {
		t.Errorf("Expected 2027-02-14, got %d-%02d-%02d", gd2.Year, gd2.Month, gd2.Day)
	}
}

// --- Lunar Converter Tests ---

func TestLunarToGregorian(t *testing.T) {
	c, _ := Get(Lunar)

	// Lunar 2025-01-15 (正月十五, 元宵节) → should be around Feb 12, 2025
	gd, err := c.ToGregorian(DateInfo{Day: 15, Month: 1, Year: 2025})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd.Year != 2025 || gd.Month != 2 {
		t.Errorf("Expected 2025-02-xx, got %d-%02d-%02d", gd.Year, gd.Month, gd.Day)
	}
}

func TestLunarFromGregorian(t *testing.T) {
	c, _ := Get(Lunar)

	// Feb 12, 2025 → should be around Lunar 2025-01-15
	di, err := c.FromGregorian(GregorianDate{Day: 12, Month: 2, Year: 2025})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if di.Year != 2025 || di.Month != 1 {
		t.Errorf("Expected lunar 2025-01-xx, got %d-%02d-%02d", di.Year, di.Month, di.Day)
	}
}

func TestLunarRoundTrip(t *testing.T) {
	c, _ := Get(Lunar)

	original := DateInfo{Day: 15, Month: 1, Year: 2025}
	gd, err := c.ToGregorian(original)
	if err != nil {
		t.Fatalf("ToGregorian error: %v", err)
	}
	back, err := c.FromGregorian(gd)
	if err != nil {
		t.Fatalf("FromGregorian error: %v", err)
	}
	if back.Day != original.Day || back.Month != original.Month || back.Year != original.Year {
		t.Errorf("Round trip failed: %+v → %+v → %+v", original, gd, back)
	}
}

func TestLunarNextOccurrence(t *testing.T) {
	c, _ := Get(Lunar)

	// Lunar 正月十五, after Jan 1 2026 → should find the 2026 occurrence
	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	gd, err := c.NextOccurrence(DateInfo{Day: 15, Month: 1}, after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd.Year != 2026 {
		t.Errorf("Expected year 2026, got %d", gd.Year)
	}
	// Should be in Feb or Mar
	if gd.Month < 1 || gd.Month > 3 {
		t.Errorf("Expected month 1-3 for lunar new year period, got %d", gd.Month)
	}
}

func TestLunarNextOccurrenceAfterDate(t *testing.T) {
	c, _ := Get(Lunar)

	// Lunar 八月十五 (Mid-Autumn), after Nov 1 2026 → should be 2027
	after := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	gd, err := c.NextOccurrence(DateInfo{Day: 15, Month: 8}, after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd.Year != 2027 {
		t.Errorf("Expected year 2027, got %d", gd.Year)
	}
}

func TestLunarDayOverflowHandled(t *testing.T) {
	c, _ := Get(Lunar)

	gd, err := c.ToGregorian(DateInfo{Day: 30, Month: 2, Year: 2025})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gd.Year == 0 {
		t.Error("Expected valid year, got 0")
	}
}

func TestConverterType(t *testing.T) {
	gc, _ := Get(Gregorian)
	if gc.Type() != Gregorian {
		t.Errorf("Expected Gregorian, got %s", gc.Type())
	}

	lc, _ := Get(Lunar)
	if lc.Type() != Lunar {
		t.Errorf("Expected Lunar, got %s", lc.Type())
	}
}
