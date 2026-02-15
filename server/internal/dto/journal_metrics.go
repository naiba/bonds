package dto

import "time"

type CreateJournalMetricRequest struct {
	Label string `json:"label" validate:"required" example:"Happiness"`
}

type JournalMetricResponse struct {
	ID        uint      `json:"id" example:"1"`
	JournalID uint      `json:"journal_id" example:"1"`
	Label     string    `json:"label" example:"Happiness"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
