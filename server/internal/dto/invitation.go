package dto

import "time"

type CreateInvitationRequest struct {
	Email      string `json:"email" validate:"required"`
	Permission int    `json:"permission" validate:"required"`
}

type AcceptInvitationRequest struct {
	Token     string `json:"token" validate:"required"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name"`
	Password  string `json:"password" validate:"required"`
}

type InvitationResponse struct {
	ID         uint       `json:"id"`
	Email      string     `json:"email"`
	Permission int        `json:"permission"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at"`
	CreatedAt  time.Time  `json:"created_at"`
}
