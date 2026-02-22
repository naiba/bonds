package models

import "time"

type UserNotificationChannel struct {
	ID                uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            *string    `json:"user_id" gorm:"type:text;index"`
	Type              string     `json:"type" gorm:"not null"`
	Label             *string    `json:"label"`
	Content           string     `json:"content" gorm:"type:text;not null"`
	PreferredTime     *string    `json:"preferred_time" gorm:"type:time"`
	Active            bool       `json:"active" gorm:"default:false"`
	Fails             int        `json:"fails" gorm:"default:0"`
	VerifiedAt        *time.Time `json:"verified_at"`
	VerificationToken *string    `json:"verification_token"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	User                 *User                  `json:"user,omitempty" gorm:"foreignKey:UserID"`
	UserNotificationSent []UserNotificationSent `json:"user_notification_sent,omitempty" gorm:"foreignKey:UserNotificationChannelID"`
}

type UserNotificationSent struct {
	ID                        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserNotificationChannelID uint      `json:"user_notification_channel_id" gorm:"not null;index"`
	SentAt                    time.Time `json:"sent_at" gorm:"not null"`
	SubjectLine               string    `json:"subject_line" gorm:"not null"`
	Payload                   *string   `json:"payload" gorm:"type:text"`
	Error                     *string   `json:"error" gorm:"type:text"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`

	NotificationChannel UserNotificationChannel `json:"notification_channel,omitempty" gorm:"foreignKey:UserNotificationChannelID"`
}
