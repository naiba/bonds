package dto

import "time"

type CreateNoteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body" validate:"required"`
}

type UpdateNoteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body" validate:"required"`
}

type NoteResponse struct {
	ID        uint      `json:"id"`
	ContactID string    `json:"contact_id"`
	VaultID   string    `json:"vault_id"`
	AuthorID  string    `json:"author_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
