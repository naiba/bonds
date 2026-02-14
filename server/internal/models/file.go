package models

import "time"

type File struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID      string    `json:"vault_id" gorm:"type:text;not null;index"`
	FileableID   *uint     `json:"fileable_id" gorm:"index:idx_fileable"`
	FileableType *string   `json:"fileable_type" gorm:"index:idx_fileable"`
	UfileableID  *string   `json:"ufileable_id" gorm:"type:text;index:idx_ufileable"`
	UUID         string    `json:"uuid" gorm:"not null"`
	OriginalURL  *string   `json:"original_url"`
	CdnURL       *string   `json:"cdn_url"`
	MimeType     string    `json:"mime_type" gorm:"not null"`
	Name         string    `json:"name" gorm:"not null"`
	Type         string    `json:"type" gorm:"not null"`
	Size         int       `json:"size" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Vault Vault `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
}
