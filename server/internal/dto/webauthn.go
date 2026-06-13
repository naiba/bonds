package dto

import "time"

type WebAuthnLoginBeginRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type WebAuthnCredentialResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"YubiKey 5"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
