package models

import "time"

type PersonalAccessToken struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     string     `json:"user_id" gorm:"type:text;not null;index"`
	AccountID  string     `json:"account_id" gorm:"type:text;not null;index"`
	Name       string     `json:"name" gorm:"type:text;not null"`
	TokenHash  string     `json:"-" gorm:"type:text;uniqueIndex;not null"`
	TokenHint  string     `json:"token_hint" gorm:"type:text;not null"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
