package models

import "time"

// DavSyncLog records each sync action for audit/debugging
type DavSyncLog struct {
	ID                        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AddressBookSubscriptionID string    `json:"address_book_subscription_id" gorm:"type:text;not null;index"`
	ContactID                 *string   `json:"contact_id" gorm:"type:text;index"`
	DistantURI                string    `json:"distant_uri" gorm:"size:2096"`
	DistantEtag               string    `json:"distant_etag" gorm:"size:256"`
	Action                    string    `json:"action" gorm:"size:32;not null"` // "created", "updated", "skipped", "deleted", "error"
	ErrorMessage              *string   `json:"error_message" gorm:"type:text"`
	CreatedAt                 time.Time `json:"created_at"`
}
