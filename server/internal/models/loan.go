package models

import "time"

type Loan struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID     string     `json:"vault_id" gorm:"type:text;not null;index"`
	Type        string     `json:"type" gorm:"not null"`
	Name        string     `json:"name" gorm:"not null"`
	Description *string    `json:"description" gorm:"type:text"`
	AmountLent  *int       `json:"amount_lent"`
	CurrencyID  *uint      `json:"currency_id" gorm:"index"`
	LoanedAt    *time.Time `json:"loaned_at"`
	Settled     bool       `json:"settled" gorm:"default:false"`
	SettledAt   *time.Time `json:"settled_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Vault    Vault     `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Currency *Currency `json:"currency,omitempty" gorm:"foreignKey:CurrencyID"`
	Loaners  []Contact `json:"loaners,omitempty" gorm:"many2many:contact_loan;joinForeignKey:LoanID;joinReferences:LoanerID"`
	Loanees  []Contact `json:"loanees,omitempty" gorm:"many2many:contact_loan;joinForeignKey:LoanID;joinReferences:LoaneeID"`
}

type ContactLoan struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	LoanID    uint      `json:"loan_id" gorm:"not null;index"`
	LoanerID  string    `json:"loaner_id" gorm:"type:text;not null;index"`
	LoaneeID  string    `json:"loanee_id" gorm:"type:text;not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ContactLoan) TableName() string {
	return "contact_loan"
}
