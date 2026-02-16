package dto

import "time"

type AddContactToGroupRequest struct {
	GroupID         uint  `json:"group_id" validate:"required" example:"1"`
	GroupTypeRoleID *uint `json:"group_type_role_id" example:"1"`
}

type CreateGroupRequest struct {
	Name string `json:"name" validate:"required" example:"Close Friends"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name" validate:"required" example:"Book Club"`
	GroupTypeID *uint  `json:"group_type_id" example:"1"`
}

type GroupResponse struct {
	ID          uint                   `json:"id" example:"1"`
	VaultID     string                 `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	GroupTypeID *uint                  `json:"group_type_id" example:"1"`
	Name        string                 `json:"name" example:"Book Club"`
	Contacts    []GroupContactResponse `json:"contacts,omitempty"`
	CreatedAt   time.Time              `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time              `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type GroupContactResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
}
