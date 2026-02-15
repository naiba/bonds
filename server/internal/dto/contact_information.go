package dto

import "time"

type CreateContactInformationRequest struct {
	TypeID uint   `json:"type_id" validate:"required" example:"1"`
	Data   string `json:"data" validate:"required" example:"+1-555-0123"`
	Kind   string `json:"kind" example:"personal"`
	Pref   *bool  `json:"pref" example:"true"`
}

type UpdateContactInformationRequest struct {
	TypeID uint   `json:"type_id" validate:"required" example:"1"`
	Data   string `json:"data" validate:"required" example:"+1-555-0123"`
	Kind   string `json:"kind" example:"personal"`
	Pref   *bool  `json:"pref" example:"true"`
}

type ContactInformationResponse struct {
	ID        uint      `json:"id" example:"1"`
	ContactID string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TypeID    uint      `json:"type_id" example:"1"`
	Data      string    `json:"data" example:"+1-555-0123"`
	Kind      string    `json:"kind" example:"personal"`
	Pref      bool      `json:"pref" example:"true"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
