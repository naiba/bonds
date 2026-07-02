package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
)

const (
	importantDatePrecisionFull     = "full"
	importantDatePrecisionMonth    = "month"
	importantDatePrecisionYear     = "year"
	importantDatePrecisionMonthDay = "month_day"
)

var ErrImportantDateInvalidPrecision = errors.New("invalid important date precision")

func applyImportantDatePrecision(date *models.ContactImportantDate, requestedPrecision string) error {
	precision := requestedPrecision
	if precision == "" {
		precision = inferImportantDatePrecision(date)
	}

	switch precision {
	case "":
		date.DatePrecision = ""
		date.IsYearUnknown = false
		return nil
	case importantDatePrecisionFull:
		if date.Day == nil || date.Month == nil || date.Year == nil {
			return ErrImportantDateInvalidPrecision
		}
		date.IsYearUnknown = false
	case importantDatePrecisionMonth:
		if date.Day != nil || date.Month == nil || date.Year == nil {
			return ErrImportantDateInvalidPrecision
		}
		date.IsYearUnknown = false
	case importantDatePrecisionYear:
		if date.Day != nil || date.Month != nil || date.Year == nil {
			return ErrImportantDateInvalidPrecision
		}
		date.IsYearUnknown = false
	case importantDatePrecisionMonthDay:
		if date.Day == nil || date.Month == nil || date.Year != nil {
			return ErrImportantDateInvalidPrecision
		}
		date.IsYearUnknown = true
	default:
		return ErrImportantDateInvalidPrecision
	}

	date.DatePrecision = precision
	clearFieldsOutsideImportantDatePrecision(date)
	return nil
}

func inferImportantDatePrecision(date *models.ContactImportantDate) string {
	if date.Day != nil && date.Month != nil && date.Year != nil {
		return importantDatePrecisionFull
	}
	if date.Day == nil && date.Month != nil && date.Year != nil {
		return importantDatePrecisionMonth
	}
	if date.Day == nil && date.Month == nil && date.Year != nil {
		return importantDatePrecisionYear
	}
	if date.Day != nil && date.Month != nil && date.Year == nil {
		return importantDatePrecisionMonthDay
	}
	return ""
}

func clearFieldsOutsideImportantDatePrecision(date *models.ContactImportantDate) {
	switch date.DatePrecision {
	case importantDatePrecisionFull:
		return
	case importantDatePrecisionMonth:
		date.Day = nil
	case importantDatePrecisionYear:
		date.Day = nil
		date.Month = nil
	case importantDatePrecisionMonthDay:
		date.Year = nil
	}
}

func responseImportantDatePrecision(date *models.ContactImportantDate) string {
	if date.DatePrecision != "" {
		return date.DatePrecision
	}
	if date.IsYearUnknown {
		return importantDatePrecisionMonthDay
	}
	return inferImportantDatePrecision(date)
}

func importantDateCanScheduleReminder(date *models.ContactImportantDate) bool {
	return date.Day != nil && date.Month != nil
}
