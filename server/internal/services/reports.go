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

func (s *ReportService) ImportantDatesReport(vaultID, userID string) ([]dto.ImportantDateReportItem, error) {
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
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
		VaultID       string  `gorm:"column:vault_id"`
		FirstName     *string `gorm:"column:first_name"`
		LastName      *string `gorm:"column:last_name"`
		MiddleName    *string `gorm:"column:middle_name"`
		Nickname      *string `gorm:"column:nickname"`
		MaidenName    *string `gorm:"column:maiden_name"`
		Prefix        *string `gorm:"column:prefix"`
		Suffix        *string `gorm:"column:suffix"`
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
	queryErr := s.db.Model(&models.ContactImportantDate{}).
		Select("contact_important_dates.contact_id, contacts.vault_id, contacts.first_name, contacts.last_name, contacts.middle_name, contacts.nickname, contacts.maiden_name, contacts.prefix, contacts.suffix, contact_important_dates.label, contact_important_dates.day, contact_important_dates.month, contact_important_dates.year, contact_important_dates.calendar_type, contact_important_dates.original_day, contact_important_dates.original_month, contact_important_dates.original_year").
		Joins("JOIN contacts ON contacts.id = contact_important_dates.contact_id").
		Where("contact_important_dates.contact_id IN ?", contactIDs).
		Order("contact_important_dates.month ASC, contact_important_dates.day ASC").
		Scan(&rows).Error
	if queryErr != nil {
		return nil, queryErr
	}

	result := make([]dto.ImportantDateReportItem, len(rows))
	for i, r := range rows {
		contact := models.Contact{VaultID: r.VaultID, FirstName: r.FirstName, LastName: r.LastName, MiddleName: r.MiddleName, Nickname: r.Nickname, MaidenName: r.MaidenName, Prefix: r.Prefix, Suffix: r.Suffix}
		contactName, err := formatter.format(&contact, "")
		if err != nil {
			return nil, err
		}
		result[i] = dto.ImportantDateReportItem{
			ContactID:     r.ContactID,
			ContactName:   contactName,
			FirstName:     ptrToStr(r.FirstName),
			LastName:      ptrToStr(r.LastName),
			MiddleName:    ptrToStr(r.MiddleName),
			Nickname:      ptrToStr(r.Nickname),
			MaidenName:    ptrToStr(r.MaidenName),
			Prefix:        ptrToStr(r.Prefix),
			Suffix:        ptrToStr(r.Suffix),
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

func (s *ReportService) Overview(vaultID string) (*dto.ReportOverviewResponse, error) {
	var totalContacts int64
	// Exclude shadow contacts (Listed=false) created for UserVault self-contact
	if err := s.db.Model(&models.Contact{}).Where("vault_id = ? AND listed = ?", vaultID, true).Count(&totalContacts).Error; err != nil {
		return nil, err
	}

	var totalAddresses int64
	if err := s.db.Model(&models.Address{}).Where("vault_id = ?", vaultID).Count(&totalAddresses).Error; err != nil {
		return nil, err
	}

	var contactIDs []string
	if err := s.db.Model(&models.Contact{}).Where("vault_id = ? AND listed = ?", vaultID, true).Pluck("id", &contactIDs).Error; err != nil {
		return nil, err
	}

	var totalDates int64
	if len(contactIDs) > 0 {
		if err := s.db.Model(&models.ContactImportantDate{}).Where("contact_id IN ?", contactIDs).Count(&totalDates).Error; err != nil {
			return nil, err
		}
	}

	var totalMood int64
	var paramIDs []uint
	if err := s.db.Model(&models.MoodTrackingParameter{}).Where("vault_id = ?", vaultID).Pluck("id", &paramIDs).Error; err != nil {
		return nil, err
	}
	if len(paramIDs) > 0 {
		if err := s.db.Model(&models.MoodTrackingEvent{}).Where("mood_tracking_parameter_id IN ?", paramIDs).Count(&totalMood).Error; err != nil {
			return nil, err
		}
	}

	return &dto.ReportOverviewResponse{
		TotalContacts:       int(totalContacts),
		TotalAddresses:      int(totalAddresses),
		TotalImportantDates: int(totalDates),
		TotalMoodEntries:    int(totalMood),
	}, nil
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
