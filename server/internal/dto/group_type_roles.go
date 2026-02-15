package dto

import "time"

type CreateGroupTypeRoleRequest struct {
	Label    string `json:"label" validate:"required" example:"Leader"`
	Position *int   `json:"position" example:"1"`
}

type UpdateGroupTypeRoleRequest struct {
	Label    string `json:"label" validate:"required" example:"Leader"`
	Position *int   `json:"position" example:"1"`
}

type GroupTypeRoleResponse struct {
	ID          uint      `json:"id" example:"1"`
	GroupTypeID uint      `json:"group_type_id" example:"1"`
	Label       string    `json:"label" example:"Leader"`
	Position    *int      `json:"position" example:"1"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
