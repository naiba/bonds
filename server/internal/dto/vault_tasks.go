package dto

import "time"

type VaultTaskResponse struct {
	ID          uint       `json:"id" example:"1"`
	ContactID   string     `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	VaultID     string     `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContactName string     `json:"contact_name" example:"Jane Doe"`
	AuthorName  string     `json:"author_name" example:"John Doe"`
	Label       string     `json:"label" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	Status      string     `json:"status" example:"todo"`
	Position    int        `json:"position" example:"0"`
	Completed   bool       `json:"completed" example:"false"`
	CompletedAt *time.Time `json:"completed_at" example:"2026-01-15T10:30:00Z"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type CreateVaultTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	Status      string     `json:"status" example:"todo"`
	// ContactID — optional. When omitted/empty, the task is a standalone vault-level task.
	ContactID string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateVaultTaskRequest replaces the editable fields of a vault task in one
// call. ContactID may be set, cleared (empty string = standalone), or changed
// to a different contact in the same vault. Status is validated against the
// known set; an empty Status means "leave unchanged".
type UpdateVaultTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	Status      string     `json:"status" example:"in_progress"`
	// ContactID — empty string clears the link (standalone). Non-empty must
	// belong to the same vault.
	ContactID string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}
