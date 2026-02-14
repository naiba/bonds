package dto

import "time"

type CreateGoalRequest struct {
	Name string `json:"name" validate:"required"`
}

type UpdateGoalRequest struct {
	Name   string `json:"name" validate:"required"`
	Active *bool  `json:"active"`
}

type AddStreakRequest struct {
	HappenedAt time.Time `json:"happened_at" validate:"required"`
}

type GoalResponse struct {
	ID        uint             `json:"id"`
	ContactID string           `json:"contact_id"`
	Name      string           `json:"name"`
	Active    bool             `json:"active"`
	Streaks   []StreakResponse `json:"streaks,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type StreakResponse struct {
	ID         uint      `json:"id"`
	GoalID     uint      `json:"goal_id"`
	HappenedAt time.Time `json:"happened_at"`
	CreatedAt  time.Time `json:"created_at"`
}
