package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type ReportService struct {
	db *gorm.DB
}

func NewReportService(db *gorm.DB) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) AddressReport(vaultID string) ([]dto.AddressReportItem, error) {
	var results []dto.AddressReportItem
	err := s.db.Model(&models.Address{}).
		Select("COALESCE(country, '') as country, COALESCE(province, '') as province, COALESCE(city, '') as city, COUNT(*) as count").
		Where("vault_id = ?", vaultID).
		Group("country, province, city").
		Order("count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []dto.AddressReportItem{}
	}
	return results, nil
}

func (s *ReportService) ImportantDatesReport(vaultID string) ([]dto.ImportantDateReportItem, error) {
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Select("id").Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
	}
	if len(contactIDs) == 0 {
		return []dto.ImportantDateReportItem{}, nil
	}

	type dateRow struct {
		ContactID     string  `gorm:"column:contact_id"`
		FirstName     *string `gorm:"column:first_name"`
		LastName      *string `gorm:"column:last_name"`
		Label         string  `gorm:"column:label"`
		Day           *int    `gorm:"column:day"`
		Month         *int    `gorm:"column:month"`
		Year          *int    `gorm:"column:year"`
		CalendarType  string  `gorm:"column:calendar_type"`
		OriginalDay   *int    `gorm:"column:original_day"`
		OriginalMonth *int    `gorm:"column:original_month"`
		OriginalYear  *int    `gorm:"column:original_year"`
	}

	var rows []dateRow
	err := s.db.Model(&models.ContactImportantDate{}).
		Select("contact_important_dates.contact_id, contacts.first_name, contacts.last_name, contact_important_dates.label, contact_important_dates.day, contact_important_dates.month, contact_important_dates.year, contact_important_dates.calendar_type, contact_important_dates.original_day, contact_important_dates.original_month, contact_important_dates.original_year").
		Joins("JOIN contacts ON contacts.id = contact_important_dates.contact_id").
		Where("contact_important_dates.contact_id IN ?", contactIDs).
		Order("contact_important_dates.month ASC, contact_important_dates.day ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.ImportantDateReportItem, len(rows))
	for i, r := range rows {
		result[i] = dto.ImportantDateReportItem{
			ContactID:     r.ContactID,
			FirstName:     ptrToStr(r.FirstName),
			LastName:      ptrToStr(r.LastName),
			Label:         r.Label,
			Day:           r.Day,
			Month:         r.Month,
			Year:          r.Year,
			CalendarType:  r.CalendarType,
			OriginalDay:   r.OriginalDay,
			OriginalMonth: r.OriginalMonth,
			OriginalYear:  r.OriginalYear,
		}
	}
	return result, nil
}

func (s *ReportService) MoodReport(vaultID string) ([]dto.MoodReportItem, error) {
	var params []models.MoodTrackingParameter
	if err := s.db.Where("vault_id = ?", vaultID).Find(&params).Error; err != nil {
		return nil, err
	}

	result := make([]dto.MoodReportItem, 0, len(params))
	for _, p := range params {
		var count int64
		s.db.Model(&models.MoodTrackingEvent{}).Where("mood_tracking_parameter_id = ?", p.ID).Count(&count)
		result = append(result, dto.MoodReportItem{
			ParameterLabel: ptrToStr(p.Label),
			HexColor:       p.HexColor,
			Count:          int(count),
		})
	}
	return result, nil
}
