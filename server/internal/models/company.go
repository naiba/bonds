package models

import "time"

type Company struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	Type      *string   `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Vault    Vault     `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Contacts []Contact `json:"contacts,omitempty" gorm:"foreignKey:CompanyID"`
}
