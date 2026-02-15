package dto

import "time"

type CreateCallRequest struct {
	CalledAt     time.Time `json:"called_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Type         string    `json:"type" validate:"required" example:"phone"`
	WhoInitiated string    `json:"who_initiated" validate:"required" example:"me"`
	Description  string    `json:"description" example:"Discussed upcoming vacation plans"`
	Duration     *int      `json:"duration" example:"30"`
	Answered     *bool     `json:"answered" example:"true"`
	CallReasonID *uint     `json:"call_reason_id" example:"1"`
}

type UpdateCallRequest struct {
	CalledAt     time.Time `json:"called_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Type         string    `json:"type" validate:"required" example:"phone"`
	WhoInitiated string    `json:"who_initiated" validate:"required" example:"me"`
	Description  string    `json:"description" example:"Discussed upcoming vacation plans"`
	Duration     *int      `json:"duration" example:"30"`
	Answered     *bool     `json:"answered" example:"true"`
	CallReasonID *uint     `json:"call_reason_id" example:"1"`
}

type CallResponse struct {
	ID           uint      `json:"id" example:"1"`
	ContactID    string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID     string    `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorName   string    `json:"author_name" example:"John Doe"`
	CallReasonID *uint     `json:"call_reason_id" example:"1"`
	CalledAt     time.Time `json:"called_at" example:"2026-01-15T10:30:00Z"`
	Duration     *int      `json:"duration" example:"30"`
	Type         string    `json:"type" example:"phone"`
	Description  string    `json:"description" example:"Discussed upcoming vacation plans"`
	Answered     bool      `json:"answered" example:"true"`
	WhoInitiated string    `json:"who_initiated" example:"me"`
	CreatedAt    time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt    time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
