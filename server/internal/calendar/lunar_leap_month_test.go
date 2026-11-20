package calendar

import "testing"

// A date in a leap month exists only in the years that have that leap month
// (2025 has a leap 6th month, 2026 has none). The yearly projections used by
// the calendar export ask for it in every year; the underlying library panics
// on a month that does not exist, which took the whole export down.
func TestLunarToGregorian_LeapMonthDateInYearWithoutThatLeapMonth(t *testing.T) {
	converter, ok := Get(Lunar)
	if !ok {
		t.Fatal("lunar converter is not registered")
	}

	inLeapYear, err := converter.ToGregorian(DateInfo{Year: 2025, Month: -6, Day: 10})
	if err != nil {
		t.Fatalf("leap month in its own year: %v", err)
	}
	regularSixthMonth2025, err := converter.ToGregorian(DateInfo{Year: 2025, Month: 6, Day: 10})
	if err != nil {
		t.Fatalf("regular month: %v", err)
	}
	if inLeapYear == regularSixthMonth2025 {
		t.Fatalf("leap 6th month and regular 6th month of 2025 both map to %+v", inLeapYear)
	}

	// Without that leap month the date is observed in the regular month.
	withoutLeapMonth, err := converter.ToGregorian(DateInfo{Year: 2026, Month: -6, Day: 10})
	if err != nil {
		t.Fatalf("leap month date in a year without it: %v", err)
	}
	regularSixthMonth2026, err := converter.ToGregorian(DateInfo{Year: 2026, Month: 6, Day: 10})
	if err != nil {
		t.Fatalf("regular month: %v", err)
	}
	if withoutLeapMonth != regularSixthMonth2026 {
		t.Fatalf("leap month date in 2026 = %+v, want the regular 6th month %+v", withoutLeapMonth, regularSixthMonth2026)
	}
}

func TestLunarToGregorian_ReportsImpossibleDateInsteadOfPanicking(t *testing.T) {
	converter, _ := Get(Lunar)

	if _, err := converter.ToGregorian(DateInfo{Year: 2026, Month: 13, Day: 1}); err == nil {
		t.Fatal("month 13 was converted without an error")
	}
}
