package dto

import "time"

type CreateCallReasonRequest struct {
	Label string `json:"label" validate:"required" example:"Just to say hello"`
}

type UpdateCallReasonRequest struct {
	Label string `json:"label" validate:"required" example:"Just to say hello"`
}

type CallReasonResponse struct {
	ID               uint      `json:"id" example:"1"`
	CallReasonTypeID uint      `json:"call_reason_type_id" example:"1"`
	Label            string    `json:"label" example:"Just to say hello"`
	CreatedAt        time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt        time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
