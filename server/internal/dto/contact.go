package dto

import "time"

type CreateContactRequest struct {
	FirstName string `json:"first_name" validate:"required,min=1,max=255"`
	LastName  string `json:"last_name" validate:"max=255"`
	Nickname  string `json:"nickname" validate:"max=255"`
}

type UpdateContactRequest struct {
	FirstName string `json:"first_name" validate:"required,min=1,max=255"`
	LastName  string `json:"last_name" validate:"max=255"`
	Nickname  string `json:"nickname" validate:"max=255"`
}

type ContactResponse struct {
	ID         string    `json:"id"`
	VaultID    string    `json:"vault_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Nickname   string    `json:"nickname"`
	IsArchived bool      `json:"is_archived"`
	IsFavorite bool      `json:"is_favorite"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ContactListResponse struct {
	Contacts []ContactResponse `json:"contacts"`
}
