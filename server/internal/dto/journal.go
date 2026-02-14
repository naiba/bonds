package dto

import "time"

type CreateJournalRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type UpdateJournalRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type JournalResponse struct {
	ID          uint      `json:"id"`
	VaultID     string    `json:"vault_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PostCount   int       `json:"post_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
