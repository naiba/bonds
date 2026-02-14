package dto

import "time"

type CreateTaskRequest struct {
	Label       string     `json:"label" validate:"required"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at"`
}

type UpdateTaskRequest struct {
	Label       string     `json:"label" validate:"required"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at"`
}

type TaskResponse struct {
	ID          uint       `json:"id"`
	ContactID   string     `json:"contact_id"`
	AuthorID    string     `json:"author_id"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
	DueAt       *time.Time `json:"due_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
