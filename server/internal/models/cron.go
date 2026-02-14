package models

import "time"

type Cron struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Command   string     `json:"command" gorm:"uniqueIndex;not null"`
	LastRunAt *time.Time `json:"last_run_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
