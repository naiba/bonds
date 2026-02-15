package dto

import "time"

type CreateNoteRequest struct {
	Title string `json:"title" example:"Meeting Notes"`
	Body  string `json:"body" validate:"required" example:"Discussed project timeline and deliverables"`
}

type UpdateNoteRequest struct {
	Title string `json:"title" example:"Meeting Notes"`
	Body  string `json:"body" validate:"required" example:"Discussed project timeline and deliverables"`
}

type NoteResponse struct {
	ID        uint      `json:"id" example:"1"`
	ContactID string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	VaultID   string    `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID  string    `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title     string    `json:"title" example:"Meeting Notes"`
	Body      string    `json:"body" example:"Discussed project timeline and deliverables"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
