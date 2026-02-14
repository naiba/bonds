package dto

import "time"

type CreateCallRequest struct {
	CalledAt     time.Time `json:"called_at" validate:"required"`
	Type         string    `json:"type" validate:"required"`
	WhoInitiated string    `json:"who_initiated" validate:"required"`
	Description  string    `json:"description"`
	Duration     *int      `json:"duration"`
	Answered     *bool     `json:"answered"`
	CallReasonID *uint     `json:"call_reason_id"`
}

type UpdateCallRequest struct {
	CalledAt     time.Time `json:"called_at" validate:"required"`
	Type         string    `json:"type" validate:"required"`
	WhoInitiated string    `json:"who_initiated" validate:"required"`
	Description  string    `json:"description"`
	Duration     *int      `json:"duration"`
	Answered     *bool     `json:"answered"`
	CallReasonID *uint     `json:"call_reason_id"`
}

type CallResponse struct {
	ID           uint      `json:"id"`
	ContactID    string    `json:"contact_id"`
	AuthorID     string    `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	CallReasonID *uint     `json:"call_reason_id"`
	CalledAt     time.Time `json:"called_at"`
	Duration     *int      `json:"duration"`
	Type         string    `json:"type"`
	Description  string    `json:"description"`
	Answered     bool      `json:"answered"`
	WhoInitiated string    `json:"who_initiated"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
