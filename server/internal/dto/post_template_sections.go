package dto

import "time"

type CreatePostTemplateSectionRequest struct {
	Label    string `json:"label" validate:"required"`
	Position int    `json:"position"`
}

type UpdatePostTemplateSectionRequest struct {
	Label    string `json:"label" validate:"required"`
	Position int    `json:"position"`
}

type UpdatePositionRequest struct {
	Position int `json:"position"`
}

type PostTemplateSectionResponse struct {
	ID             uint      `json:"id"`
	PostTemplateID uint      `json:"post_template_id"`
	Label          string    `json:"label"`
	Position       int       `json:"position"`
	CanBeDeleted   bool      `json:"can_be_deleted"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
