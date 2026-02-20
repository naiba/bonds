package models

import "time"

// SystemSetting stores instance-wide configuration as key-value pairs.
// DB values override environment variable defaults.
type SystemSetting struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null;type:text"`
	Value     string    `json:"value" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
