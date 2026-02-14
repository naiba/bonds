package models

import "time"

type GiftOccasion struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID           string    `json:"account_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            *int      `json:"position"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

type GiftState struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID           string    `json:"account_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            *int      `json:"position"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Account Account `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

type Gift struct {
	ID             uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID      string     `json:"contact_id" gorm:"type:text;not null;index"`
	Type           string     `json:"type" gorm:"not null"`
	Name           string     `json:"name" gorm:"not null"`
	Description    *string    `json:"description" gorm:"type:text"`
	EstimatedPrice *int       `json:"estimated_price"`
	CurrencyID     *uint      `json:"currency_id" gorm:"index"`
	ReceivedAt     *time.Time `json:"received_at"`
	GivenAt        *time.Time `json:"given_at"`
	BoughtAt       *time.Time `json:"bought_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Contact  Contact   `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Currency *Currency `json:"currency,omitempty" gorm:"foreignKey:CurrencyID"`
}

type ContactGift struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	LoanID    uint      `json:"loan_id" gorm:"not null;index"`
	LoanerID  string    `json:"loaner_id" gorm:"type:text;not null;index"`
	LoaneeID  string    `json:"loanee_id" gorm:"type:text;not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ContactGift) TableName() string {
	return "contact_gift"
}
