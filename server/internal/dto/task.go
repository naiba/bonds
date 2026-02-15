package dto

import "time"

type CreateTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
}

type UpdateTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
}

type TaskResponse struct {
	ID          uint       `json:"id" example:"1"`
	ContactID   string     `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID    string     `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label       string     `json:"label" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	Completed   bool       `json:"completed" example:"false"`
	CompletedAt *time.Time `json:"completed_at" example:"2026-01-15T10:30:00Z"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
