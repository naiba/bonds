package dto

import "time"

type FeedItemResponse struct {
	ID          uint      `json:"id" example:"1"`
	ContactID   string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID    string    `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Action      string    `json:"action" example:"contact_created"`
	Description string    `json:"description" example:"Contact John Doe was created"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
