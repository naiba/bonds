package dto

import "time"

type CompanyResponse struct {
	ID        uint                  `json:"id"`
	VaultID   string                `json:"vault_id"`
	Name      string                `json:"name"`
	Type      string                `json:"type"`
	Contacts  []CompanyContactBrief `json:"contacts,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

type CompanyContactBrief struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
