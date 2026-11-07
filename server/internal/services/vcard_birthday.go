package services

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emersion/go-vcard"
)

const vcardParamAppleOmitYear = "X-APPLE-OMIT-YEAR"

// parseVCardBirthday is shared by file imports, subscription pulls and DAV PUT.
// Reading just BDAY's value would persist Apple's placeholder as a real year.
// Only the explicitly marked year is omitted: an unmarked 1604, or a real year
// entered by a client retaining the old parameter, must remain a known year.
func parseVCardBirthday(card vcard.Card) (year, month, day int, err error) {
	fields := card[vcard.FieldBirthday]
	if len(fields) == 0 {
		return
	}
	if len(fields) != 1 || fields[0] == nil {
		return 0, 0, 0, ErrVCardInvalidData
	}
	field := fields[0]
	value := strings.TrimSpace(field.Value)
	markers, marked := field.Params[vcardParamAppleOmitYear]
	if value == "" && !marked {
		return
	}
	year, month, day = parseBirthdayString(value)
	if month == 0 || day == 0 {
		if marked {
			return 0, 0, 0, ErrVCardInvalidData
		}
		// Preserve file import's existing handling of unsupported optional
		// birthdays (including VALUE=text); replacement validates below its
		// own boundary so it cannot silently erase an existing birthday.
		return 0, 0, 0, nil
	}
	if marked {
		if len(markers) != 1 {
			return 0, 0, 0, ErrVCardInvalidData
		}
		omittedYear, parseErr := strconv.Atoi(markers[0])
		if parseErr != nil || omittedYear <= 0 || omittedYear > 9999 || fmt.Sprintf("%04d", omittedYear) != markers[0] {
			return 0, 0, 0, ErrVCardInvalidData
		}
		if year == omittedYear {
			year = 0
		}
	}
	return
}
