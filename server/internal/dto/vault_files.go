package dto

import "time"

type VaultFileResponse struct {
	ID        uint      `json:"id" example:"1"`
	VaultID   string    `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UUID      string    `json:"uuid" example:"image/jpeg"`
	Name      string    `json:"name" example:"photo.jpg"`
	MimeType  string    `json:"mime_type" example:"image/jpeg"`
	Type      string    `json:"type" example:"photo"`
	Size      int       `json:"size" example:"1048576"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
