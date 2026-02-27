package dto

import "time"

type CreateCompanyRequest struct {
	Name string `json:"name" validate:"required" example:"Acme Corporation"`
	Type string `json:"type" example:"employer"`
}

type UpdateCompanyRequest struct {
	Name string `json:"name" validate:"required" example:"Acme Corporation"`
	Type string `json:"type" example:"employer"`
}

type CompanyResponse struct {
	ID        uint                  `json:"id" example:"1"`
	VaultID   string                `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string                `json:"name" example:"Acme Corporation"`
	Type      string                `json:"type" example:"employer"`
	Contacts  []CompanyContactBrief `json:"contacts,omitempty"`
	CreatedAt time.Time             `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time             `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type CompanyContactBrief struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName   string `json:"first_name" example:"John"`
	LastName    string `json:"last_name" example:"Doe"`
	JobPosition string `json:"job_position" example:"Engineer"`
	JobID       uint   `json:"job_id" example:"1"`
}

type AddEmployeeRequest struct {
	ContactID   string `json:"contact_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	JobPosition string `json:"job_position" example:"Software Engineer"`
}
