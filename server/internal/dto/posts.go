package dto

import "time"

type CreatePostRequest struct {
	Title     string             `json:"title"`
	Published bool               `json:"published"`
	WrittenAt time.Time          `json:"written_at" validate:"required"`
	Sections  []PostSectionInput `json:"sections"`
}

type UpdatePostRequest struct {
	Title     string             `json:"title"`
	Published bool               `json:"published"`
	WrittenAt time.Time          `json:"written_at"`
	Sections  []PostSectionInput `json:"sections"`
}

type PostSectionInput struct {
	Position int    `json:"position"`
	Label    string `json:"label"`
	Content  string `json:"content"`
}

type PostResponse struct {
	ID        uint                  `json:"id"`
	JournalID uint                  `json:"journal_id"`
	Title     string                `json:"title"`
	Published bool                  `json:"published"`
	WrittenAt time.Time             `json:"written_at"`
	ViewCount int                   `json:"view_count"`
	Sections  []PostSectionResponse `json:"sections,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

type PostSectionResponse struct {
	ID       uint   `json:"id"`
	Position int    `json:"position"`
	Label    string `json:"label"`
	Content  string `json:"content"`
}
