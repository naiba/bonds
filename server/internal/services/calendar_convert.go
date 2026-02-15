package services

import (
	"log"

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
