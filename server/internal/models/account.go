package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ID               string    `json:"id" gorm:"primaryKey;type:text"`
	StorageLimitInMB int       `json:"storage_limit_in_mb" gorm:"default:0"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Users                   []User                   `json:"users,omitempty" gorm:"foreignKey:AccountID"`
	Vaults                  []Vault                  `json:"vaults,omitempty" gorm:"foreignKey:AccountID"`
	Templates               []Template               `json:"templates,omitempty" gorm:"foreignKey:AccountID"`
	Modules                 []Module                 `json:"modules,omitempty" gorm:"foreignKey:AccountID"`
	GroupTypes              []GroupType              `json:"group_types,omitempty" gorm:"foreignKey:AccountID"`
	RelationshipGroupTypes  []RelationshipGroupType  `json:"relationship_group_types,omitempty" gorm:"foreignKey:AccountID"`
	Genders                 []Gender                 `json:"genders,omitempty" gorm:"foreignKey:AccountID"`
	Pronouns                []Pronoun                `json:"pronouns,omitempty" gorm:"foreignKey:AccountID"`
	ContactInformationTypes []ContactInformationType `json:"contact_information_types,omitempty" gorm:"foreignKey:AccountID"`
	AddressTypes            []AddressType            `json:"address_types,omitempty" gorm:"foreignKey:AccountID"`
	PetCategories           []PetCategory            `json:"pet_categories,omitempty" gorm:"foreignKey:AccountID"`
	Emotions                []Emotion                `json:"emotions,omitempty" gorm:"foreignKey:AccountID"`
	Currencies              []Currency               `json:"currencies,omitempty" gorm:"many2many:account_currency"`
	CallReasonTypes         []CallReasonType         `json:"call_reason_types,omitempty" gorm:"foreignKey:AccountID"`
	GiftOccasions           []GiftOccasion           `json:"gift_occasions,omitempty" gorm:"foreignKey:AccountID"`
	GiftStates              []GiftState              `json:"gift_states,omitempty" gorm:"foreignKey:AccountID"`
	PostTemplates           []PostTemplate           `json:"post_templates,omitempty" gorm:"foreignKey:AccountID"`
	Religions               []Religion               `json:"religions,omitempty" gorm:"foreignKey:AccountID"`
}

func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
