package models

import (
	"time"

	"gorm.io/gorm"
)

type GroupType struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID           string    `json:"account_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            int       `json:"position" gorm:"not null"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Account        Account         `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	GroupTypeRoles []GroupTypeRole `json:"group_type_roles,omitempty" gorm:"foreignKey:GroupTypeID"`
}

type GroupTypeRole struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupTypeID         uint      `json:"group_type_id" gorm:"not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            *int      `json:"position"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	GroupType GroupType `json:"group_type,omitempty" gorm:"foreignKey:GroupTypeID"`
}

type Group struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	UUID        *string        `json:"uuid" gorm:"type:text"`
	VaultID     string         `json:"vault_id" gorm:"type:text;not null;index"`
	GroupTypeID *uint          `json:"group_type_id" gorm:"index"`
	Name        string         `json:"name" gorm:"not null"`
	Vcard       *string        `json:"vcard" gorm:"type:text"`
	DistantUUID *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI  *string        `json:"distant_uri" gorm:"size:2096"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	Vault     Vault      `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	GroupType *GroupType `json:"group_type,omitempty" gorm:"foreignKey:GroupTypeID"`
	Contacts  []Contact  `json:"contacts,omitempty" gorm:"many2many:contact_group"`
}

type ContactGroup struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID         uint      `json:"group_id" gorm:"not null;index"`
	ContactID       string    `json:"contact_id" gorm:"type:text;not null;index"`
	GroupTypeRoleID *uint     `json:"group_type_role_id" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ContactGroup) TableName() string {
	return "contact_group"
}
