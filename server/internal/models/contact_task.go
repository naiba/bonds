package models

import (
	"time"

	"gorm.io/gorm"
)

type ContactTask struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID   string         `json:"contact_id" gorm:"type:text;not null;index"`
	AuthorID    *string        `json:"author_id" gorm:"type:text;index"`
	UUID        *string        `json:"uuid" gorm:"type:text;index"`
	Vcalendar   *string        `json:"vcalendar" gorm:"type:text"`
	DistantUUID *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI  *string        `json:"distant_uri" gorm:"size:2096"`
	AuthorName  string         `json:"author_name" gorm:"not null"`
	Label       string         `json:"label" gorm:"not null"`
	Description *string        `json:"description" gorm:"type:text"`
	Completed   bool           `json:"completed" gorm:"default:false"`
	CompletedAt *time.Time     `json:"completed_at"`
	DueAt       *time.Time     `json:"due_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	Contact Contact `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Author  *User   `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
}
