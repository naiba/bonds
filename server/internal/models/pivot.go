package models

import "time"

type UserVault struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID    string    `json:"vault_id" gorm:"type:text;not null;index"`
	UserID     string    `json:"user_id" gorm:"type:text;not null;index"`
	ContactID  string    `json:"contact_id" gorm:"type:text;not null;index"`
	Permission int       `json:"permission" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (UserVault) TableName() string {
	return "user_vault"
}

type ContactVaultUser struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID     string    `json:"contact_id" gorm:"type:text;not null;index"`
	VaultID       string    `json:"vault_id" gorm:"type:text;not null;index"`
	UserID        string    `json:"user_id" gorm:"type:text;not null;index"`
	NumberOfViews int       `json:"number_of_views" gorm:"not null"`
	IsFavorite    bool      `json:"is_favorite" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ContactVaultUser) TableName() string {
	return "contact_vault_user"
}

const (
	PermissionManager = 100
	PermissionEditor  = 200
	PermissionViewer  = 300
)
