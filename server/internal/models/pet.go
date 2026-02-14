package models

import "time"

type PetCategory struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID          string    `json:"account_id" gorm:"type:text;not null;index"`
	Name               *string   `json:"name"`
	NameTranslationKey *string   `json:"name_translation_key"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

type Pet struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UUID          *string   `json:"uuid" gorm:"type:text"`
	ContactID     string    `json:"contact_id" gorm:"type:text;not null;index"`
	PetCategoryID uint      `json:"pet_category_id" gorm:"not null;index"`
	Name          *string   `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Contact     Contact     `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	PetCategory PetCategory `json:"pet_category,omitempty" gorm:"foreignKey:PetCategoryID"`
}
