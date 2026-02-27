package dto

import "time"

type CreateContactRequest struct {
	FirstName  string `json:"first_name" validate:"required,min=1,max=255" example:"John"`
	LastName   string `json:"last_name" validate:"max=255" example:"Doe"`
	MiddleName string `json:"middle_name" validate:"max=255" example:"Michael"`
	Nickname   string `json:"nickname" validate:"max=255" example:"Johnny"`
	MaidenName string `json:"maiden_name" validate:"max=255" example:"Smith"`
	Prefix     string `json:"prefix" validate:"max=255" example:"Mr."`
	Suffix     string `json:"suffix" validate:"max=255" example:"Jr."`
	GenderID   *uint  `json:"gender_id" example:"1"`
	PronounID  *uint  `json:"pronoun_id" example:"1"`
	TemplateID *uint  `json:"template_id" example:"1"`
	Listed     *bool  `json:"listed" example:"true"`
}

type UpdateContactRequest struct {
	FirstName  string `json:"first_name" validate:"required,min=1,max=255" example:"John"`
	LastName   string `json:"last_name" validate:"max=255" example:"Doe"`
	MiddleName string `json:"middle_name" validate:"max=255" example:"Michael"`
	Nickname   string `json:"nickname" validate:"max=255" example:"Johnny"`
	MaidenName string `json:"maiden_name" validate:"max=255" example:"Smith"`
	Prefix     string `json:"prefix" validate:"max=255" example:"Mr."`
	Suffix     string `json:"suffix" validate:"max=255" example:"Jr."`
	GenderID   *uint  `json:"gender_id" example:"1"`
	PronounID  *uint  `json:"pronoun_id" example:"1"`
	TemplateID *uint  `json:"template_id" example:"1"`
}

type ContactResponse struct {
	ID             string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	VaultID        string              `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName      string              `json:"first_name" example:"John"`
	LastName       string              `json:"last_name" example:"Doe"`
	MiddleName     string              `json:"middle_name" example:"Michael"`
	Nickname       string              `json:"nickname" example:"Johnny"`
	MaidenName     string              `json:"maiden_name" example:"Smith"`
	Prefix         string              `json:"prefix" example:"Mr."`
	Suffix         string              `json:"suffix" example:"Jr."`
	GenderID       *uint               `json:"gender_id" example:"1"`
	PronounID      *uint               `json:"pronoun_id" example:"1"`
	TemplateID     *uint               `json:"template_id" example:"1"`
	CompanyID      *uint               `json:"company_id" example:"1"`
	ReligionID     *uint               `json:"religion_id" example:"1"`
	FileID         *uint               `json:"file_id" example:"1"`
	JobPosition    string              `json:"job_position" example:"Software Engineer"`
	Listed         bool                `json:"listed" example:"true"`
	ShowQuickFacts bool                `json:"show_quick_facts" example:"false"`
	IsArchived     bool                `json:"is_archived" example:"false"`
	IsFavorite     bool                `json:"is_favorite" example:"true"`
	CreatedAt      time.Time           `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt      time.Time           `json:"updated_at" example:"2026-01-15T10:30:00Z"`
	Birthday       *string             `json:"birthday,omitempty" example:"1990-06-15"`
	Age            *int                `json:"age,omitempty" example:"35"`
	Groups         []ContactGroupBrief `json:"groups,omitempty"`
}

type ContactListResponse struct {
	Contacts []ContactResponse `json:"contacts"`
}

type AddContactLabelRequest struct {
	LabelID uint `json:"label_id" validate:"required" example:"1"`
}

type ContactLabelResponse struct {
	ID        uint      `json:"id" example:"1"`
	LabelID   uint      `json:"label_id" example:"1"`
	Name      string    `json:"name" example:"Friend"`
	BgColor   string    `json:"bg_color" example:"#3B82F6"`
	TextColor string    `json:"text_color" example:"#FFFFFF"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type UpdateContactLabelRequest struct {
	LabelID uint `json:"label_id" validate:"required" example:"1"`
}

type UpdateContactReligionRequest struct {
	ReligionID *uint `json:"religion_id" example:"1"`
}

type UpdateJobInfoRequest struct {
	CompanyID   *uint  `json:"company_id" example:"1"`
	JobPosition string `json:"job_position" example:"Software Engineer"`
}

type MoveContactRequest struct {
	TargetVaultID string `json:"target_vault_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type UpdateContactTemplateRequest struct {
	TemplateID *uint `json:"template_id" example:"1"`
}

type UpdateContactSortRequest struct {
	SortOrder string `json:"sort_order" validate:"required" example:"first_name_asc"`
}

type ContactSearchRequest struct {
	SearchTerm string `json:"search_term" validate:"required,min=1" example:"John"`
}

type ContactSearchItem struct {
	ID   string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name string `json:"name" example:"John Doe"`
}

type ContactGroupBrief struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"Family"`
}

type CreateContactJobRequest struct {
	CompanyID   uint   `json:"company_id" validate:"required" example:"1"`
	JobPosition string `json:"job_position" example:"Software Engineer"`
}

type UpdateContactJobRequest struct {
	CompanyID   uint   `json:"company_id" validate:"required" example:"1"`
	JobPosition string `json:"job_position" example:"Software Engineer"`
}

type ContactJobResponse struct {
	ID          uint      `json:"id" example:"1"`
	ContactID   string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CompanyID   uint      `json:"company_id" example:"1"`
	CompanyName string    `json:"company_name" example:"Acme Corporation"`
	JobPosition string    `json:"job_position" example:"Software Engineer"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
