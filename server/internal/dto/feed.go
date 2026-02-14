package dto

import "time"

type FeedItemResponse struct {
	ID          uint      `json:"id"`
	ContactID   string    `json:"contact_id"`
	AuthorID    string    `json:"author_id"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
