package models

import "time"

type AddressType struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID          string    `json:"account_id" gorm:"type:text;not null;index"`
	Name               *string   `json:"name"`
	Type               *string   `json:"type"`
	NameTranslationKey *string   `json:"name_translation_key"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

type Address struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID       string    `json:"vault_id" gorm:"type:text;not null;index"`
	AddressTypeID *uint     `json:"address_type_id" gorm:"index"`
	Line1         *string   `json:"line_1"`
	Line2         *string   `json:"line_2"`
	City          *string   `json:"city"`
	Province      *string   `json:"province"`
	PostalCode    *string   `json:"postal_code"`
	Country       *string   `json:"country"`
	Latitude      *float64  `json:"latitude"`
	Longitude     *float64  `json:"longitude"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Vault       Vault        `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	AddressType *AddressType `json:"address_type,omitempty" gorm:"foreignKey:AddressTypeID"`
	Contacts    []Contact    `json:"contacts,omitempty" gorm:"many2many:contact_address"`
}

type ContactAddress struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID     string    `json:"contact_id" gorm:"type:text;not null;index"`
	AddressID     uint      `json:"address_id" gorm:"not null;index"`
	IsPastAddress bool      `json:"is_past_address" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ContactAddress) TableName() string {
	return "contact_address"
}
