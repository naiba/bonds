package dto

import "time"

type CreateTimelineEventRequest struct {
	StartedAt    time.Time `json:"started_at" validate:"required"`
	Label        string    `json:"label"`
	Participants []string  `json:"participants"`
}

type CreateLifeEventRequest struct {
	LifeEventTypeID   uint      `json:"life_event_type_id" validate:"required"`
	HappenedAt        time.Time `json:"happened_at" validate:"required"`
	Summary           string    `json:"summary"`
	Description       string    `json:"description"`
	Costs             *int      `json:"costs"`
	CurrencyID        *uint     `json:"currency_id"`
	DurationInMinutes *int      `json:"duration_in_minutes"`
	Distance          *int      `json:"distance"`
	DistanceUnit      string    `json:"distance_unit"`
	FromPlace         string    `json:"from_place"`
	ToPlace           string    `json:"to_place"`
	Place             string    `json:"place"`
}

type UpdateLifeEventRequest struct {
	LifeEventTypeID   uint      `json:"life_event_type_id"`
	HappenedAt        time.Time `json:"happened_at"`
	Summary           string    `json:"summary"`
	Description       string    `json:"description"`
	Costs             *int      `json:"costs"`
	CurrencyID        *uint     `json:"currency_id"`
	DurationInMinutes *int      `json:"duration_in_minutes"`
	Distance          *int      `json:"distance"`
	DistanceUnit      string    `json:"distance_unit"`
	FromPlace         string    `json:"from_place"`
	ToPlace           string    `json:"to_place"`
	Place             string    `json:"place"`
}

type TimelineEventResponse struct {
	ID         uint                `json:"id"`
	VaultID    string              `json:"vault_id"`
	StartedAt  time.Time           `json:"started_at"`
	Label      string              `json:"label"`
	Collapsed  bool                `json:"collapsed"`
	LifeEvents []LifeEventResponse `json:"life_events,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type LifeEventResponse struct {
	ID                uint      `json:"id"`
	TimelineEventID   uint      `json:"timeline_event_id"`
	LifeEventTypeID   uint      `json:"life_event_type_id"`
	HappenedAt        time.Time `json:"happened_at"`
	Summary           string    `json:"summary"`
	Description       string    `json:"description"`
	Costs             *int      `json:"costs"`
	CurrencyID        *uint     `json:"currency_id"`
	DurationInMinutes *int      `json:"duration_in_minutes"`
	Distance          *int      `json:"distance"`
	DistanceUnit      string    `json:"distance_unit"`
	FromPlace         string    `json:"from_place"`
	ToPlace           string    `json:"to_place"`
	Place             string    `json:"place"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
