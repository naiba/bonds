package dto

import "time"

type CreateImportantDateRequest struct {
	Label                      string `json:"label" example:"Birthday"`
	DatePrecision              string `json:"date_precision" example:"full"`
	Day                        *int   `json:"day" example:"15"`
	Month                      *int   `json:"month" example:"6"`
	Year                       *int   `json:"year" example:"1990"`
	CalendarType               string `json:"calendar_type" example:"gregorian"`
	OriginalDay                *int   `json:"original_day" example:"15"`
	OriginalMonth              *int   `json:"original_month" example:"6"`
	OriginalYear               *int   `json:"original_year" example:"1990"`
	ContactImportantDateTypeID *uint  `json:"contact_important_date_type_id" example:"1"`
	RemindMe                   *bool  `json:"remind_me" example:"true"`
}

type UpdateImportantDateRequest struct {
	Label                      string `json:"label" example:"Birthday"`
	DatePrecision              string `json:"date_precision" example:"full"`
	Day                        *int   `json:"day" example:"15"`
	Month                      *int   `json:"month" example:"6"`
	Year                       *int   `json:"year" example:"1990"`
	CalendarType               string `json:"calendar_type" example:"gregorian"`
	OriginalDay                *int   `json:"original_day" example:"15"`
	OriginalMonth              *int   `json:"original_month" example:"6"`
	OriginalYear               *int   `json:"original_year" example:"1990"`
	ContactImportantDateTypeID *uint  `json:"contact_important_date_type_id" example:"1"`
	RemindMe                   *bool  `json:"remind_me" example:"true"`
}

// UpdateImportantDateWithIDRequest identifies one existing important date and
// carries its desired value. Contact profile updates use this shape so the
// contact row and all date/reminder changes can commit atomically.
type UpdateImportantDateWithIDRequest struct {
	ID            uint                       `json:"id" validate:"required" example:"1"`
	ImportantDate UpdateImportantDateRequest `json:"important_date"`
}

// ImportantDateChangesRequest is the mutation set submitted with a contact
// profile update. Separate create/update/delete collections keep the generated
// API contract explicit and avoid ambiguous optional IDs.
type ImportantDateChangesRequest struct {
	Create []CreateImportantDateRequest       `json:"create"`
	Update []UpdateImportantDateWithIDRequest `json:"update"`
	Delete []uint                             `json:"delete" example:"1,2"`
}

type ImportantDateResponse struct {
	ID                         uint      `json:"id" example:"1"`
	ContactID                  string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label                      string    `json:"label" example:"Birthday"`
	DatePrecision              string    `json:"date_precision" example:"full"`
	Day                        *int      `json:"day" example:"15"`
	Month                      *int      `json:"month" example:"6"`
	Year                       *int      `json:"year" example:"1990"`
	CalendarType               string    `json:"calendar_type" example:"gregorian"`
	OriginalDay                *int      `json:"original_day" example:"15"`
	OriginalMonth              *int      `json:"original_month" example:"6"`
	OriginalYear               *int      `json:"original_year" example:"1990"`
	ContactImportantDateTypeID *uint     `json:"contact_important_date_type_id" example:"1"`
	RemindMe                   bool      `json:"remind_me" example:"false"`
	CreatedAt                  time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt                  time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
