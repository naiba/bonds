package dto

import "time"

type CreatePersonalAccessTokenRequest struct {
	Name      string     `json:"name" validate:"required" example:"CI/CD Token"`
	ExpiresAt *time.Time `json:"expires_at" example:"2027-01-15T10:30:00Z"`
}

type PersonalAccessTokenResponse struct {
	ID         uint       `json:"id" example:"1"`
	Name       string     `json:"name" example:"CI/CD Token"`
	TokenHint  string     `json:"token_hint" example:"...abc123"`
	ExpiresAt  *time.Time `json:"expires_at" example:"2027-01-15T10:30:00Z"`
	LastUsedAt *time.Time `json:"last_used_at" example:"2026-02-23T08:00:00Z"`
	CreatedAt  time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type PersonalAccessTokenCreatedResponse struct {
	ID        uint       `json:"id" example:"1"`
	Name      string     `json:"name" example:"CI/CD Token"`
	Token     string     `json:"token" example:"bonds_pat_xxxxxxxxxxxxxxxxxxxx"`
	TokenHint string     `json:"token_hint" example:"...abc123"`
	ExpiresAt *time.Time `json:"expires_at" example:"2027-01-15T10:30:00Z"`
	CreatedAt time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
}