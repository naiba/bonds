package dto

import "time"

type CreateTemplatePageRequest struct {
	Name     string `json:"name" validate:"required"`
	Slug     string `json:"slug" validate:"required"`
	Position int    `json:"position"`
	Type     string `json:"type"`
}

type UpdateTemplatePageRequest struct {
	Name     string `json:"name" validate:"required"`
	Slug     string `json:"slug"`
	Position int    `json:"position"`
	Type     string `json:"type"`
}

type TemplatePageResponse struct {
	ID                 uint      `json:"id"`
	TemplateID         uint      `json:"template_id"`
	Name               string    `json:"name"`
	NameTranslationKey string    `json:"name_translation_key"`
	Slug               string    `json:"slug"`
	Position           int       `json:"position"`
	Type               string    `json:"type"`
	CanBeDeleted       bool      `json:"can_be_deleted"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AddModuleToPageRequest struct {
	ModuleID uint `json:"module_id" validate:"required"`
	Position int  `json:"position"`
}

type TemplatePageModuleResponse struct {
	ModuleID       uint      `json:"module_id"`
	TemplatePageID uint      `json:"template_page_id"`
	Position       int       `json:"position"`
	ModuleName     string    `json:"module_name"`
	ModuleType     string    `json:"module_type"`
	CreatedAt      time.Time `json:"created_at"`
}
