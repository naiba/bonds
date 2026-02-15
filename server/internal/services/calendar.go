package services

import (
	"strconv"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type CalendarService struct {
	db *gorm.DB
}

func NewCalendarService(db *gorm.DB) *CalendarService {
	return &CalendarService{db: db}
}

func (s *CalendarService) GetCalendar(vaultID string, month, year int) (*dto.CalendarResponse, error) {
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Select("id").Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
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
		query = query.Where("month = ?", month)
	}
	if year > 0 {
		query = query.Where("year = ? OR year IS NULL", year)
	}
	if err := query.Find(&dates).Error; err != nil {
		return nil, err
	}

	importantDates := make([]dto.CalendarDateItem, len(dates))
	for i, d := range dates {
		importantDates[i] = dto.CalendarDateItem{
			ID:            d.ID,
			ContactID:     d.ContactID,
			Label:         d.Label,
			Day:           d.Day,
			Month:         d.Month,
			Year:          d.Year,
			CalendarType:  d.CalendarType,
			OriginalDay:   d.OriginalDay,
			OriginalMonth: d.OriginalMonth,
			OriginalYear:  d.OriginalYear,
		}
	}
	resp.ImportantDates = importantDates

	var reminders []models.ContactReminder
	rQuery := s.db.Where("contact_id IN ?", contactIDs)
	if month > 0 {
		rQuery = rQuery.Where("month = ?", month)
	}
	if err := rQuery.Find(&reminders).Error; err != nil {
		return nil, err
	}

	reminderItems := make([]dto.CalendarReminderItem, len(reminders))
	for i, r := range reminders {
		reminderItems[i] = dto.CalendarReminderItem{
			ID:            r.ID,
			ContactID:     r.ContactID,
			Label:         r.Label,
			Day:           r.Day,
			Month:         r.Month,
			Year:          r.Year,
			CalendarType:  r.CalendarType,
			OriginalDay:   r.OriginalDay,
			OriginalMonth: r.OriginalMonth,
			OriginalYear:  r.OriginalYear,
			Type:          r.Type,
			CreatedAt:     r.CreatedAt,
		}
	}
	resp.Reminders = reminderItems

	return resp, nil
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
