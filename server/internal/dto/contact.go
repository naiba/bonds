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

type AddContactLabelRequest struct {
	LabelID uint `json:"label_id" validate:"required"`
}

type ContactLabelResponse struct {
	ID        uint      `json:"id"`
	LabelID   uint      `json:"label_id"`
	Name      string    `json:"name"`
	BgColor   string    `json:"bg_color"`
	TextColor string    `json:"text_color"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateContactLabelRequest struct {
	LabelID uint `json:"label_id" validate:"required"`
}

type UpdateContactReligionRequest struct {
	ReligionID *uint `json:"religion_id"`
}

type UpdateJobInfoRequest struct {
	CompanyID   *uint  `json:"company_id"`
	JobPosition string `json:"job_position"`
}

type MoveContactRequest struct {
	TargetVaultID string `json:"target_vault_id" validate:"required"`
}

type UpdateContactTemplateRequest struct {
	TemplateID *uint `json:"template_id"`
}

type UpdateContactSortRequest struct {
	SortOrder string `json:"sort_order" validate:"required"`
}

type ContactSearchRequest struct {
	SearchTerm string `json:"search_term" validate:"required,min=1"`
}

type ContactSearchItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
