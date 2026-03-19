package dto

import "time"

type CreateLifeMetricRequest struct {
	Label string `json:"label" validate:"required" example:"Body Weight"`
}

type UpdateLifeMetricRequest struct {
	Label string `json:"label" validate:"required" example:"Body Weight"`
}

// LifeMetricStats holds event counts for a life metric scoped to the current user.
type LifeMetricStats struct {
	WeeklyEvents  int `json:"weekly_events" example:"3"`
	MonthlyEvents int `json:"monthly_events" example:"12"`
	YearlyEvents  int `json:"yearly_events" example:"52"`
}

type LifeMetricResponse struct {
	ID        uint            `json:"id" example:"1"`
	VaultID   string          `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label     string          `json:"label" example:"Body Weight"`
	Stats     LifeMetricStats `json:"stats"`
	CreatedAt time.Time       `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time       `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

// LifeMetricIncrementResponse wraps a full LifeMetricResponse with recalculated stats after an increment.
type LifeMetricIncrementResponse = LifeMetricResponse

// LifeMetricMonthData holds event count for one calendar month.
type LifeMetricMonthData struct {
	Month        int    `json:"month" example:"1"`
	FriendlyName string `json:"friendly_name" example:"January"`
	Events       int    `json:"events" example:"5"`
}

// LifeMetricDetailResponse extends LifeMetricResponse with a 12-month breakdown for a given year.
type LifeMetricDetailResponse struct {
	ID        uint                  `json:"id" example:"1"`
	VaultID   string                `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label     string                `json:"label" example:"Body Weight"`
	Stats     LifeMetricStats       `json:"stats"`
	Months    []LifeMetricMonthData `json:"months"`
	MaxEvents int                   `json:"max_events" example:"10"`
	CreatedAt time.Time             `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time             `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
