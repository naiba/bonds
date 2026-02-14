package dto

import "time"

type CreateQuickFactRequest struct {
	Content string `json:"content" validate:"required"`
}

type UpdateQuickFactRequest struct {
	Content string `json:"content" validate:"required"`
}

type QuickFactResponse struct {
	ID                        uint      `json:"id"`
	VaultQuickFactsTemplateID uint      `json:"vault_quick_facts_template_id"`
	ContactID                 string    `json:"contact_id"`
	Content                   string    `json:"content"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}
