package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrImportantDateNotFound = errors.New("important date not found")
var ErrImportantDateLabelRequired = errors.New("label is required when no type is selected")

type ImportantDateService struct {
	db *gorm.DB
}

func NewImportantDateService(db *gorm.DB) *ImportantDateService {
	return &ImportantDateService{db: db}
}

func (s *ImportantDateService) List(contactID, vaultID string) ([]dto.ImportantDateResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var dates []models.ContactImportantDate
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&dates).Error; err != nil {
		return nil, err
	}
	result := make([]dto.ImportantDateResponse, len(dates))
	for i, d := range dates {
		result[i] = toImportantDateResponse(&d)
	}
	return result, nil
}

func (s *ImportantDateService) Create(contactID, vaultID string, req dto.CreateImportantDateRequest) (*dto.ImportantDateResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	label := req.Label
	if label == "" && req.ContactImportantDateTypeID != nil {
		label = s.resolveTypeLabel(*req.ContactImportantDateTypeID)
	}
	if label == "" {
		return nil, ErrImportantDateLabelRequired
	}
	date := models.ContactImportantDate{
		ContactID:                  contactID,
		Label:                      label,
		Day:                        req.Day,
		Month:                      req.Month,
		Year:                       req.Year,
		ContactImportantDateTypeID: req.ContactImportantDateTypeID,
	}
	applyCalendarFields(&date.CalendarType, &date.OriginalDay, &date.OriginalMonth, &date.OriginalYear,
		&date.Day, &date.Month, &date.Year,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	if err := s.db.Create(&date).Error; err != nil {
		return nil, err
	}

	if req.RemindMe != nil && *req.RemindMe {
		date.RemindMe = true
		s.db.Model(&date).Update("remind_me", true)
		s.ensureReminder(contactID, &date)
	}

	resp := toImportantDateResponse(&date)
	return &resp, nil
}

func (s *ImportantDateService) Update(id uint, contactID, vaultID string, req dto.UpdateImportantDateRequest) (*dto.ImportantDateResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var date models.ContactImportantDate
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&date).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImportantDateNotFound
		}
		return nil, err
	}
	oldRemindMe := date.RemindMe
	label := req.Label
	if label == "" && req.ContactImportantDateTypeID != nil {
		label = s.resolveTypeLabel(*req.ContactImportantDateTypeID)
	}
	if label == "" {
		return nil, ErrImportantDateLabelRequired
	}
	date.Label = label
	date.Day = req.Day
	date.Month = req.Month
	date.Year = req.Year
	date.ContactImportantDateTypeID = req.ContactImportantDateTypeID
	applyCalendarFields(&date.CalendarType, &date.OriginalDay, &date.OriginalMonth, &date.OriginalYear,
		&date.Day, &date.Month, &date.Year,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	if err := s.db.Save(&date).Error; err != nil {
		return nil, err
	}

	if req.RemindMe != nil {
		newRemindMe := *req.RemindMe
		if newRemindMe != oldRemindMe {
			s.db.Model(&date).Update("remind_me", newRemindMe)
			date.RemindMe = newRemindMe
		}
		if newRemindMe {
			s.ensureReminder(contactID, &date)
		} else {
			s.removeReminder(contactID, date.ID)
		}
	}

	resp := toImportantDateResponse(&date)
	return &resp, nil
}

func (s *ImportantDateService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	s.removeReminder(contactID, id)
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactImportantDate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrImportantDateNotFound
	}
	return nil
}

func (s *ImportantDateService) resolveTypeLabel(typeID uint) string {
	var dateType models.ContactImportantDateType
	if err := s.db.First(&dateType, typeID).Error; err != nil {
		return ""
	}
	return dateType.Label
}

func (s *ImportantDateService) ensureReminder(contactID string, date *models.ContactImportantDate) error {
	var existing models.ContactReminder
	err := s.db.Where("contact_id = ? AND important_date_id = ?", contactID, date.ID).First(&existing).Error
	if err == nil {
		// Already exists, update it
		existing.Label = date.Label
		existing.Day = date.Day
		existing.Month = date.Month
		existing.Year = date.Year
		existing.CalendarType = date.CalendarType
		existing.OriginalDay = date.OriginalDay
		existing.OriginalMonth = date.OriginalMonth
		existing.OriginalYear = date.OriginalYear
		return s.db.Save(&existing).Error
	}
	// Create new yearly recurring reminder
	reminder := models.ContactReminder{
		ContactID:       contactID,
		ImportantDateID: &date.ID,
		Label:           date.Label,
		Day:             date.Day,
		Month:           date.Month,
		Year:            date.Year,
		CalendarType:    date.CalendarType,
		OriginalDay:     date.OriginalDay,
		OriginalMonth:   date.OriginalMonth,
		OriginalYear:    date.OriginalYear,
		Type:            "recurring_year",
	}
	return s.db.Create(&reminder).Error
}

func (s *ImportantDateService) removeReminder(contactID string, dateID uint) error {
	// Delete scheduled entries first
	s.db.Where("contact_reminder_id IN (SELECT id FROM contact_reminders WHERE contact_id = ? AND important_date_id = ?)", contactID, dateID).
		Delete(&models.ContactReminderScheduled{})
	// Delete the reminder
	return s.db.Where("contact_id = ? AND important_date_id = ?", contactID, dateID).
		Delete(&models.ContactReminder{}).Error
}

func toImportantDateResponse(d *models.ContactImportantDate) dto.ImportantDateResponse {
	return dto.ImportantDateResponse{
		ID:                         d.ID,
		ContactID:                  d.ContactID,
		Label:                      d.Label,
		Day:                        d.Day,
		Month:                      d.Month,
		Year:                       d.Year,
		CalendarType:               d.CalendarType,
		OriginalDay:                d.OriginalDay,
		OriginalMonth:              d.OriginalMonth,
		OriginalYear:               d.OriginalYear,
		ContactImportantDateTypeID: d.ContactImportantDateTypeID,
		RemindMe:                   d.RemindMe,
		CreatedAt:                  d.CreatedAt,
		UpdatedAt:                  d.UpdatedAt,
	}
}
