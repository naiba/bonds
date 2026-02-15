package dto

import "time"

type CreateJournalMetricRequest struct {
	Label string `json:"label" validate:"required"`
}

type JournalMetricResponse struct {
	ID        uint      `json:"id"`
	JournalID uint      `json:"journal_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
