package dto

import "time"

type CreateSliceOfLifeRequest struct {
	Name        string `json:"name" validate:"required" example:"Summer 2026"`
	Description string `json:"description" example:"Memories from summer vacation"`
}

type UpdateSliceOfLifeRequest struct {
	Name        string `json:"name" validate:"required" example:"Summer 2026"`
	Description string `json:"description" example:"Memories from summer vacation"`
}

type UpdateSliceCoverRequest struct {
	FileID uint `json:"file_id" validate:"required" example:"1"`
}

type SliceOfLifeResponse struct {
	ID               uint      `json:"id" example:"1"`
	JournalID        uint      `json:"journal_id" example:"1"`
	Name             string    `json:"name" example:"Summer 2026"`
	Description      string    `json:"description" example:"Memories from summer vacation"`
	FileCoverImageID *uint     `json:"file_cover_image_id" example:"1"`
	CreatedAt        time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt        time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
