package services

import (
	"fmt"
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
)

const calendarProjectionReferenceYear = 2000

func calendarOccurrenceInYear(converter calendarPkg.Converter, original calendarPkg.DateInfo, year int) (calendarPkg.GregorianDate, error) {
	after := time.Date(year-1, time.December, 31, 23, 59, 59, 0, time.UTC)
	gd, err := converter.NextOccurrence(original, after)
	if err != nil {
		return calendarPkg.GregorianDate{}, err
	}
	if gd.Year != year {
		return calendarPkg.GregorianDate{}, fmt.Errorf("calendar occurrence fell outside requested year %d: %+v", year, gd)
	}
	return gd, nil
}

func applyCalendarFields(
	modelCalType *string, modelOrigDay, modelOrigMonth, modelOrigYear **int,
	modelDay, modelMonth, modelYear **int,
	reqCalType string, reqOrigDay, reqOrigMonth, reqOrigYear *int,
) {
	if reqCalType == "" || reqCalType == string(calendarPkg.Gregorian) {
		*modelCalType = string(calendarPkg.Gregorian)
		*modelOrigDay = nil
		*modelOrigMonth = nil
		*modelOrigYear = nil
		return
	}

	converter, ok := calendarPkg.Get(calendarPkg.CalendarType(reqCalType))
	if !ok {
		log.Printf("[calendar] unsupported calendar type %q, falling back to gregorian", reqCalType)
		*modelCalType = string(calendarPkg.Gregorian)
		return
	}

	*modelCalType = reqCalType
	*modelOrigDay = reqOrigDay
	*modelOrigMonth = reqOrigMonth
	*modelOrigYear = reqOrigYear

	if reqOrigDay == nil || reqOrigMonth == nil {
		return
	}

	original := calendarPkg.DateInfo{
		Day:   *reqOrigDay,
		Month: *reqOrigMonth,
	}
	var gd calendarPkg.GregorianDate
	var err error
	conversionYear := calendarProjectionReferenceYear
	if reqOrigYear == nil {
		// Partial alternative-calendar dates recur yearly. Converting them
		// through an arbitrary year with ToGregorian is unsafe: leap months
		// can be absent and a day 30 can fall in a 29-day month. NextOccurrence
		// applies the converter's recurrence/clamping rules and gives us a
		// deterministic projection for storage.
		gd, err = calendarOccurrenceInYear(converter, original, calendarProjectionReferenceYear)
	} else {
		original.Year = *reqOrigYear
		conversionYear = *reqOrigYear
		gd, err = converter.ToGregorian(original)
	}
	if err != nil {
		log.Printf("[calendar] conversion failed for %s date %d/%d/%d: %v", reqCalType, conversionYear, *reqOrigMonth, *reqOrigDay, err)
		return
	}

	*modelDay = &gd.Day
	*modelMonth = &gd.Month
	*modelYear = &gd.Year
}

// applyTimeCalendarFields is the time.Time-shaped sibling of
// applyCalendarFields for models whose date column is a single time.Time
// (Activity.HappenedAt, ContactTask.DueAt, Post.WrittenAt) rather than a
// Day/Month/Year triple. Same contract:
//   - empty / "gregorian" calendar type ⇒ clear Original* and pin model
//     calendar type to "gregorian"; the time.Time column is left as supplied
//     by the request
//   - unsupported calendar type ⇒ same fallback, logged
//   - "lunar" (or any registered converter): persist Original* verbatim, then
//     overwrite the Gregorian projection by running the converter, so that
//     downstream queries (sorting, kanban filtering, calendar view) operate
//     on a date guaranteed consistent with the lunar anchor — the frontend's
//     own projection is treated as advisory, not authoritative
//
// Clock fields (hour/minute/second/nanosecond/location) are preserved from
// modelTime so timezone-bound rows do not silently shift to midnight UTC on
// edit. The converter only emits Y/M/D.
func applyTimeCalendarFields(
	modelCalType *string,
	modelOrigDay, modelOrigMonth, modelOrigYear **int,
	modelTime *time.Time,
	reqCalType string,
	reqOrigDay, reqOrigMonth, reqOrigYear *int,
) {
	if reqCalType == "" || reqCalType == string(calendarPkg.Gregorian) {
		*modelCalType = string(calendarPkg.Gregorian)
		*modelOrigDay = nil
		*modelOrigMonth = nil
		*modelOrigYear = nil
		return
	}

	converter, ok := calendarPkg.Get(calendarPkg.CalendarType(reqCalType))
	if !ok {
		log.Printf("[calendar] unsupported calendar type %q, falling back to gregorian", reqCalType)
		*modelCalType = string(calendarPkg.Gregorian)
		*modelOrigDay = nil
		*modelOrigMonth = nil
		*modelOrigYear = nil
		return
	}

	*modelCalType = reqCalType
	*modelOrigDay = reqOrigDay
	*modelOrigMonth = reqOrigMonth
	*modelOrigYear = reqOrigYear

	if reqOrigDay == nil || reqOrigMonth == nil || modelTime == nil || modelTime.IsZero() {
		return
	}

	year := modelTime.Year()
	if reqOrigYear != nil {
		year = *reqOrigYear
	}

	gd, err := converter.ToGregorian(calendarPkg.DateInfo{
		Day:   *reqOrigDay,
		Month: *reqOrigMonth,
		Year:  year,
	})
	if err != nil {
		log.Printf("[calendar] conversion failed for %s date %d/%d/%d: %v", reqCalType, year, *reqOrigMonth, *reqOrigDay, err)
		return
	}

	*modelTime = time.Date(
		gd.Year, time.Month(gd.Month), gd.Day,
		modelTime.Hour(), modelTime.Minute(), modelTime.Second(), modelTime.Nanosecond(),
		modelTime.Location(),
	)
}
