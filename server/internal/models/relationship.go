package models

import "time"

type RelationshipGroupType struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID          string    `json:"account_id" gorm:"type:text;not null;index"`
	Name               *string   `json:"name"`
	NameTranslationKey *string   `json:"name_translation_key"`
	Type               *string   `json:"type"`
	CanBeDeleted       bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Account Account            `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	Types   []RelationshipType `json:"types,omitempty" gorm:"foreignKey:RelationshipGroupTypeID"`
}

type RelationshipType struct {
	ID                                    uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	RelationshipGroupTypeID               uint      `json:"relationship_group_type_id" gorm:"not null;index"`
	Name                                  *string   `json:"name"`
	NameTranslationKey                    *string   `json:"name_translation_key"`
	NameReverseRelationship               *string   `json:"name_reverse_relationship"`
	NameReverseRelationshipTranslationKey *string   `json:"name_reverse_relationship_translation_key"`
	Type                                  *string   `json:"type"`
	CanBeDeleted                          bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt                             time.Time `json:"created_at"`
	UpdatedAt                             time.Time `json:"updated_at"`

	GroupType RelationshipGroupType `json:"group_type,omitempty" gorm:"foreignKey:RelationshipGroupTypeID"`
}

type Relationship struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	RelationshipTypeID uint      `json:"relationship_type_id" gorm:"not null;index"`
	ContactID          string    `json:"contact_id" gorm:"type:text;not null;index"`
	RelatedContactID   string    `json:"related_contact_id" gorm:"type:text;not null;index"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	RelationshipType RelationshipType `json:"relationship_type,omitempty" gorm:"foreignKey:RelationshipTypeID"`
	Contact          Contact          `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	RelatedContact   Contact          `json:"related_contact,omitempty" gorm:"foreignKey:RelatedContactID"`
}
