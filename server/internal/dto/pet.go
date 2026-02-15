package dto

import "time"

type CreatePetRequest struct {
	PetCategoryID uint   `json:"pet_category_id" validate:"required" example:"1"`
	Name          string `json:"name" example:"Buddy"`
}

type UpdatePetRequest struct {
	PetCategoryID uint   `json:"pet_category_id" validate:"required" example:"1"`
	Name          string `json:"name" example:"Buddy"`
}

type PetResponse struct {
	ID            uint      `json:"id" example:"1"`
	ContactID     string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PetCategoryID uint      `json:"pet_category_id" example:"1"`
	Name          string    `json:"name" example:"Buddy"`
	CreatedAt     time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
