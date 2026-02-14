package dto

import "time"

type VaultTaskResponse struct {
	ID          uint       `json:"id"`
	ContactID   string     `json:"contact_id"`
	AuthorName  string     `json:"author_name"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
	DueAt       *time.Time `json:"due_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
