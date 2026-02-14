package models

import "time"

type UserToken struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       string    `json:"user_id" gorm:"type:text;not null;index:idx_user_token_driver"`
	DriverID     string    `json:"driver_id" gorm:"size:256;not null;index:idx_user_token_driver"`
	Driver       string    `json:"driver" gorm:"size:50;not null;index:idx_user_token_driver"`
	Format       string    `json:"format" gorm:"size:6;not null"`
	Email        *string   `json:"email" gorm:"size:1024"`
	Token        string    `json:"token" gorm:"size:4096;not null"`
	TokenSecret  *string   `json:"token_secret" gorm:"size:2048"`
	RefreshToken *string   `json:"refresh_token" gorm:"size:2048"`
	ExpiresIn    *uint64   `json:"expires_in"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
