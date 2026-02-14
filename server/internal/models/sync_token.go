package models

import "time"

type SyncToken struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID string    `json:"account_id" gorm:"type:text;not null;index:idx_sync_token"`
	UserID    string    `json:"user_id" gorm:"type:text;not null;index:idx_sync_token"`
	Name      string    `json:"name" gorm:"default:'contacts';index:idx_sync_token"`
	Timestamp time.Time `json:"timestamp" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
