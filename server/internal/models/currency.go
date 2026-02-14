package models

import "time"

type Currency struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Code      string    `json:"code" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Accounts []Account `json:"accounts,omitempty" gorm:"many2many:account_currency"`
}

type AccountCurrency struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CurrencyID uint      `json:"currency_id" gorm:"not null;index"`
	AccountID  string    `json:"account_id" gorm:"type:text;not null;index"`
	Active     bool      `json:"active" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (AccountCurrency) TableName() string {
	return "account_currency"
}
