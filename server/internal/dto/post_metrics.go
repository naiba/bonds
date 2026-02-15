package dto

import "time"

type CreatePostMetricRequest struct {
	JournalMetricID uint `json:"journal_metric_id" validate:"required"`
	Value           int  `json:"value" validate:"required"`
}

type PostMetricResponse struct {
	ID              uint      `json:"id"`
	PostID          uint      `json:"post_id"`
	JournalMetricID uint      `json:"journal_metric_id"`
	Value           int       `json:"value"`
	Label           string    `json:"label"`
	CreatedAt       time.Time `json:"created_at"`
}
