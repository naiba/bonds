package dto

import "time"

type CreateQuickFactRequest struct {
	Content string `json:"content" validate:"required" example:"Loves Italian food and hiking"`
}

type UpdateQuickFactRequest struct {
	Content string `json:"content" validate:"required" example:"Loves Italian food and hiking"`
}

type QuickFactResponse struct {
	ID                        uint      `json:"id" example:"1"`
	VaultQuickFactsTemplateID uint      `json:"vault_quick_facts_template_id" example:"1"`
	ContactID                 string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Content                   string    `json:"content" example:"Loves Italian food and hiking"`
	CreatedAt                 time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt                 time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
