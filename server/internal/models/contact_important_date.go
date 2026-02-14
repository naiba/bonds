package models

import (
	"time"

	"gorm.io/gorm"
)

type ContactImportantDateType struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID      string    `json:"vault_id" gorm:"type:text;not null;index"`
	Label        string    `json:"label" gorm:"not null"`
	InternalType *string   `json:"internal_type"`
	CanBeDeleted bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Vault Vault `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
}

type ContactImportantDate struct {
	ID                         uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID                  string         `json:"contact_id" gorm:"type:text;not null;index"`
	UUID                       *string        `json:"uuid" gorm:"type:text;index"`
	Vcalendar                  *string        `json:"vcalendar" gorm:"type:mediumtext"`
	DistantUUID                *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag                *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI                 *string        `json:"distant_uri" gorm:"size:2096"`
	ContactImportantDateTypeID *uint          `json:"contact_important_date_type_id" gorm:"index"`
	Label                      string         `json:"label" gorm:"not null"`
	Day                        *int           `json:"day"`
	Month                      *int           `json:"month"`
	Year                       *int           `json:"year"`
	DeletedAt                  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt                  time.Time      `json:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at"`

	Contact                  Contact                   `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	ContactImportantDateType *ContactImportantDateType `json:"contact_important_date_type,omitempty" gorm:"foreignKey:ContactImportantDateTypeID"`
}
