package services

import (
	"strconv"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/utils"
	"gorm.io/gorm"
)

type CalendarService struct {
	db *gorm.DB
}

func NewCalendarService(db *gorm.DB) *CalendarService {
	return &CalendarService{db: db}
}

func (s *CalendarService) GetCalendar(vaultID, userID string, month, year int, locale string) (*dto.CalendarResponse, error) {
	nameOrder, err := GetUserNameOrder(s.db, userID)
	if err != nil {
		return nil, err
	}

	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).
		Select("id, first_name, middle_name, last_name, nickname, maiden_name, prefix, suffix").
		Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactIDs := make([]string, len(contacts))
	contactNames := make(map[string]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
		contactNames[c.ID] = utils.FormatContactName(nameOrder, &contacts[i], i18n.T(locale, "reminder.unknown_contact"))
	}

	resp := &dto.CalendarResponse{
		ImportantDates: []dto.CalendarDateItem{},
		Reminders:      []dto.CalendarReminderItem{},
	}

	if len(contactIDs) == 0 {
		return resp, nil
	}

	var dates []models.ContactImportantDate
	query := s.db.Where("contact_id IN ?", contactIDs)
	if month > 0 {
		query = query.Where(
			"month = ? OR (calendar_type <> ? AND original_day IS NOT NULL AND original_month IS NOT NULL)",
			month, string(calendarPkg.Gregorian),
		)
	}
	// Important dates (especially birthdays) should appear every year regardless
	// of what year is stored. The year field is only used for display, not filtering.
	if err := query.Find(&dates).Error; err != nil {
		return nil, err
	}

	importantDates := make([]dto.CalendarDateItem, 0, len(dates))
	for _, d := range dates {
		day, itemMonth := d.Day, d.Month
		if projectedDay, projectedMonth, ok := projectAlternativeCalendarDate(
			d.CalendarType, d.OriginalDay, d.OriginalMonth, year,
		); ok {
			day = projectedDay
			itemMonth = projectedMonth
		}
		if month > 0 && (itemMonth == nil || *itemMonth != month) {
			continue
		}
		importantDates = append(importantDates, dto.CalendarDateItem{
			ID:            d.ID,
			ContactID:     d.ContactID,
			ContactName:   contactNames[d.ContactID],
			Label:         d.Label,
			Day:           day,
			Month:         itemMonth,
			Year:          d.Year,
			CalendarType:  d.CalendarType,
			OriginalDay:   d.OriginalDay,
			OriginalMonth: d.OriginalMonth,
			OriginalYear:  d.OriginalYear,
		})
	}
	resp.ImportantDates = importantDates

	var reminders []models.ContactReminder
	rQuery := s.db.Where("contact_id IN ?", contactIDs)
	if month > 0 {
		rQuery = rQuery.Where(
			"month = ? OR (type = ? AND calendar_type <> ? AND original_day IS NOT NULL AND original_month IS NOT NULL)",
			month, "recurring_year", string(calendarPkg.Gregorian),
		)
	}
	if err := rQuery.Find(&reminders).Error; err != nil {
		return nil, err
	}

	reminderItems := make([]dto.CalendarReminderItem, 0, len(reminders))
	for _, r := range reminders {
		if r.Type == "one_time" && r.Year != nil && year > 0 && *r.Year != year {
			continue
		}
		day, itemMonth := r.Day, r.Month
		if r.Type == "recurring_year" {
			if projectedDay, projectedMonth, ok := projectAlternativeCalendarDate(
				r.CalendarType, r.OriginalDay, r.OriginalMonth, year,
			); ok {
				day = projectedDay
				itemMonth = projectedMonth
			}
		}
		if month > 0 && (itemMonth == nil || *itemMonth != month) {
			continue
		}
		reminderItems = append(reminderItems, dto.CalendarReminderItem{
			ID:            r.ID,
			ContactID:     r.ContactID,
			ContactName:   contactNames[r.ContactID],
			Label:         r.Label,
			Day:           day,
			Month:         itemMonth,
			Year:          r.Year,
			CalendarType:  r.CalendarType,
			OriginalDay:   r.OriginalDay,
			OriginalMonth: r.OriginalMonth,
			OriginalYear:  r.OriginalYear,
			Type:          r.Type,
			CreatedAt:     r.CreatedAt,
		})
	}
	resp.Reminders = reminderItems

	return resp, nil
}

func projectAlternativeCalendarDate(calendarType string, originalDay, originalMonth *int, year int) (*int, *int, bool) {
	if year <= 0 || calendarType == "" || calendarType == "gregorian" || originalDay == nil || originalMonth == nil {
		return nil, nil, false
	}
	converter, ok := calendarPkg.Get(calendarPkg.CalendarType(calendarType))
	if !ok {
		return nil, nil, false
	}
	gd, err := calendarOccurrenceInYear(converter, calendarPkg.DateInfo{
		Day:   *originalDay,
		Month: *originalMonth,
	}, year)
	if err != nil {
		return nil, nil, false
	}
	return &gd.Day, &gd.Month, true
}

func ParseIntParam(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
