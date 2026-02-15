package models

import "time"

type ContactReminder struct {
	ID                   uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID            string     `json:"contact_id" gorm:"type:text;not null;index"`
	Label                string     `json:"label" gorm:"not null"`
	Day                  *int       `json:"day"`
	Month                *int       `json:"month"`
	Year                 *int       `json:"year"`
	CalendarType         string     `json:"calendar_type" gorm:"default:'gregorian'"`
	OriginalDay          *int       `json:"original_day"`
	OriginalMonth        *int       `json:"original_month"`
	OriginalYear         *int       `json:"original_year"`
	Type                 string     `json:"type" gorm:"not null"`
	FrequencyNumber      *int       `json:"frequency_number"`
	LastTriggeredAt      *time.Time `json:"last_triggered_at"`
	NumberTimesTriggered int        `json:"number_times_triggered" gorm:"default:0"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	Contact                  Contact                   `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	UserNotificationChannels []UserNotificationChannel `json:"user_notification_channels,omitempty" gorm:"many2many:contact_reminder_scheduled"`
}

type ContactReminderScheduled struct {
	ID                        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserNotificationChannelID uint       `json:"user_notification_channel_id" gorm:"not null;index"`
	ContactReminderID         uint       `json:"contact_reminder_id" gorm:"not null;index"`
	ScheduledAt               time.Time  `json:"scheduled_at" gorm:"not null"`
	TriggeredAt               *time.Time `json:"triggered_at"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`

	ContactReminder         ContactReminder         `json:"contact_reminder,omitempty" gorm:"foreignKey:ContactReminderID"`
	UserNotificationChannel UserNotificationChannel `json:"user_notification_channel,omitempty" gorm:"foreignKey:UserNotificationChannelID"`
}

func (ContactReminderScheduled) TableName() string {
	return "contact_reminder_scheduled"
}
