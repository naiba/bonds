package models

import "time"

type ContactFeedItem struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AuthorID     *string   `json:"author_id" gorm:"type:text;index"`
	ContactID    string    `json:"contact_id" gorm:"type:text;not null;index"`
	Action       string    `json:"action" gorm:"not null"`
	Description  *string   `json:"description"`
	FeedableID   *uint     `json:"feedable_id" gorm:"index:idx_feedable"`
	FeedableType *string   `json:"feedable_type" gorm:"index:idx_feedable"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Author  *User   `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Contact Contact `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
}
