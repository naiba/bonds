package models

import "time"

type Template struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID          string    `json:"account_id" gorm:"type:text;not null;index"`
	Name               *string   `json:"name"`
	NameTranslationKey *string   `json:"name_translation_key"`
	CanBeDeleted       bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Account  Account        `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	Pages    []TemplatePage `json:"pages,omitempty" gorm:"foreignKey:TemplateID"`
	Contacts []Contact      `json:"contacts,omitempty" gorm:"foreignKey:TemplateID"`
}

type TemplatePage struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TemplateID         uint      `json:"template_id" gorm:"not null;index"`
	Name               *string   `json:"name"`
	NameTranslationKey *string   `json:"name_translation_key"`
	Slug               string    `json:"slug" gorm:"not null"`
	Position           *int      `json:"position"`
	Type               *string   `json:"type"`
	CanBeDeleted       bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Template Template `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Modules  []Module `json:"modules,omitempty" gorm:"many2many:module_template_page"`
}
