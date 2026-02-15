package dto

import "time"

type CreateLifeMetricRequest struct {
	Label string `json:"label" validate:"required"`
}

type UpdateLifeMetricRequest struct {
	Label string `json:"label" validate:"required"`
}

type AddLifeMetricContactRequest struct {
	ContactID string `json:"contact_id" validate:"required"`
}

type LifeMetricResponse struct {
	ID        uint      `json:"id"`
	VaultID   string    `json:"vault_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
