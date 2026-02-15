package dto

import "time"

type CreateVaultRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255" example:"Family"`
	Description string `json:"description" validate:"max=1024" example:"Vault for family contacts"`
}

type UpdateVaultRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255" example:"Family"`
	Description string `json:"description" validate:"max=1024" example:"Vault for family contacts"`
}

type VaultResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountID   string    `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string    `json:"name" example:"Family"`
	Description string    `json:"description" example:"Vault for family contacts"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
