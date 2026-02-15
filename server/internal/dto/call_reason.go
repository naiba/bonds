package dto

import "time"

type CreateCallReasonRequest struct {
	Label string `json:"label" validate:"required"`
}

type UpdateCallReasonRequest struct {
	Label string `json:"label" validate:"required"`
}

type CallReasonResponse struct {
	ID               uint      `json:"id"`
	CallReasonTypeID uint      `json:"call_reason_type_id"`
	Label            string    `json:"label"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
