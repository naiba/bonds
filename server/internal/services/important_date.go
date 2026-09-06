package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrImportantDateNotFound = errors.New("important date not found")
var ErrImportantDateLabelRequired = errors.New("label is required when no type is selected")
var ErrImportantDateTypeNotFound = errors.New("important date type not found")
var ErrImportantDateSingletonConflict = errors.New("important date type already exists for contact")

var singletonImportantDateInternalTypes = map[string]struct{}{
	"birthdate":     {},
	"deceased_date": {},
}

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
	var result *dto.ImportantDateResponse
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = NewImportantDateService(tx).create(contactID, vaultID, req)
		return err
	})
	return result, err
}

func (s *ImportantDateService) create(contactID, vaultID string, req dto.CreateImportantDateRequest) (*dto.ImportantDateResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	label := req.Label
	var dateType *models.ContactImportantDateType
	if label == "" && req.ContactImportantDateTypeID != nil {
		resolved, err := s.resolveType(*req.ContactImportantDateTypeID, vaultID)
		if err != nil {
			return nil, err
		}
		dateType = resolved
		label = resolved.Label
	} else if req.ContactImportantDateTypeID != nil {
		resolved, err := s.resolveType(*req.ContactImportantDateTypeID, vaultID)
		if err != nil {
			return nil, err
		}
		dateType = resolved
	}
	if label == "" {
		return nil, ErrImportantDateLabelRequired
	}
	if err := s.ensureSingletonAvailable(contactID, dateType, 0); err != nil {
		return nil, err
	}
	date := models.ContactImportantDate{
		ContactID:                  contactID,
		Label:                      label,
		DatePrecision:              req.DatePrecision,
		Day:                        req.Day,
		Month:                      req.Month,
		Year:                       req.Year,
		ContactImportantDateTypeID: req.ContactImportantDateTypeID,
	}
	applyCalendarFields(&date.CalendarType, &date.OriginalDay, &date.OriginalMonth, &date.OriginalYear,
		&date.Day, &date.Month, &date.Year,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	clearYearlessAlternativeCalendarProjectionYear(&date, req.DatePrecision)
	if err := applyImportantDatePrecision(&date, req.DatePrecision); err != nil {
		return nil, err
	}
	if err := validateImportantDateCalendarDay(&date); err != nil {
		return nil, err
	}
	if err := s.db.Create(&date).Error; err != nil {
		return nil, err
	}

	if req.RemindMe != nil && *req.RemindMe && importantDateCanScheduleReminder(&date) {
		date.RemindMe = true
		if err := s.db.Model(&date).Update("remind_me", true).Error; err != nil {
			return nil, err
		}
		if err := s.ensureReminder(contactID, &date); err != nil {
			return nil, err
		}
	}

	resp := toImportantDateResponse(&date)
	return &resp, nil
}

func (s *ImportantDateService) Update(id uint, contactID, vaultID string, req dto.UpdateImportantDateRequest) (*dto.ImportantDateResponse, error) {
	var result *dto.ImportantDateResponse
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = NewImportantDateService(tx).update(id, contactID, vaultID, req)
		return err
	})
	return result, err
}

func (s *ImportantDateService) update(id uint, contactID, vaultID string, req dto.UpdateImportantDateRequest) (*dto.ImportantDateResponse, error) {
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
	var dateType *models.ContactImportantDateType
	if label == "" && req.ContactImportantDateTypeID != nil {
		resolved, err := s.resolveType(*req.ContactImportantDateTypeID, vaultID)
		if err != nil {
			return nil, err
		}
		dateType = resolved
		label = resolved.Label
	} else if req.ContactImportantDateTypeID != nil {
		resolved, err := s.resolveType(*req.ContactImportantDateTypeID, vaultID)
		if err != nil {
			return nil, err
		}
		dateType = resolved
	}
	if label == "" {
		return nil, ErrImportantDateLabelRequired
	}
	if req.ContactImportantDateTypeID == nil || date.ContactImportantDateTypeID == nil || *req.ContactImportantDateTypeID != *date.ContactImportantDateTypeID {
		if err := s.ensureSingletonAvailable(contactID, dateType, date.ID); err != nil {
			return nil, err
		}
	}
	date.Label = label
	date.DatePrecision = req.DatePrecision
	date.Day = req.Day
	date.Month = req.Month
	date.Year = req.Year
	date.ContactImportantDateTypeID = req.ContactImportantDateTypeID
	applyCalendarFields(&date.CalendarType, &date.OriginalDay, &date.OriginalMonth, &date.OriginalYear,
		&date.Day, &date.Month, &date.Year,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	clearYearlessAlternativeCalendarProjectionYear(&date, req.DatePrecision)
	if err := applyImportantDatePrecision(&date, req.DatePrecision); err != nil {
		return nil, err
	}
	if err := validateImportantDateCalendarDay(&date); err != nil {
		return nil, err
	}
	if err := s.db.Save(&date).Error; err != nil {
		return nil, err
	}

	if !importantDateCanScheduleReminder(&date) {
		if date.RemindMe {
			if err := s.db.Model(&date).Update("remind_me", false).Error; err != nil {
				return nil, err
			}
			date.RemindMe = false
		}
		if err := s.removeReminder(contactID, date.ID); err != nil {
			return nil, err
		}
	} else if req.RemindMe != nil {
		newRemindMe := *req.RemindMe && importantDateCanScheduleReminder(&date)
		if newRemindMe != oldRemindMe {
			if err := s.db.Model(&date).Update("remind_me", newRemindMe).Error; err != nil {
				return nil, err
			}
			date.RemindMe = newRemindMe
		}
		if newRemindMe {
			if err := s.ensureReminder(contactID, &date); err != nil {
				return nil, err
			}
		} else {
			if err := s.removeReminder(contactID, date.ID); err != nil {
				return nil, err
			}
		}
	} else if date.RemindMe {
		if err := s.ensureReminder(contactID, &date); err != nil {
			return nil, err
		}
	}

	resp := toImportantDateResponse(&date)
	return &resp, nil
}

func (s *ImportantDateService) Delete(id uint, contactID, vaultID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return NewImportantDateService(tx).delete(id, contactID, vaultID)
	})
}

func (s *ImportantDateService) delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	if err := s.removeReminder(contactID, id); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactImportantDate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrImportantDateNotFound
	}
	return nil
}

func (s *ImportantDateService) resolveType(typeID uint, vaultID string) (*models.ContactImportantDateType, error) {
	var dateType models.ContactImportantDateType
	if err := s.db.Where("id = ? AND vault_id = ?", typeID, vaultID).First(&dateType).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImportantDateTypeNotFound
		}
		return nil, err
	}
	return &dateType, nil
}

func (s *ImportantDateService) ensureSingletonAvailable(contactID string, dateType *models.ContactImportantDateType, excludeID uint) error {
	if dateType == nil || dateType.InternalType == nil {
		return nil
	}
	if _, singleton := singletonImportantDateInternalTypes[*dateType.InternalType]; !singleton {
		return nil
	}
	query := s.db.Model(&models.ContactImportantDate{}).
		Joins("JOIN contact_important_date_types ON contact_important_date_types.id = contact_important_dates.contact_important_date_type_id").
		Where("contact_important_dates.contact_id = ? AND contact_important_date_types.internal_type = ?", contactID, *dateType.InternalType)
	if excludeID != 0 {
		query = query.Where("contact_important_dates.id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrImportantDateSingletonConflict
	}
	return nil
}

func (s *ImportantDateService) ensureReminder(contactID string, date *models.ContactImportantDate) error {
	if !importantDateCanScheduleReminder(date) {
		return nil
	}
	var existing models.ContactReminder
	err := s.db.Where("contact_id = ? AND important_date_id = ?", contactID, date.ID).First(&existing).Error
	if err == nil {
		existing.Label = date.Label
		existing.Day = date.Day
		existing.Month = date.Month
		existing.Year = date.Year
		existing.CalendarType = date.CalendarType
		existing.OriginalDay = date.OriginalDay
		existing.OriginalMonth = date.OriginalMonth
		existing.OriginalYear = date.OriginalYear
		if err := s.db.Save(&existing).Error; err != nil {
			return err
		}
		return reschedulePendingReminder(s.db, &existing)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
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
	if err := s.db.Create(&reminder).Error; err != nil {
		return err
	}
	return scheduleReminderForVaultUsers(s.db, &reminder)
}

func (s *ImportantDateService) removeReminder(contactID string, dateID uint) error {
	// Delete scheduled entries first
	if err := s.db.Where("contact_reminder_id IN (SELECT id FROM contact_reminders WHERE contact_id = ? AND important_date_id = ?)", contactID, dateID).
		Delete(&models.ContactReminderScheduled{}).Error; err != nil {
		return err
	}
	if err := s.db.Where("contact_reminder_id IN (SELECT id FROM contact_reminders WHERE contact_id = ? AND important_date_id = ?)", contactID, dateID).
		Delete(&models.ContactReminderSelectedUser{}).Error; err != nil {
		return err
	}
	// Delete the reminder
	return s.db.Where("contact_id = ? AND important_date_id = ?", contactID, dateID).
		Delete(&models.ContactReminder{}).Error
}

func toImportantDateResponse(d *models.ContactImportantDate) dto.ImportantDateResponse {
	return dto.ImportantDateResponse{
		ID:                         d.ID,
		ContactID:                  d.ContactID,
		Label:                      d.Label,
		DatePrecision:              responseImportantDatePrecision(d),
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

func validateImportantDateCalendarDay(date *models.ContactImportantDate) error {
	if date.Day == nil || date.Month == nil {
		return nil
	}
	if !isValidReminderMonth(*date.Month) {
		return ErrImportantDateInvalidPrecision
	}
	if date.Year == nil {
		if !isValidReminderMonthDay(*date.Month, *date.Day) {
			return ErrImportantDateInvalidPrecision
		}
		return nil
	}
	if !isValidReminderDay(*date.Year, *date.Month, *date.Day) {
		return ErrImportantDateInvalidPrecision
	}
	return nil
}
