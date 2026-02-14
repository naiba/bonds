package dto

import "time"

type AddContactToGroupRequest struct {
	GroupID         uint  `json:"group_id" validate:"required"`
	GroupTypeRoleID *uint `json:"group_type_role_id"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name" validate:"required"`
	GroupTypeID *uint  `json:"group_type_id"`
}

type GroupResponse struct {
	ID          uint                   `json:"id"`
	VaultID     string                 `json:"vault_id"`
	GroupTypeID *uint                  `json:"group_type_id"`
	Name        string                 `json:"name"`
	Contacts    []GroupContactResponse `json:"contacts,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type GroupContactResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
