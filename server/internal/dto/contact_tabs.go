package dto

// ContactTabsResponse represents the template structure with pages and modules for a contact.
type ContactTabsResponse struct {
	TemplateID   uint             `json:"template_id" example:"1"`
	TemplateName string           `json:"template_name" example:"Default template"`
	Pages        []ContactTabPage `json:"pages"`
}

// ContactTabPage represents a template page with its modules.
type ContactTabPage struct {
	ID       uint               `json:"id" example:"1"`
	Name     string             `json:"name" example:"Contact information"`
	Slug     string             `json:"slug" example:"contact"`
	Position int                `json:"position" example:"1"`
	Type     string             `json:"type" example:"contact"`
	Modules  []ContactTabModule `json:"modules"`
}

// ContactTabModule represents a module assigned to a template page.
type ContactTabModule struct {
	ID       uint   `json:"id" example:"1"`
	Name     string `json:"name" example:"Notes"`
	Type     string `json:"type" example:"notes"`
	Position int    `json:"position" example:"1"`
}
