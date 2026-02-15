package dto

import "time"

type CreateSliceOfLifeRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type UpdateSliceOfLifeRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type UpdateSliceCoverRequest struct {
	FileID uint `json:"file_id" validate:"required"`
}

type SliceOfLifeResponse struct {
	ID               uint      `json:"id"`
	JournalID        uint      `json:"journal_id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	FileCoverImageID *uint     `json:"file_cover_image_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
