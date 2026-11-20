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

func (l *lunarConverter) ToGregorian(date DateInfo) (result GregorianDate, err error) {
	// lunar-go 对不存在的农历日期（比如当年没有的闰月）直接 panic。
	// 日历导出会把同一个日期按之后几十年逐年投影，这里必须把它变成普通错误，
	// 否则一个闰月生日就能让整份日历导出失败。
	defer func() {
		if recovered := recover(); recovered != nil {
			result, err = GregorianDate{}, fmt.Errorf("invalid lunar date %d-%d-%d: %v", date.Year, date.Month, date.Day, recovered)
		}
	}()

	month := date.Month
	if date.IsLeapMonth() && lunarCalendar.NewLunarYear(date.Year).GetLeapMonth() != date.AbsMonth() {
		// 这一年没有对应的闰月：按习惯落在同名的平月，与 tryOccurrenceInYear 保持一致。
		month = date.AbsMonth()
	}
	day := date.Day
	lunarMonthObj := lunarCalendar.NewLunarMonthFromYm(date.Year, month)
	if lunarMonthObj != nil {
		dayCount := lunarMonthObj.GetDayCount()
		if day > dayCount {
			day = dayCount
		}
	}
	lunar := lunarCalendar.NewLunarFromYmd(date.Year, month, day)
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
