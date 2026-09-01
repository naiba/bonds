package models

import "time"

// GeocodingProviderConfig stores exactly one structured configuration per
// provider. Config is an encrypted JSON object when SETTINGS_ENC_KEY is set;
// provider-specific fields live inside it so adding a provider does not add
// columns or proliferate system_settings keys.
type GeocodingProviderConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Provider  string    `json:"provider" gorm:"uniqueIndex;not null;type:text"`
	Config    string    `json:"-" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
