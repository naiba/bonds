package dto

import "time"

type CreateGroupTypeRoleRequest struct {
	Label    string `json:"label" validate:"required"`
	Position *int   `json:"position"`
}

type UpdateGroupTypeRoleRequest struct {
	Label    string `json:"label" validate:"required"`
	Position *int   `json:"position"`
}

type GroupTypeRoleResponse struct {
	ID          uint      `json:"id"`
	GroupTypeID uint      `json:"group_type_id"`
	Label       string    `json:"label"`
	Position    *int      `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
