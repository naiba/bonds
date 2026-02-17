package dto

import "time"

type CreatePostRequest struct {
	Title     string             `json:"title" example:"A Wonderful Day"`
	Published bool               `json:"published" example:"true"`
	WrittenAt time.Time          `json:"written_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Sections  []PostSectionInput `json:"sections"`
}

type UpdatePostRequest struct {
	Title      string             `json:"title" example:"A Wonderful Day"`
	Published  bool               `json:"published" example:"true"`
	WrittenAt  time.Time          `json:"written_at" example:"2026-01-15T10:30:00Z"`
	Sections   []PostSectionInput `json:"sections"`
	ContactIDs []string           `json:"contact_ids" example:"550e8400-e29b-41d4-a716-446655440000"`
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
	Contacts  []PostContactResponse `json:"contacts,omitempty"`
	CreatedAt time.Time             `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time             `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type PostSectionResponse struct {
	ID       uint   `json:"id" example:"1"`
	Position int    `json:"position" example:"1"`
	Label    string `json:"label" example:"Main Body"`
	Content  string `json:"content" example:"Today was a wonderful day spent with family."`
}

type PostContactResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
}
