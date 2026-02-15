package dto

import "time"

type CreatePostTemplateSectionRequest struct {
	Label    string `json:"label" validate:"required" example:"Main Body"`
	Position int    `json:"position" example:"1"`
}

type UpdatePostTemplateSectionRequest struct {
	Label    string `json:"label" validate:"required" example:"Main Body"`
	Position int    `json:"position" example:"1"`
}

type UpdatePositionRequest struct {
	Position int `json:"position" example:"1"`
}

type PostTemplateSectionResponse struct {
	ID             uint      `json:"id" example:"1"`
	PostTemplateID uint      `json:"post_template_id" example:"1"`
	Label          string    `json:"label" example:"Main Body"`
	Position       int       `json:"position" example:"1"`
	CanBeDeleted   bool      `json:"can_be_deleted" example:"false"`
	CreatedAt      time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt      time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
