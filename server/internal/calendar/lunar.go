package calendar

import (
	"fmt"
	"time"

	lunarCalendar "github.com/6tail/lunar-go/calendar"
)

type lunarConverter struct{}

func init() {
	Register(&lunarConverter{})
}

func (l *lunarConverter) Type() CalendarType {
	return Lunar
}

func (l *lunarConverter) ToGregorian(date DateInfo) (GregorianDate, error) {
	day := date.Day
	lunarMonthObj := lunarCalendar.NewLunarMonthFromYm(date.Year, date.Month)
	if lunarMonthObj != nil {
		dayCount := lunarMonthObj.GetDayCount()
		if day > dayCount {
			day = dayCount
		}
	}
	lunar := lunarCalendar.NewLunarFromYmd(date.Year, date.Month, day)
	solar := lunar.GetSolar()
	return GregorianDate{
		Day:   solar.GetDay(),
		Month: solar.GetMonth(),
		Year:  solar.GetYear(),
	}, nil
}

func (l *lunarConverter) FromGregorian(date GregorianDate) (DateInfo, error) {
	solar := lunarCalendar.NewSolarFromYmd(date.Year, date.Month, date.Day)
	lunar := solar.GetLunar()
	return DateInfo{
		Day:   lunar.GetDay(),
		Month: lunar.GetMonth(),
		Year:  lunar.GetYear(),
	}, nil
}

func (l *lunarConverter) NextOccurrence(originalDate DateInfo, after time.Time) (GregorianDate, error) {
	afterSolar := lunarCalendar.NewSolarFromYmd(after.Year(), int(after.Month()), after.Day())
	afterLunar := afterSolar.GetLunar()

	for _, lunarYear := range []int{afterLunar.GetYear(), afterLunar.GetYear() + 1} {
		gd, err := l.tryOccurrenceInYear(originalDate, lunarYear)
		if err != nil {
			continue
		}
		candidate := time.Date(gd.Year, time.Month(gd.Month), gd.Day, 0, 0, 0, 0, after.Location())
		if candidate.After(after) {
			return gd, nil
		}
	}

	return GregorianDate{}, fmt.Errorf("cannot find next occurrence for lunar date %d/%d after %v", originalDate.Month, originalDate.Day, after)
}

func (l *lunarConverter) tryOccurrenceInYear(originalDate DateInfo, lunarYear int) (GregorianDate, error) {
	month := originalDate.Month
	day := originalDate.Day

	if originalDate.IsLeapMonth() {
		absMonth := originalDate.AbsMonth()
		lunarYearObj := lunarCalendar.NewLunarYear(lunarYear)
		leapMonth := lunarYearObj.GetLeapMonth()
		if leapMonth != absMonth {
			month = absMonth
		} else {
			month = -absMonth
		}
	}

	lunarMonthObj := lunarCalendar.NewLunarMonthFromYm(lunarYear, month)
	if lunarMonthObj == nil {
		return GregorianDate{}, fmt.Errorf("invalid lunar month %d in year %d", month, lunarYear)
	}
	dayCount := lunarMonthObj.GetDayCount()
	if day > dayCount {
		day = dayCount
	}

	lunar := lunarCalendar.NewLunarFromYmd(lunarYear, month, day)
	solar := lunar.GetSolar()

	return GregorianDate{
		Day:   solar.GetDay(),
		Month: solar.GetMonth(),
		Year:  solar.GetYear(),
	}, nil
}
