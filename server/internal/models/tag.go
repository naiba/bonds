package models

import "time"

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	Slug      string    `json:"slug" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Vault Vault  `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Posts []Post `json:"posts,omitempty" gorm:"many2many:post_tag"`
}
