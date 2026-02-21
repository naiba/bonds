package models

import "time"

// OAuthProvider stores a configured OAuth/OIDC identity provider.
type OAuthProvider struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Type         string    `json:"type" gorm:"not null;type:text"`             // github, google, gitlab, discord, oidc
	Name         string    `json:"name" gorm:"uniqueIndex;not null;type:text"` // unique slug used in callback URL
	ClientID     string    `json:"client_id" gorm:"not null;type:text"`
	ClientSecret string    `json:"client_secret" gorm:"not null;type:text"`
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	DisplayName  string    `json:"display_name" gorm:"type:text"`  // shown on login page
	DiscoveryURL string    `json:"discovery_url" gorm:"type:text"` // for OIDC only
	Scopes       string    `json:"scopes" gorm:"type:text"`        // comma-separated custom scopes
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
