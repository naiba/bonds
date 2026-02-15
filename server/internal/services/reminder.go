package services

import (
	"errors"
	"log"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrReminderNotFound = errors.New("reminder not found")

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
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&reminders).Error; err != nil {
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
	applyCalendarFields(&reminder.CalendarType, &reminder.OriginalDay, &reminder.OriginalMonth, &reminder.OriginalYear,
		&reminder.Day, &reminder.Month, &reminder.Year,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	if err := s.db.Create(&reminder).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "ContactReminder"
		s.feedRecorder.Record(contactID, "", ActionReminderCreated, "Created reminder: "+req.Label, &reminder.ID, &entityType)
	}

	s.scheduleReminder(&reminder)

	resp := toReminderResponse(&reminder)
	return &resp, nil
}

func (s *ReminderService) Update(id uint, contactID, vaultID string, req dto.UpdateReminderRequest) (*dto.ReminderResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var reminder models.ContactReminder
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&reminder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReminderNotFound
		}
		return nil, err
	}
	reminder.Label = req.Label
	reminder.Day = req.Day
	reminder.Month = req.Month
	reminder.Year = req.Year
	reminder.Type = req.Type
	reminder.FrequencyNumber = req.FrequencyNumber
	applyCalendarFields(&reminder.CalendarType, &reminder.OriginalDay, &reminder.OriginalMonth, &reminder.OriginalYear,
		&reminder.Day, &reminder.Month, &reminder.Year,
		req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)
	if err := s.db.Save(&reminder).Error; err != nil {
		return nil, err
	}
	resp := toReminderResponse(&reminder)
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

	s.db.Where("contact_reminder_id = ? AND triggered_at IS NULL", reminder.ID).
		Delete(&models.ContactReminderScheduled{})

	if err := s.db.Delete(&reminder).Error; err != nil {
		return err
	}
	return nil
}

func (s *ReminderService) scheduleReminder(reminder *models.ContactReminder) {
	var contact models.Contact
	if err := s.db.First(&contact, "id = ?", reminder.ContactID).Error; err != nil {
		log.Printf("[reminder] Failed to load contact %s for scheduling: %v", reminder.ContactID, err)
		return
	}

	var userVaults []models.UserVault
	if err := s.db.Where("vault_id = ?", contact.VaultID).Find(&userVaults).Error; err != nil {
		log.Printf("[reminder] Failed to load vault users for vault %s: %v", contact.VaultID, err)
		return
	}

	userIDs := make([]string, len(userVaults))
	for i, uv := range userVaults {
		userIDs[i] = uv.UserID
	}

	var channels []models.UserNotificationChannel
	if err := s.db.Where("user_id IN ? AND active = ?", userIDs, true).Find(&channels).Error; err != nil {
		log.Printf("[reminder] Failed to load notification channels: %v", err)
		return
	}

	scheduledAt := calcInitialSchedule(reminder)

	for _, ch := range channels {
		s.db.Create(&models.ContactReminderScheduled{
			UserNotificationChannelID: ch.ID,
			ContactReminderID:         reminder.ID,
			ScheduledAt:               scheduledAt,
		})
	}
}

func calcInitialSchedule(reminder *models.ContactReminder) time.Time {
	now := time.Now()

	ct := calendarPkg.CalendarType(reminder.CalendarType)
	if ct != "" && ct != calendarPkg.Gregorian && reminder.OriginalMonth != nil && reminder.OriginalDay != nil {
		converter, ok := calendarPkg.Get(ct)
		if ok {
			origDate := calendarPkg.DateInfo{
				Day:   *reminder.OriginalDay,
				Month: *reminder.OriginalMonth,
			}
			if reminder.OriginalYear != nil {
				origDate.Year = *reminder.OriginalYear
			} else {
				afterSolar := calendarPkg.GregorianDate{Day: now.Day(), Month: int(now.Month()), Year: now.Year()}
				if fromG, err := converter.FromGregorian(afterSolar); err == nil {
					origDate.Year = fromG.Year
				}
			}
			gd, err := converter.NextOccurrence(origDate, now.AddDate(0, 0, -1))
			if err == nil {
				return time.Date(gd.Year, time.Month(gd.Month), gd.Day, 9, 0, 0, 0, now.Location())
			}
		}
	}

	year := now.Year()
	month := time.January
	day := 1

	if reminder.Year != nil {
		year = *reminder.Year
	}
	if reminder.Month != nil {
		month = time.Month(*reminder.Month)
	}
	if reminder.Day != nil {
		day = *reminder.Day
	}

	scheduled := time.Date(year, month, day, 9, 0, 0, 0, now.Location())

	if reminder.Year == nil && scheduled.Before(now) {
		scheduled = scheduled.AddDate(1, 0, 0)
	}

	return scheduled
}

func toReminderResponse(r *models.ContactReminder) dto.ReminderResponse {
	return dto.ReminderResponse{
		ID:                   r.ID,
		ContactID:            r.ContactID,
		Label:                r.Label,
		Day:                  r.Day,
		Month:                r.Month,
		Year:                 r.Year,
		CalendarType:         r.CalendarType,
		OriginalDay:          r.OriginalDay,
		OriginalMonth:        r.OriginalMonth,
		OriginalYear:         r.OriginalYear,
		Type:                 r.Type,
		FrequencyNumber:      r.FrequencyNumber,
		LastTriggeredAt:      r.LastTriggeredAt,
		NumberTimesTriggered: r.NumberTimesTriggered,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}
