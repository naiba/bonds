package models

import "time"

type Log struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID      uint64    `json:"group_id" gorm:"index;not null"`
	Level        string    `json:"level" gorm:"index;not null"`
	LevelName    string    `json:"level_name" gorm:"not null"`
	Channel      string    `json:"channel" gorm:"index;not null"`
	Message      *string   `json:"message" gorm:"type:longtext"`
	Context      *string   `json:"context" gorm:"type:longtext"`
	Extra        *string   `json:"extra" gorm:"type:longtext"`
	Formatted    *string   `json:"formatted" gorm:"type:longtext"`
	LoggableType string    `json:"loggable_type" gorm:"index:idx_loggable;not null"`
	LoggableID   uint64    `json:"loggable_id" gorm:"index:idx_loggable;not null"`
	LoggedAt     time.Time `json:"logged_at" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
