package models

import "time"

type Note struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID string    `json:"contact_id" gorm:"type:text;not null;index"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index"`
	AuthorID  *string   `json:"author_id" gorm:"type:text;index"`
	EmotionID *uint     `json:"emotion_id" gorm:"index"`
	Title     *string   `json:"title"`
	Body      string    `json:"body" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Contact Contact  `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Vault   Vault    `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Author  *User    `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Emotion *Emotion `json:"emotion,omitempty" gorm:"foreignKey:EmotionID"`
}
