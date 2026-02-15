package dto

import "time"

type CreatePostMetricRequest struct {
	JournalMetricID uint `json:"journal_metric_id" validate:"required" example:"1"`
	Value           int  `json:"value" validate:"required" example:"5"`
}

type PostMetricResponse struct {
	ID              uint      `json:"id" example:"1"`
	PostID          uint      `json:"post_id" example:"1"`
	JournalMetricID uint      `json:"journal_metric_id" example:"1"`
	Value           int       `json:"value" example:"5"`
	Label           string    `json:"label" example:"Happiness"`
	CreatedAt       time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
