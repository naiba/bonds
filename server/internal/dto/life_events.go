package dto

import "time"

type CreateTimelineEventRequest struct {
	StartedAt    time.Time `json:"started_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Label        string    `json:"label" example:"Summer Vacation 2026"`
	Participants []string  `json:"participants" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type CreateLifeEventRequest struct {
	LifeEventTypeID   uint      `json:"life_event_type_id" validate:"required" example:"1"`
	HappenedAt        time.Time `json:"happened_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Summary           string    `json:"summary" example:"Got promoted at work"`
	Description       string    `json:"description" example:"Received a promotion to senior engineer"`
	Costs             *int      `json:"costs" example:"50"`
	CurrencyID        *uint     `json:"currency_id" example:"1"`
	DurationInMinutes *int      `json:"duration_in_minutes" example:"30"`
	Distance          *int      `json:"distance" example:"100"`
	DistanceUnit      string    `json:"distance_unit" example:"km"`
	FromPlace         string    `json:"from_place" example:"San Francisco"`
	ToPlace           string    `json:"to_place" example:"Los Angeles"`
	Place             string    `json:"place" example:"Downtown Office"`
}

type UpdateLifeEventRequest struct {
	LifeEventTypeID   uint      `json:"life_event_type_id" example:"1"`
	HappenedAt        time.Time `json:"happened_at" example:"2026-01-15T10:30:00Z"`
	Summary           string    `json:"summary" example:"Got promoted at work"`
	Description       string    `json:"description" example:"Received a promotion to senior engineer"`
	Costs             *int      `json:"costs" example:"50"`
	CurrencyID        *uint     `json:"currency_id" example:"1"`
	DurationInMinutes *int      `json:"duration_in_minutes" example:"30"`
	Distance          *int      `json:"distance" example:"100"`
	DistanceUnit      string    `json:"distance_unit" example:"km"`
	FromPlace         string    `json:"from_place" example:"San Francisco"`
	ToPlace           string    `json:"to_place" example:"Los Angeles"`
	Place             string    `json:"place" example:"Downtown Office"`
}

type TimelineEventResponse struct {
	ID         uint                `json:"id" example:"1"`
	VaultID    string              `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartedAt  time.Time           `json:"started_at" example:"2026-01-15T10:30:00Z"`
	Label      string              `json:"label" example:"Summer Vacation 2026"`
	Collapsed  bool                `json:"collapsed" example:"false"`
	LifeEvents []LifeEventResponse `json:"life_events,omitempty"`
	CreatedAt  time.Time           `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt  time.Time           `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type LifeEventResponse struct {
	ID                uint      `json:"id" example:"1"`
	TimelineEventID   uint      `json:"timeline_event_id" example:"1"`
	LifeEventTypeID   uint      `json:"life_event_type_id" example:"1"`
	HappenedAt        time.Time `json:"happened_at" example:"2026-01-15T10:30:00Z"`
	Collapsed         bool      `json:"collapsed" example:"false"`
	Summary           string    `json:"summary" example:"Got promoted at work"`
	Description       string    `json:"description" example:"Received a promotion to senior engineer"`
	Costs             *int      `json:"costs" example:"50"`
	CurrencyID        *uint     `json:"currency_id" example:"1"`
	DurationInMinutes *int      `json:"duration_in_minutes" example:"30"`
	Distance          *int      `json:"distance" example:"100"`
	DistanceUnit      string    `json:"distance_unit" example:"km"`
	FromPlace         string    `json:"from_place" example:"San Francisco"`
	ToPlace           string    `json:"to_place" example:"Los Angeles"`
	Place             string    `json:"place" example:"Downtown Office"`
	CreatedAt         time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt         time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
