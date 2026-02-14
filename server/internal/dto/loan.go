package dto

import "time"

type CreateLoanRequest struct {
	Name        string     `json:"name" validate:"required"`
	Type        string     `json:"type" validate:"required"`
	Description string     `json:"description"`
	AmountLent  *int       `json:"amount_lent"`
	CurrencyID  *uint      `json:"currency_id"`
	LoanedAt    *time.Time `json:"loaned_at"`
}

type UpdateLoanRequest struct {
	Name        string     `json:"name" validate:"required"`
	Type        string     `json:"type" validate:"required"`
	Description string     `json:"description"`
	AmountLent  *int       `json:"amount_lent"`
	CurrencyID  *uint      `json:"currency_id"`
	LoanedAt    *time.Time `json:"loaned_at"`
}

type LoanResponse struct {
	ID          uint       `json:"id"`
	VaultID     string     `json:"vault_id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	AmountLent  *int       `json:"amount_lent"`
	CurrencyID  *uint      `json:"currency_id"`
	LoanedAt    *time.Time `json:"loaned_at"`
	Settled     bool       `json:"settled"`
	SettledAt   *time.Time `json:"settled_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
