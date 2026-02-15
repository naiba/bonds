package dto

import "time"

type CreateLifeMetricRequest struct {
	Label string `json:"label" validate:"required" example:"Body Weight"`
}

type UpdateLifeMetricRequest struct {
	Label string `json:"label" validate:"required" example:"Body Weight"`
}

type AddLifeMetricContactRequest struct {
	ContactID string `json:"contact_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type LifeMetricResponse struct {
	ID        uint      `json:"id" example:"1"`
	VaultID   string    `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label     string    `json:"label" example:"Body Weight"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
