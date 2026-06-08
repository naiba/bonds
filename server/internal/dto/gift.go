package dto

import "time"

type CreateGiftRequest struct {
	Name           string     `json:"name" validate:"required" example:"Birthday book"`
	Type           string     `json:"type" example:"given"`
	Description    string     `json:"description" example:"Signed first edition"`
	EstimatedPrice *int       `json:"estimated_price" example:"2500"`
	CurrencyID     *uint      `json:"currency_id" example:"1"`
	GiftOccasionID uint       `json:"gift_occasion_id" validate:"required" example:"1"`
	GiftStateID    uint       `json:"gift_state_id" validate:"required" example:"1"`
	StatusDate     *time.Time `json:"status_date" example:"2026-01-15T10:30:00Z"`
	ReceivedAt     *time.Time `json:"received_at" example:"2026-01-15T10:30:00Z"`
	GivenAt        *time.Time `json:"given_at" example:"2026-01-15T10:30:00Z"`
	BoughtAt       *time.Time `json:"bought_at" example:"2026-01-15T10:30:00Z"`
}

type UpdateGiftRequest struct {
	Name           string     `json:"name" validate:"required" example:"Birthday book"`
	Type           string     `json:"type" example:"given"`
	Description    string     `json:"description" example:"Signed first edition"`
	EstimatedPrice *int       `json:"estimated_price" example:"2500"`
	CurrencyID     *uint      `json:"currency_id" example:"1"`
	GiftOccasionID uint       `json:"gift_occasion_id" validate:"required" example:"1"`
	GiftStateID    uint       `json:"gift_state_id" validate:"required" example:"1"`
	StatusDate     *time.Time `json:"status_date" example:"2026-01-15T10:30:00Z"`
	ReceivedAt     *time.Time `json:"received_at" example:"2026-01-15T10:30:00Z"`
	GivenAt        *time.Time `json:"given_at" example:"2026-01-15T10:30:00Z"`
	BoughtAt       *time.Time `json:"bought_at" example:"2026-01-15T10:30:00Z"`
}

type GiftResponse struct {
	ID                uint       `json:"id" example:"1"`
	ContactID         string     `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name              string     `json:"name" example:"Birthday book"`
	Type              string     `json:"type" example:"given"`
	Description       string     `json:"description" example:"Signed first edition"`
	EstimatedPrice    *int       `json:"estimated_price" example:"2500"`
	CurrencyID        *uint      `json:"currency_id" example:"1"`
	GiftOccasionID    *uint      `json:"gift_occasion_id" example:"1"`
	GiftOccasionLabel string     `json:"gift_occasion_label" example:"Birthday"`
	GiftStateID       *uint      `json:"gift_state_id" example:"1"`
	GiftStateLabel    string     `json:"gift_state_label" example:"Idea"`
	StatusDate        *time.Time `json:"status_date" example:"2026-01-15T10:30:00Z"`
	ReceivedAt        *time.Time `json:"received_at" example:"2026-01-15T10:30:00Z"`
	GivenAt           *time.Time `json:"given_at" example:"2026-01-15T10:30:00Z"`
	BoughtAt          *time.Time `json:"bought_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt         time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
