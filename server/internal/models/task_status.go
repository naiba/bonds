package models

import "time"

// TaskStatus is an account-scoped, user-configurable status that ContactTask
// rows reference via the Status string column (matched on Slug). The set is
// seeded with 5 defaults at account creation; users can add more or remove
// non-core ones via the Personalize page. Position controls kanban column
// order. is_default marks the status assigned to a task when none is given.
type TaskStatus struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID          string    `json:"account_id" gorm:"type:text;not null;index"`
	Name               *string   `json:"name"`
	NameTranslationKey *string   `json:"name_translation_key"`
	// Slug — stable identifier used as the value of ContactTask.Status.
	// Auto-derived from Name on user-create; pre-set on seed for the cores.
	Slug         string    `json:"slug" gorm:"type:text;not null;index"`
	Position     int       `json:"position" gorm:"default:0"`
	IsDefault    bool      `json:"is_default" gorm:"default:false"`
	CanBeDeleted bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}
