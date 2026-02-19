package dto

import "time"

type CreateImportantDateRequest struct {
	Label                      string `json:"label" example:"Birthday"`
	Day                        *int   `json:"day" example:"15"`
	Month                      *int   `json:"month" example:"6"`
	Year                       *int   `json:"year" example:"1990"`
	CalendarType               string `json:"calendar_type" example:"gregorian"`
	OriginalDay                *int   `json:"original_day" example:"15"`
	OriginalMonth              *int   `json:"original_month" example:"6"`
	OriginalYear               *int   `json:"original_year" example:"1990"`
	ContactImportantDateTypeID *uint  `json:"contact_important_date_type_id" example:"1"`
}

type UpdateImportantDateRequest struct {
	Label                      string `json:"label" example:"Birthday"`
	Day                        *int   `json:"day" example:"15"`
	Month                      *int   `json:"month" example:"6"`
	Year                       *int   `json:"year" example:"1990"`
	CalendarType               string `json:"calendar_type" example:"gregorian"`
	OriginalDay                *int   `json:"original_day" example:"15"`
	OriginalMonth              *int   `json:"original_month" example:"6"`
	OriginalYear               *int   `json:"original_year" example:"1990"`
	ContactImportantDateTypeID *uint  `json:"contact_important_date_type_id" example:"1"`
}

type ImportantDateResponse struct {
	ID                         uint      `json:"id" example:"1"`
	ContactID                  string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label                      string    `json:"label" example:"Birthday"`
	Day                        *int      `json:"day" example:"15"`
	Month                      *int      `json:"month" example:"6"`
	Year                       *int      `json:"year" example:"1990"`
	CalendarType               string    `json:"calendar_type" example:"gregorian"`
	OriginalDay                *int      `json:"original_day" example:"15"`
	OriginalMonth              *int      `json:"original_month" example:"6"`
	OriginalYear               *int      `json:"original_year" example:"1990"`
	ContactImportantDateTypeID *uint     `json:"contact_important_date_type_id" example:"1"`
	CreatedAt                  time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt                  time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
