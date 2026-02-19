package models

import "time"

// ContactSubscriptionState tracks per-subscription remote state for a contact.
// This allows a single contact to be pushed to multiple remote CardDAV servers,
// each with its own URI and ETag.
type ContactSubscriptionState struct {
	ID                        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID                 string    `json:"contact_id" gorm:"type:text;not null;uniqueIndex:idx_contact_sub"`
	AddressBookSubscriptionID string    `json:"address_book_subscription_id" gorm:"type:text;not null;uniqueIndex:idx_contact_sub"`
	DistantURI                string    `json:"distant_uri" gorm:"size:2096;not null"`
	DistantEtag               string    `json:"distant_etag" gorm:"size:256"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}
