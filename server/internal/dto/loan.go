package dto

import "time"

type CreateLoanRequest struct {
	Name        string     `json:"name" validate:"required" example:"Laptop loan"`
	Type        string     `json:"type" validate:"required" example:"debt"`
	Description string     `json:"description" example:"Borrowed money for new laptop"`
	AmountLent  *int       `json:"amount_lent" example:"500"`
	CurrencyID  *uint      `json:"currency_id" example:"1"`
	LoanedAt    *time.Time `json:"loaned_at" example:"2026-01-15T10:30:00Z"`
}

type UpdateLoanRequest struct {
	Name        string     `json:"name" validate:"required" example:"Laptop loan"`
	Type        string     `json:"type" validate:"required" example:"debt"`
	Description string     `json:"description" example:"Borrowed money for new laptop"`
	AmountLent  *int       `json:"amount_lent" example:"500"`
	CurrencyID  *uint      `json:"currency_id" example:"1"`
	LoanedAt    *time.Time `json:"loaned_at" example:"2026-01-15T10:30:00Z"`
}

type LoanResponse struct {
	ID          uint       `json:"id" example:"1"`
	VaultID     string     `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string     `json:"name" example:"Laptop loan"`
	Type        string     `json:"type" example:"debt"`
	Description string     `json:"description" example:"Borrowed money for new laptop"`
	AmountLent  *int       `json:"amount_lent" example:"500"`
	CurrencyID  *uint      `json:"currency_id" example:"1"`
	LoanedAt    *time.Time `json:"loaned_at" example:"2026-01-15T10:30:00Z"`
	Settled     bool       `json:"settled" example:"false"`
	SettledAt   *time.Time `json:"settled_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
