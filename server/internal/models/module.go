package models

import "time"

type Module struct {
	ID                           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID                    string    `json:"account_id" gorm:"type:text;not null;index"`
	Name                         *string   `json:"name"`
	NameTranslationKey           *string   `json:"name_translation_key"`
	Type                         *string   `json:"type"`
	ReservedToContactInformation bool      `json:"reserved_to_contact_information" gorm:"default:false"`
	CanBeDeleted                 bool      `json:"can_be_deleted" gorm:"default:true"`
	Pagination                   *int      `json:"pagination"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`

	Account       Account        `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	Rows          []ModuleRow    `json:"rows,omitempty" gorm:"foreignKey:ModuleID"`
	TemplatePages []TemplatePage `json:"template_pages,omitempty" gorm:"many2many:module_template_page"`
}

type ModuleRow struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ModuleID  uint      `json:"module_id" gorm:"not null;index"`
	Position  *int      `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Module Module           `json:"module,omitempty" gorm:"foreignKey:ModuleID"`
	Fields []ModuleRowField `json:"fields,omitempty" gorm:"foreignKey:ModuleRowID"`
}

type ModuleRowField struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ModuleRowID     uint      `json:"module_row_id" gorm:"not null;index"`
	Label           string    `json:"label" gorm:"not null"`
	ModuleFieldType string    `json:"module_field_type" gorm:"not null"`
	Required        bool      `json:"required" gorm:"default:false"`
	Position        *int      `json:"position"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Row ModuleRow `json:"row,omitempty" gorm:"foreignKey:ModuleRowID"`
}

type ModuleTemplatePage struct {
	TemplatePageID uint      `json:"template_page_id" gorm:"not null;index"`
	ModuleID       uint      `json:"module_id" gorm:"not null;index"`
	Position       *int      `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (ModuleTemplatePage) TableName() string {
	return "module_template_page"
}
