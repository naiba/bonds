package services

import (
	"errors"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
)

var ErrReminderInvalidDate = errors.New("invalid reminder date")

func validateAndApplyReminderDate(day **int, month **int, year **int, calendarType *string, originalDay **int, originalMonth **int, originalYear **int,
	requestedCalendarType string, requestedOriginalDay *int, requestedOriginalMonth *int, requestedOriginalYear *int) error {
	applyCalendarFields(calendarType, originalDay, originalMonth, originalYear, day, month, year,
		requestedCalendarType, requestedOriginalDay, requestedOriginalMonth, requestedOriginalYear)

	if *day == nil || *month == nil {
		return ErrReminderInvalidDate
	}
	if !isValidReminderMonth(**month) {
		return ErrReminderInvalidDate
	}

	if *year == nil {
		if !isValidReminderMonthDay(**month, **day) {
			return ErrReminderInvalidDate
		}
		return nil
	}

	if !isValidReminderDay(**year, **month, **day) {
		return ErrReminderInvalidDate
	}
	return nil
}

func isValidReminderMonth(month int) bool {
	return month >= 1 && month <= 12
}

func isValidReminderDay(year, month, day int) bool {
	if day < 1 {
		return false
	}
	lastDayOfMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return day <= lastDayOfMonth
}

func isValidReminderMonthDay(month, day int) bool {
	return isValidReminderDay(2000, month, day)
}

func inferReminderPrecision(day, month, year *int) string {
	if day != nil && month != nil && year == nil {
		return "month_day"
	}
	return "full"
}

func reminderOriginalDateForScheduling(reminder *calendarPkg.DateInfo, month, day int, year *int) {
	reminder.Month = month
	reminder.Day = day
	if year != nil {
		reminder.Year = *year
	}
}
