package services

import (
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
)

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

	year := 0
	if reqOrigYear != nil {
		year = *reqOrigYear
	} else if *modelYear != nil {
		year = **modelYear
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

	*modelDay = &gd.Day
	*modelMonth = &gd.Month
	*modelYear = &gd.Year
}

// applyTimeCalendarFields is the time.Time-shaped sibling of
// applyCalendarFields for models whose date column is a single time.Time
// (LifeEvent.HappenedAt, ContactTask.DueAt, Post.WrittenAt) rather than a
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

