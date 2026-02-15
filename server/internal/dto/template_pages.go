package dto

import "time"

type CreateTemplatePageRequest struct {
	Name     string `json:"name" validate:"required" example:"Contact Information"`
	Slug     string `json:"slug" validate:"required" example:"contact-info"`
	Position int    `json:"position" example:"1"`
	Type     string `json:"type" example:"contact"`
}

type UpdateTemplatePageRequest struct {
	Name     string `json:"name" validate:"required" example:"Contact Information"`
	Slug     string `json:"slug" example:"contact-info"`
	Position int    `json:"position" example:"1"`
	Type     string `json:"type" example:"contact"`
}

type TemplatePageResponse struct {
	ID                 uint      `json:"id" example:"1"`
	TemplateID         uint      `json:"template_id" example:"1"`
	Name               string    `json:"name" example:"Contact Information"`
	NameTranslationKey string    `json:"name_translation_key" example:"template_page.contact_info"`
	Slug               string    `json:"slug" example:"contact-info"`
	Position           int       `json:"position" example:"1"`
	Type               string    `json:"type" example:"contact"`
	CanBeDeleted       bool      `json:"can_be_deleted" example:"false"`
	CreatedAt          time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt          time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type AddModuleToPageRequest struct {
	ModuleID uint `json:"module_id" validate:"required" example:"1"`
	Position int  `json:"position" example:"1"`
}

type TemplatePageModuleResponse struct {
	ModuleID       uint      `json:"module_id" example:"1"`
	TemplatePageID uint      `json:"template_page_id" example:"1"`
	Position       int       `json:"position" example:"1"`
	ModuleName     string    `json:"module_name" example:"notes"`
	ModuleType     string    `json:"module_type" example:"notes"`
	CreatedAt      time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
