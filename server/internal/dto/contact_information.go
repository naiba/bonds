package dto

import "time"

type CreateContactInformationRequest struct {
	TypeID uint   `json:"type_id" validate:"required"`
	Data   string `json:"data" validate:"required"`
	Kind   string `json:"kind"`
	Pref   *bool  `json:"pref"`
}

type UpdateContactInformationRequest struct {
	TypeID uint   `json:"type_id" validate:"required"`
	Data   string `json:"data" validate:"required"`
	Kind   string `json:"kind"`
	Pref   *bool  `json:"pref"`
}

type ContactInformationResponse struct {
	ID        uint      `json:"id"`
	ContactID string    `json:"contact_id"`
	TypeID    uint      `json:"type_id"`
	Data      string    `json:"data"`
	Kind      string    `json:"kind"`
	Pref      bool      `json:"pref"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
