package dto

import "time"

type CreatePetRequest struct {
	PetCategoryID uint   `json:"pet_category_id" validate:"required"`
	Name          string `json:"name"`
}

type UpdatePetRequest struct {
	PetCategoryID uint   `json:"pet_category_id" validate:"required"`
	Name          string `json:"name"`
}

type PetResponse struct {
	ID            uint      `json:"id"`
	ContactID     string    `json:"contact_id"`
	PetCategoryID uint      `json:"pet_category_id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
