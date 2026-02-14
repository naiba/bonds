package dto

import "time"

type CreateImportantDateRequest struct {
	Label                      string `json:"label" validate:"required"`
	Day                        *int   `json:"day"`
	Month                      *int   `json:"month"`
	Year                       *int   `json:"year"`
	ContactImportantDateTypeID *uint  `json:"contact_important_date_type_id"`
}

type UpdateImportantDateRequest struct {
	Label                      string `json:"label" validate:"required"`
	Day                        *int   `json:"day"`
	Month                      *int   `json:"month"`
	Year                       *int   `json:"year"`
	ContactImportantDateTypeID *uint  `json:"contact_important_date_type_id"`
}

type ImportantDateResponse struct {
	ID                         uint      `json:"id"`
	ContactID                  string    `json:"contact_id"`
	Label                      string    `json:"label"`
	Day                        *int      `json:"day"`
	Month                      *int      `json:"month"`
	Year                       *int      `json:"year"`
	ContactImportantDateTypeID *uint     `json:"contact_important_date_type_id"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}
