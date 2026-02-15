package dto

import "time"

type UpdateNumberFormatRequest struct {
	NumberFormat string `json:"number_format" validate:"required"`
}

type UpdateDistanceFormatRequest struct {
	DistanceFormat string `json:"distance_format" validate:"required"`
}

type UpdateMapsPreferenceRequest struct {
	DefaultMapSite string `json:"default_map_site" validate:"required"`
}

type UpdateHelpShownRequest struct {
	HelpShown bool `json:"help_shown"`
}

type NotificationLogResponse struct {
	ID          uint      `json:"id"`
	SentAt      time.Time `json:"sent_at"`
	SubjectLine string    `json:"subject_line"`
	Payload     string    `json:"payload"`
	Error       string    `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateManagedUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password" validate:"required,min=6"`
	IsAdmin   bool   `json:"is_admin"`
}

type UpdateManagedUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsAdmin   bool   `json:"is_admin"`
}

type UserManagementResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CancelAccountRequest struct {
	Password string `json:"password" validate:"required"`
}

type StorageResponse struct {
	UsedBytes  int64 `json:"used_bytes"`
	LimitBytes int64 `json:"limit_bytes"`
}

type CurrencyResponse struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
}
