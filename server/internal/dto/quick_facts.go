package dto

import "time"

type CreateQuickFactRequest struct {
	Content     string   `json:"content" example:"Loves Italian food and hiking"`
	ValueText   *string  `json:"value_text" example:"Loves Italian food and hiking"`
	ValueNumber *float64 `json:"value_number" example:"42"`
	ValueDate   *string  `json:"value_date" example:"2026-01-15"`
	ValueOption *string  `json:"value_option" example:"Yes"`
}

type UpdateQuickFactRequest struct {
	Content     string   `json:"content" example:"Loves Italian food and hiking"`
	ValueText   *string  `json:"value_text" example:"Loves Italian food and hiking"`
	ValueNumber *float64 `json:"value_number" example:"42"`
	ValueDate   *string  `json:"value_date" example:"2026-01-15"`
	ValueOption *string  `json:"value_option" example:"Yes"`
}

type QuickFactResponse struct {
	ID                        uint                   `json:"id" example:"1"`
	VaultQuickFactsTemplateID uint                   `json:"vault_quick_facts_template_id" example:"1"`
	ContactID                 string                 `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Content                   string                 `json:"content" example:"Loves Italian food and hiking"`
	FieldType                 string                 `json:"field_type" example:"text"`
	ValueText                 *string                `json:"value_text,omitempty" example:"Loves Italian food and hiking"`
	ValueNumber               *float64               `json:"value_number,omitempty" example:"42"`
	ValueDate                 *string                `json:"value_date,omitempty" example:"2026-01-15"`
	ValueOption               *string                `json:"value_option,omitempty" example:"Yes"`
	FileID                    *uint                  `json:"file_id,omitempty" example:"1"`
	File                      *QuickFactFileResponse `json:"file,omitempty"`
	CreatedAt                 time.Time              `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt                 time.Time              `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type QuickFactFileResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"photo.jpg"`
	MimeType  string    `json:"mime_type" example:"image/jpeg"`
	Type      string    `json:"type" example:"photo"`
	Size      int       `json:"size" example:"1048576"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type QuickFactGroupResponse struct {
	TemplateID    uint                `json:"template_id" example:"1"`
	TemplateLabel string              `json:"template_label" example:"Hobbies"`
	FieldType     string              `json:"field_type" example:"text"`
	SelectOptions []string            `json:"select_options,omitempty" example:"Yes,No"`
	Required      bool                `json:"required" example:"false"`
	HelpText      *string             `json:"help_text,omitempty" example:"Add one useful detail"`
	DefaultValue  *string             `json:"default_value,omitempty" example:"Unknown"`
	Position      int                 `json:"position" example:"1"`
	Facts         []QuickFactResponse `json:"facts"`
}
