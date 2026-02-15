package calendar

import "time"

type gregorianConverter struct{}

func init() {
	Register(&gregorianConverter{})
}

func (g *gregorianConverter) Type() CalendarType {
	return Gregorian
}

func (g *gregorianConverter) ToGregorian(date DateInfo) (GregorianDate, error) {
	return GregorianDate{Day: date.Day, Month: date.Month, Year: date.Year}, nil
}

func (g *gregorianConverter) FromGregorian(date GregorianDate) (DateInfo, error) {
	return DateInfo{Day: date.Day, Month: date.Month, Year: date.Year}, nil
}

func (g *gregorianConverter) NextOccurrence(originalDate DateInfo, after time.Time) (GregorianDate, error) {
	candidate := time.Date(after.Year(), time.Month(originalDate.Month), originalDate.Day, 0, 0, 0, 0, after.Location())
	if !candidate.After(after) {
		candidate = candidate.AddDate(1, 0, 0)
	}
	return GregorianDate{Day: candidate.Day(), Month: int(candidate.Month()), Year: candidate.Year()}, nil
}
