package calendar

import "time"

// CalendarType represents a calendar system identifier.
type CalendarType string

const (
	Gregorian CalendarType = "gregorian"
	Lunar     CalendarType = "lunar"
)

// DateInfo holds a date in a specific calendar system.
type DateInfo struct {
	Day   int
	Month int // For lunar: negative means leap month (e.g., -4 = leap April)
	Year  int
}

// IsLeapMonth returns true if the month is a leap month (negative value).
func (d DateInfo) IsLeapMonth() bool {
	return d.Month < 0
}

// AbsMonth returns the absolute month number (strips leap indicator).
func (d DateInfo) AbsMonth() int {
	if d.Month < 0 {
		return -d.Month
	}
	return d.Month
}

// GregorianDate holds a Gregorian calendar date.
type GregorianDate struct {
	Day   int
	Month int
	Year  int
}

// Converter converts between a specific calendar system and Gregorian.
type Converter interface {
	// ToGregorian converts a date from this calendar to Gregorian.
	ToGregorian(date DateInfo) (GregorianDate, error)

	// FromGregorian converts a Gregorian date to this calendar.
	FromGregorian(date GregorianDate) (DateInfo, error)

	// NextOccurrence finds the next Gregorian date for a recurring annual date
	// in this calendar system, strictly after the given time.
	// Used for yearly recurring events (e.g., lunar birthday).
	NextOccurrence(originalDate DateInfo, after time.Time) (GregorianDate, error)

	// Type returns the calendar type identifier.
	Type() CalendarType
}

// registry holds all registered converters.
var registry = map[CalendarType]Converter{}

// Register adds a converter to the global registry.
func Register(c Converter) {
	registry[c.Type()] = c
}

// Get returns the converter for a calendar type.
func Get(ct CalendarType) (Converter, bool) {
	c, ok := registry[ct]
	return c, ok
}

// IsSupported checks if a calendar type is registered.
func IsSupported(ct CalendarType) bool {
	_, ok := registry[ct]
	return ok
}

// SupportedTypes returns all registered calendar type identifiers.
func SupportedTypes() []CalendarType {
	types := make([]CalendarType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
