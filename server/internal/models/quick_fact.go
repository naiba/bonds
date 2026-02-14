package models

import "time"

type VaultQuickFactsTemplate struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID             string    `json:"vault_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            int       `json:"position" gorm:"not null"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Vault Vault `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
}

type QuickFact struct {
	ID                        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultQuickFactsTemplateID uint      `json:"vault_quick_facts_template_id" gorm:"not null;index"`
	ContactID                 string    `json:"contact_id" gorm:"type:text;not null;index"`
	Content                   string    `json:"content" gorm:"not null"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`

	VaultQuickFactsTemplate VaultQuickFactsTemplate `json:"vault_quick_facts_template,omitempty" gorm:"foreignKey:VaultQuickFactsTemplateID"`
	Contact                 Contact                 `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
}
