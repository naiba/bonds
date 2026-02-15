package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultReminderService struct {
	db *gorm.DB
}

func NewVaultReminderService(db *gorm.DB) *VaultReminderService {
	return &VaultReminderService{db: db}
}

func (s *VaultReminderService) List(vaultID string) ([]dto.VaultReminderItem, error) {
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Select("id, first_name, last_name").Find(&contacts).Error; err != nil {
		return nil, err
	}
	if len(contacts) == 0 {
		return []dto.VaultReminderItem{}, nil
	}

	contactMap := make(map[string]models.Contact, len(contacts))
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
		contactMap[c.ID] = c
	}

	var reminders []models.ContactReminder
	if err := s.db.Where("contact_id IN ?", contactIDs).Order("created_at DESC").Find(&reminders).Error; err != nil {
		return nil, err
	}

	result := make([]dto.VaultReminderItem, len(reminders))
	for i, r := range reminders {
		contact := contactMap[r.ContactID]
		result[i] = dto.VaultReminderItem{
			ReminderResponse: toReminderResponse(&r),
			ContactFirstName: ptrToStr(contact.FirstName),
			ContactLastName:  ptrToStr(contact.LastName),
		}
	}
	return result, nil
}

// GetCalendarMonth returns calendar data for a specific year/month
func (s *CalendarService) GetCalendarMonth(vaultID string, year, month int) (*dto.CalendarResponse, error) {
	return s.GetCalendar(vaultID, month, year)
}

// GetCalendarDay returns calendar data for a specific day
func (s *CalendarService) GetCalendarDay(vaultID string, year, month, day int) (*dto.CalendarResponse, error) {
	resp, err := s.GetCalendar(vaultID, month, year)
	if err != nil {
		return nil, err
	}
	// Filter to only items matching the specific day
	filteredDates := make([]dto.CalendarDateItem, 0)
	for _, d := range resp.ImportantDates {
		if d.Day != nil && *d.Day == day {
			filteredDates = append(filteredDates, d)
		}
	}
	filteredReminders := make([]dto.CalendarReminderItem, 0)
	for _, r := range resp.Reminders {
		if r.Day != nil && *r.Day == day {
			filteredReminders = append(filteredReminders, r)
		}
	}
	resp.ImportantDates = filteredDates
	resp.Reminders = filteredReminders
	return resp, nil
}
