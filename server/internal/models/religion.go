package models

import "time"

type Religion struct {
	ID             uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID      string    `json:"account_id" gorm:"type:text;not null;index"`
	Name           *string   `json:"name"`
	TranslationKey *string   `json:"translation_key"`
	Position       *int      `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}
