package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrReminderNotFound               = errors.New("reminder not found")
	ErrReminderInvalidAudience        = errors.New("invalid reminder audience")
	ErrReminderAudienceUserNotInVault = errors.New("reminder audience user is not in vault")
)

type ReminderService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewReminderService(db *gorm.DB) *ReminderService {
	return &ReminderService{db: db}
}

func (s *ReminderService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *ReminderService) List(contactID, vaultID string) ([]dto.ReminderResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var reminders []models.ContactReminder
	if err := s.db.Preload("SelectedUsers").Where("contact_id = ?", contactID).Order("created_at DESC").Find(&reminders).Error; err != nil {
		return nil, err
	}
	result := make([]dto.ReminderResponse, len(reminders))
	for i, r := range reminders {
		result[i] = toReminderResponse(&r)
	}
	return result, nil
}

func (s *ReminderService) Create(contactID, vaultID string, req dto.CreateReminderRequest) (*dto.ReminderResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	reminder := models.ContactReminder{
		ContactID:       contactID,
		Label:           req.Label,
		Day:             req.Day,
		Month:           req.Month,
		Year:            req.Year,
		Type:            req.Type,
		FrequencyNumber: req.FrequencyNumber,
	}
	audience, selectedUserIDs, err := normalizeReminderAudience(req.Audience, req.SelectedUserIDs)
	if err != nil {
		return nil, err
	}
	reminder.Audience = audience
	if err := validateAndApplyReminderDate(&reminder.Day, &reminder.Month, &reminder.Year, &reminder.CalendarType, &reminder.OriginalDay, &reminder.OriginalMonth, &reminder.OriginalYear,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear); err != nil {
		return nil, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		if err := validateReminderAudienceUsers(tx, vaultID, audience, selectedUserIDs); err != nil {
			return err
		}
		if err := tx.Create(&reminder).Error; err != nil {
			return err
		}
		if err := replaceReminderSelectedUsers(tx, reminder.ID, selectedUserIDs); err != nil {
			return err
		}
		if err := scheduleReminderForVaultUsers(tx, &reminder); err != nil {
			return err
		}
		if s.feedRecorder != nil {
			entityType := "ContactReminder"
			if err := NewFeedRecorder(tx).Record(contactID, "", ActionReminderCreated, "Created reminder: "+req.Label, &reminder.ID, &entityType); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	resp := toReminderResponse(&reminder)
	resp.SelectedUserIDs = selectedUserIDs
	return &resp, nil
}

func (s *ReminderService) Update(id uint, contactID, vaultID string, req dto.UpdateReminderRequest) (*dto.ReminderResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var reminder models.ContactReminder
	if err := s.db.Preload("SelectedUsers").Where("id = ? AND contact_id = ?", id, contactID).First(&reminder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReminderNotFound
		}
		return nil, err
	}
	audience, selectedUserIDs, err := reminderAudienceForUpdate(req, &reminder)
	if err != nil {
		return nil, err
	}
	reminder.Label = req.Label
	reminder.Day = req.Day
	reminder.Month = req.Month
	reminder.Year = req.Year
	reminder.Type = req.Type
	reminder.FrequencyNumber = req.FrequencyNumber
	reminder.Audience = audience
	if err := validateAndApplyReminderDate(&reminder.Day, &reminder.Month, &reminder.Year, &reminder.CalendarType, &reminder.OriginalDay, &reminder.OriginalMonth, &reminder.OriginalYear,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear); err != nil {
		return nil, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := validateReminderAudienceUsers(tx, vaultID, audience, selectedUserIDs); err != nil {
			return err
		}
		if err := updateContactRecord(tx, &reminder, contactID, vaultID, ErrReminderNotFound,
			"label", "day", "month", "year",
			"type", "frequency_number", "audience", "calendar_type",
			"original_day", "original_month", "original_year",
		); err != nil {
			return err
		}
		if err := replaceReminderSelectedUsers(tx, reminder.ID, selectedUserIDs); err != nil {
			return err
		}
		return reschedulePendingReminder(tx, &reminder)
	}); err != nil {
		return nil, err
	}
	resp := toReminderResponse(&reminder)
	resp.SelectedUserIDs = selectedUserIDs
	return &resp, nil
}

func (s *ReminderService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	var reminder models.ContactReminder
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&reminder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReminderNotFound
		}
		return err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("contact_reminder_id = ? AND triggered_at IS NULL", reminder.ID).
			Delete(&models.ContactReminderScheduled{}).Error; err != nil {
			return err
		}
		if err := tx.Where("contact_reminder_id = ?", reminder.ID).Delete(&models.ContactReminderSelectedUser{}).Error; err != nil {
			return err
		}
		return tx.Delete(&reminder).Error
	}); err != nil {
		return err
	}
	return nil
}
