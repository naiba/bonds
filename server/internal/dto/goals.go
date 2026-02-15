package dto

import "time"

type CreateGoalRequest struct {
	Name string `json:"name" validate:"required" example:"Exercise daily"`
}

type UpdateGoalRequest struct {
	Name   string `json:"name" validate:"required" example:"Exercise daily"`
	Active *bool  `json:"active" example:"true"`
}

type AddStreakRequest struct {
	HappenedAt time.Time `json:"happened_at" validate:"required" example:"2026-01-15T10:30:00Z"`
}

type GoalResponse struct {
	ID        uint             `json:"id" example:"1"`
	ContactID string           `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string           `json:"name" example:"Exercise daily"`
	Active    bool             `json:"active" example:"true"`
	Streaks   []StreakResponse `json:"streaks,omitempty"`
	CreatedAt time.Time        `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time        `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type StreakResponse struct {
	ID         uint      `json:"id" example:"1"`
	GoalID     uint      `json:"goal_id" example:"1"`
	HappenedAt time.Time `json:"happened_at" example:"2026-01-15T10:30:00Z"`
	CreatedAt  time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
