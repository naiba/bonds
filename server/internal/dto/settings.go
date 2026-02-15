package dto

import "time"

type PreferencesResponse struct {
	NameOrder      string `json:"name_order" example:"first_last"`
	DateFormat     string `json:"date_format" example:"YYYY-MM-DD"`
	Timezone       string `json:"timezone" example:"America/New_York"`
	Locale         string `json:"locale" example:"en"`
	NumberFormat   string `json:"number_format" example:"1,234.56"`
	DistanceFormat string `json:"distance_format" example:"km"`
	DefaultMapSite string `json:"default_map_site" example:"google_maps"`
	HelpShown      bool   `json:"help_shown" example:"true"`
}

type UpdateNameOrderRequest struct {
	NameOrder string `json:"name_order" validate:"required" example:"first_last"`
}

type UpdateDateFormatRequest struct {
	DateFormat string `json:"date_format" validate:"required" example:"YYYY-MM-DD"`
}

type UpdateTimezoneRequest struct {
	Timezone string `json:"timezone" validate:"required" example:"America/New_York"`
}

type UpdateLocaleRequest struct {
	Locale string `json:"locale" validate:"required" example:"en"`
}

type UpdatePreferencesRequest struct {
	NameOrder      string `json:"name_order" example:"first_last"`
	DateFormat     string `json:"date_format" example:"YYYY-MM-DD"`
	Timezone       string `json:"timezone" example:"America/New_York"`
	Locale         string `json:"locale" example:"en"`
	NumberFormat   string `json:"number_format" example:"1,234.56"`
	DistanceFormat string `json:"distance_format" example:"km"`
	DefaultMapSite string `json:"default_map_site" example:"google_maps"`
	HelpShown      *bool  `json:"help_shown" example:"true"`
}

type CreateNotificationChannelRequest struct {
	Type          string `json:"type" validate:"required" example:"email"`
	Label         string `json:"label" example:"Personal Email"`
	Content       string `json:"content" validate:"required" example:"user@example.com"`
	PreferredTime string `json:"preferred_time" example:"09:00"`
}

type NotificationChannelResponse struct {
	ID            uint       `json:"id" example:"1"`
	Type          string     `json:"type" example:"email"`
	Label         string     `json:"label" example:"Personal Email"`
	Content       string     `json:"content" example:"user@example.com"`
	PreferredTime string     `json:"preferred_time" example:"09:00"`
	Active        bool       `json:"active" example:"true"`
	VerifiedAt    *time.Time `json:"verified_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt     time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt     time.Time  `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type PersonalizeEntityRequest struct {
	Label    string `json:"label" example:"Male"`
	Name     string `json:"name" example:"genders"`
	Position *int   `json:"position" example:"1"`
}

type PersonalizeEntityResponse struct {
	ID        uint      `json:"id" example:"1"`
	Label     string    `json:"label" example:"Male"`
	Name      string    `json:"name" example:"genders"`
	Position  *int      `json:"position" example:"1"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
