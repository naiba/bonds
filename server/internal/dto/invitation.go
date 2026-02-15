package dto

import "time"

type CreateInvitationRequest struct {
	Email      string `json:"email" validate:"required" example:"user@example.com"`
	Permission int    `json:"permission" validate:"required" example:"100"`
}

type AcceptInvitationRequest struct {
	Token     string `json:"token" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName string `json:"first_name" validate:"required" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
	Password  string `json:"password" validate:"required" example:"secureP@ss123"`
}

type InvitationResponse struct {
	ID         uint       `json:"id" example:"1"`
	Email      string     `json:"email" example:"user@example.com"`
	Permission int        `json:"permission" example:"100"`
	ExpiresAt  time.Time  `json:"expires_at" example:"2026-01-15T10:30:00Z"`
	AcceptedAt *time.Time `json:"accepted_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt  time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
