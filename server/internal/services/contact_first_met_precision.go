package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/models"
)

const (
	contactFirstMetPrecisionFull  = "full"
	contactFirstMetPrecisionMonth = "month"
	contactFirstMetPrecisionYear  = "year"
)

var ErrContactInvalidFirstMetPrecision = errors.New("invalid contact first met precision")

func applyContactFirstMet(reqFirstMetAt *time.Time, reqPrecision *string, reqYear *int, reqMonth *int, reqDay *int, contact *models.Contact) error {
	if reqFirstMetAt != nil {
		contact.FirstMetAt = reqFirstMetAt
		fullPrecision := contactFirstMetPrecisionFull
		contact.FirstMetDatePrecision = &fullPrecision
		contact.FirstMetYear = nil
		contact.FirstMetMonth = nil
		contact.FirstMetDay = nil
		return nil
	}

	precision := ""
	if reqPrecision != nil {
		precision = *reqPrecision
	}

	if precision == "" {
		contact.FirstMetAt = nil
		contact.FirstMetDatePrecision = nil
		contact.FirstMetYear = nil
		contact.FirstMetMonth = nil
		contact.FirstMetDay = nil
		return nil
	}

	switch precision {
	case contactFirstMetPrecisionYear:
		if reqYear == nil || reqMonth != nil || reqDay != nil {
			return ErrContactInvalidFirstMetPrecision
		}
	case contactFirstMetPrecisionMonth:
		if reqYear == nil || reqMonth == nil || reqDay != nil {
			return ErrContactInvalidFirstMetPrecision
		}
		if !isValidMonth(*reqMonth) {
			return ErrContactInvalidFirstMetPrecision
		}
	case contactFirstMetPrecisionFull:
		if reqYear == nil || reqMonth == nil || reqDay == nil {
			return ErrContactInvalidFirstMetPrecision
		}
		if !isValidMonth(*reqMonth) || !isValidDay(*reqYear, *reqMonth, *reqDay) {
			return ErrContactInvalidFirstMetPrecision
		}
		fullDate := time.Date(*reqYear, time.Month(*reqMonth), *reqDay, 0, 0, 0, 0, time.UTC)
		contact.FirstMetAt = &fullDate
		fullPrecision := contactFirstMetPrecisionFull
		contact.FirstMetDatePrecision = &fullPrecision
		contact.FirstMetYear = nil
		contact.FirstMetMonth = nil
		contact.FirstMetDay = nil
		return nil
	default:
		return ErrContactInvalidFirstMetPrecision
	}

	contact.FirstMetAt = nil
	contact.FirstMetDatePrecision = reqPrecision
	contact.FirstMetYear = reqYear
	contact.FirstMetMonth = reqMonth
	contact.FirstMetDay = reqDay
	return nil
}

func responseContactFirstMetPrecision(contact *models.Contact) string {
	if contact.FirstMetDatePrecision != nil && *contact.FirstMetDatePrecision != "" {
		return *contact.FirstMetDatePrecision
	}
	if contact.FirstMetAt != nil {
		return contactFirstMetPrecisionFull
	}
	if contact.FirstMetYear != nil && contact.FirstMetMonth != nil {
		return contactFirstMetPrecisionMonth
	}
	if contact.FirstMetYear != nil {
		return contactFirstMetPrecisionYear
	}
	return ""
}

func isValidMonth(month int) bool {
	return month >= 1 && month <= 12
}

func isValidDay(year, month, day int) bool {
	if day < 1 {
		return false
	}
	lastDayOfMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return day <= lastDayOfMonth
}
