package dav

import (
	"strings"
	"testing"

	"github.com/emersion/go-ical"
	"github.com/google/uuid"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/models"
)

// TestLunarImportantDateEmitsRDateNotRRule guards the export of a lunar
// birthday as iCalendar. The old export emitted RRULE=YEARLY on the cached
// Gregorian DTSTART, which causes Apple Calendar / Thunderbird / Google
// Calendar to fire on the *same Gregorian day* every year — wrong for a
// lunar-anchored event, which lands on a different Gregorian day each year.
// The fix replaces RRULE with explicit RDATE entries computed via the
// calendar converter for the next several years' actual Gregorian
// projections, so downstream clients render the event on the right day
// without needing lunar-calendar support of their own.
func TestLunarImportantDateEmitsRDateNotRRule(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Mid", "Autumn")

	uid := uuid.New().String()
	day := 25
	month := 9
	year := 2026
	origDay := 15
	origMonth := 8
	origYear := 2025
	calType := "lunar"
	date := models.ContactImportantDate{
		ContactID:     contact.ID,
		UUID:          &uid,
		Label:         "Lunar Birthday",
		Day:           &day,
		Month:         &month,
		Year:          &year,
		CalendarType:  calType,
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	}
	if err := db.Create(&date).Error; err != nil {
		t.Fatalf("Create lunar date failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListCalendarObjects(ctx, path, nil)
	if err != nil {
		t.Fatalf("ListCalendarObjects failed: %v", err)
	}

	var lunarEvent *ical.Component
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for i, child := range obj.Data.Children {
			summary, _ := child.Props.Text(ical.PropSummary)
			if summary == "Mid Autumn - Lunar Birthday" {
				lunarEvent = obj.Data.Children[i]
				break
			}
		}
	}
	if lunarEvent == nil {
		t.Fatal("did not find Mid Autumn - Lunar Birthday VEVENT")
	}

	if rrule := lunarEvent.Props.Get(ical.PropRecurrenceRule); rrule != nil {
		t.Errorf("lunar event must not carry RRULE (drifts year-over-year), got %q", rrule.Value)
	}

	rdates := lunarEvent.Props.Values(ical.PropRecurrenceDates)
	if len(rdates) == 0 {
		t.Fatal("lunar event must carry at least one RDATE so clients know when to fire")
	}

	converter, ok := calendarPkg.Get(calendarPkg.Lunar)
	if !ok {
		t.Fatal("lunar converter not registered")
	}

	// Verify each RDATE actually matches a converter-derived Gregorian
	// projection of the original lunar date. The format is YYYYMMDD per
	// the ValueDate type, possibly comma-separated within a single line.
	allValues := []string{}
	for _, prop := range rdates {
		for _, v := range strings.Split(prop.Value, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				allValues = append(allValues, v)
			}
		}
	}
	if len(allValues) < 3 {
		t.Errorf("expected at least 3 RDATE entries for a recurring lunar birthday, got %d (%v)", len(allValues), allValues)
	}

	for _, rdate := range allValues {
		if len(rdate) != 8 {
			t.Errorf("RDATE %q not in YYYYMMDD format", rdate)
			continue
		}
	}

	// Cross-check the first RDATE against what the converter would compute
	// for the next occurrence after origYear. This proves the values aren't
	// just hardcoded — they come from the lunar converter.
	wantOrig := calendarPkg.DateInfo{Day: origDay, Month: origMonth, Year: origYear}
	gd, err := converter.ToGregorian(wantOrig)
	if err != nil {
		t.Fatalf("converter.ToGregorian: %v", err)
	}
	wantFirst := wantOrig // for clarity; we expect the YEAR's projection somewhere in the list
	_ = wantFirst
	expected := []string{}
	for y := origYear; y < origYear+5; y++ {
		gd2, err := converter.ToGregorian(calendarPkg.DateInfo{Day: origDay, Month: origMonth, Year: y})
		if err != nil {
			continue
		}
		expected = append(expected, formatYYYYMMDDDay(gd2))
	}
	if !contains(allValues, formatYYYYMMDDDay(gd)) {
		t.Errorf("RDATE list does not contain the first-year projection %s (origYear=%d). got=%v", formatYYYYMMDDDay(gd), origYear, allValues)
	}
	for _, e := range expected[:min(3, len(expected))] {
		if !contains(allValues, e) {
			t.Errorf("RDATE list missing expected lunar projection %s; got=%v", e, allValues)
		}
	}
}

// TestGregorianImportantDateKeepsRRule asserts the Gregorian export path is
// unaffected by the lunar fix — recurring Gregorian birthdays continue to
// emit RRULE=FREQ=YEARLY rather than RDATE, since the same day every year
// is the correct semantics there.
func TestGregorianImportantDateKeepsRRule(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Greg", "Birthday")

	uid := uuid.New().String()
	day := 14
	month := 6
	year := 1990
	calType := "gregorian"
	date := models.ContactImportantDate{
		ContactID:    contact.ID,
		UUID:         &uid,
		Label:        "Gregorian Birthday",
		Day:          &day,
		Month:        &month,
		Year:         &year,
		CalendarType: calType,
	}
	if err := db.Create(&date).Error; err != nil {
		t.Fatalf("Create gregorian date failed: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListCalendarObjects(ctx, path, nil)
	if err != nil {
		t.Fatalf("ListCalendarObjects: %v", err)
	}

	var event *ical.Component
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for i, child := range obj.Data.Children {
			summary, _ := child.Props.Text(ical.PropSummary)
			if summary == "Greg Birthday - Gregorian Birthday" {
				event = obj.Data.Children[i]
				break
			}
		}
	}
	if event == nil {
		t.Fatal("Gregorian event not found")
	}
	rrule := event.Props.Get(ical.PropRecurrenceRule)
	if rrule == nil {
		t.Fatal("Gregorian recurring date must keep RRULE=FREQ=YEARLY")
	}
	if !strings.Contains(rrule.Value, "YEARLY") {
		t.Errorf("Gregorian RRULE = %q; want FREQ=YEARLY", rrule.Value)
	}
	if rdates := event.Props.Values(ical.PropRecurrenceDates); len(rdates) > 0 {
		t.Errorf("Gregorian event should not carry RDATE; got %v", rdates)
	}
}

func formatYYYYMMDDDay(g calendarPkg.GregorianDate) string {
	const pad = "00000000"
	y := intToStr(g.Year, 4)
	m := intToStr(g.Month, 2)
	d := intToStr(g.Day, 2)
	_ = pad
	return y + m + d
}

func intToStr(n, width int) string {
	s := ""
	if n == 0 {
		s = "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string('0'+rune(n%10)) + s
		n /= 10
	}
	for len(s) < width {
		s = "0" + s
	}
	if neg {
		s = "-" + s
	}
	return s
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
