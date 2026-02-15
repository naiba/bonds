package dto

import "time"

type CreatePostRequest struct {
	Title     string             `json:"title" example:"A Wonderful Day"`
	Published bool               `json:"published" example:"true"`
	WrittenAt time.Time          `json:"written_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Sections  []PostSectionInput `json:"sections"`
}

type UpdatePostRequest struct {
	Title     string             `json:"title" example:"A Wonderful Day"`
	Published bool               `json:"published" example:"true"`
	WrittenAt time.Time          `json:"written_at" example:"2026-01-15T10:30:00Z"`
	Sections  []PostSectionInput `json:"sections"`
}

type PostSectionInput struct {
	Position int    `json:"position" example:"1"`
	Label    string `json:"label" example:"Main Body"`
	Content  string `json:"content" example:"Today was a wonderful day spent with family."`
}

type PostResponse struct {
	ID        uint                  `json:"id" example:"1"`
	JournalID uint                  `json:"journal_id" example:"1"`
	Title     string                `json:"title" example:"A Wonderful Day"`
	Published bool                  `json:"published" example:"true"`
	WrittenAt time.Time             `json:"written_at" example:"2026-01-15T10:30:00Z"`
	ViewCount int                   `json:"view_count" example:"10"`
	Sections  []PostSectionResponse `json:"sections,omitempty"`
	CreatedAt time.Time             `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time             `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type PostSectionResponse struct {
	ID       uint   `json:"id" example:"1"`
	Position int    `json:"position" example:"1"`
	Label    string `json:"label" example:"Main Body"`
	Content  string `json:"content" example:"Today was a wonderful day spent with family."`
}
