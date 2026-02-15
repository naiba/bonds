package dto

import "time"

type CreateJournalRequest struct {
	Name        string `json:"name" validate:"required" example:"Daily Reflections"`
	Description string `json:"description" example:"My personal daily journal"`
}

type UpdateJournalRequest struct {
	Name        string `json:"name" validate:"required" example:"Daily Reflections"`
	Description string `json:"description" example:"My personal daily journal"`
}

type JournalResponse struct {
	ID          uint      `json:"id" example:"1"`
	VaultID     string    `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string    `json:"name" example:"Daily Reflections"`
	Description string    `json:"description" example:"My personal daily journal"`
	PostCount   int       `json:"post_count" example:"10"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
