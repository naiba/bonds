package dto

import "time"

type PreferencesResponse struct {
	NameOrder  string `json:"name_order"`
	DateFormat string `json:"date_format"`
	Timezone   string `json:"timezone"`
	Locale     string `json:"locale"`
}

type UpdateNameOrderRequest struct {
	NameOrder string `json:"name_order" validate:"required"`
}

type UpdateDateFormatRequest struct {
	DateFormat string `json:"date_format" validate:"required"`
}

type UpdateTimezoneRequest struct {
	Timezone string `json:"timezone" validate:"required"`
}

type UpdateLocaleRequest struct {
	Locale string `json:"locale" validate:"required"`
}

type UpdatePreferencesRequest struct {
	NameOrder  string `json:"name_order"`
	DateFormat string `json:"date_format"`
	Timezone   string `json:"timezone"`
	Locale     string `json:"locale"`
}

type CreateNotificationChannelRequest struct {
	Type          string `json:"type" validate:"required"`
	Label         string `json:"label"`
	Content       string `json:"content" validate:"required"`
	PreferredTime string `json:"preferred_time"`
}

type NotificationChannelResponse struct {
	ID            uint       `json:"id"`
	Type          string     `json:"type"`
	Label         string     `json:"label"`
	Content       string     `json:"content"`
	PreferredTime string     `json:"preferred_time"`
	Active        bool       `json:"active"`
	VerifiedAt    *time.Time `json:"verified_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PersonalizeEntityRequest struct {
	Label    string `json:"label"`
	Name     string `json:"name"`
	Position *int   `json:"position"`
}

type PersonalizeEntityResponse struct {
	ID        uint      `json:"id"`
	Label     string    `json:"label"`
	Name      string    `json:"name"`
	Position  *int      `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
