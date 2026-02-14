package models

import "time"

type Label struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID     string    `json:"vault_id" gorm:"type:text;not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Slug        string    `json:"slug" gorm:"not null"`
	Description *string   `json:"description" gorm:"type:text"`
	BgColor     string    `json:"bg_color" gorm:"default:'bg-zinc-200'"`
	TextColor   string    `json:"text_color" gorm:"default:'text-zinc-700'"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Vault    Vault     `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Contacts []Contact `json:"contacts,omitempty" gorm:"many2many:contact_label"`
}

type ContactLabel struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	LabelID   uint      `json:"label_id" gorm:"not null;index"`
	ContactID string    `json:"contact_id" gorm:"type:text;not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ContactLabel) TableName() string {
	return "contact_label"
}
