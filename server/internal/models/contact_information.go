package models

import "time"

type ContactInformationType struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID          string    `json:"account_id" gorm:"type:text;not null;index"`
	Name               *string   `json:"name"`
	NameTranslationKey *string   `json:"name_translation_key"`
	Protocol           *string   `json:"protocol"`
	CanBeDeleted       bool      `json:"can_be_deleted" gorm:"default:true"`
	Type               *string   `json:"type"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

type ContactInformation struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID string    `json:"contact_id" gorm:"type:text;not null;index"`
	TypeID    uint      `json:"type_id" gorm:"not null;index"`
	Data      string    `json:"data" gorm:"not null"`
	Kind      *string   `json:"kind"`
	Pref      bool      `json:"pref" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Contact                Contact                `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	ContactInformationType ContactInformationType `json:"contact_information_type,omitempty" gorm:"foreignKey:TypeID"`
}

func (ContactInformation) TableName() string {
	return "contact_information"
}
