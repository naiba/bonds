package dto

import "time"

type UpdateNumberFormatRequest struct {
	NumberFormat string `json:"number_format" validate:"required" example:"1,234.56"`
}

type UpdateDistanceFormatRequest struct {
	DistanceFormat string `json:"distance_format" validate:"required" example:"km"`
}

type UpdateMapsPreferenceRequest struct {
	DefaultMapSite string `json:"default_map_site" validate:"required" example:"google_maps"`
}

type UpdateHelpShownRequest struct {
	HelpShown bool `json:"help_shown" example:"true"`
}

type NotificationLogResponse struct {
	ID          uint      `json:"id" example:"1"`
	SentAt      time.Time `json:"sent_at" example:"2026-01-15T10:30:00Z"`
	SubjectLine string    `json:"subject_line" example:"Reminder: Birthday of John Doe"`
	Payload     string    `json:"payload" example:"{contact_name: John Doe, label: Birthday}"`
	Error       string    `json:"error" example:""`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type UpdateManagedUserRequest struct {
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
	IsAdmin   bool   `json:"is_admin" example:"true"`
}

type UserManagementResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string    `json:"email" example:"user@example.com"`
	FirstName string    `json:"first_name" example:"John"`
	LastName  string    `json:"last_name" example:"Doe"`
	IsAdmin   bool      `json:"is_admin" example:"true"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type CancelAccountRequest struct {
	Password string `json:"password" validate:"required" example:"secureP@ss123"`
}

type StorageResponse struct {
	UsedBytes  int64 `json:"used_bytes" example:"5242880"`
	LimitBytes int64 `json:"limit_bytes" example:"10737418240"`
}

type CurrencyResponse struct {
	ID   uint   `json:"id" example:"1"`
	Code string `json:"code" example:"USD"`
}
