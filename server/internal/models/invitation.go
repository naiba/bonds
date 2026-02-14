package models

import "time"

type Invitation struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID  string     `json:"account_id" gorm:"type:text;not null;index"`
	Email      string     `json:"email" gorm:"type:text;not null"`
	Token      string     `json:"token" gorm:"type:text;uniqueIndex;not null"`
	Permission int        `json:"permission" gorm:"not null;default:300"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	AcceptedAt *time.Time `json:"accepted_at"`
	CreatedBy  string     `json:"created_by" gorm:"type:text;not null"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
