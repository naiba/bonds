package dto

import "time"

type CreateMoodTrackingEventRequest struct {
	MoodTrackingParameterID uint      `json:"mood_tracking_parameter_id" validate:"required" example:"1"`
	RatedAt                 time.Time `json:"rated_at" validate:"required" example:"2026-01-15T10:30:00Z"`
	Note                    string    `json:"note" example:"Feeling great after morning run"`
	NumberOfHoursSlept      *int      `json:"number_of_hours_slept" example:"8"`
}

type MoodTrackingEventResponse struct {
	ID                      uint      `json:"id" example:"1"`
	ContactID               string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MoodTrackingParameterID uint      `json:"mood_tracking_parameter_id" example:"1"`
	RatedAt                 time.Time `json:"rated_at" example:"2026-01-15T10:30:00Z"`
	Note                    string    `json:"note" example:"Feeling great after morning run"`
	NumberOfHoursSlept      *int      `json:"number_of_hours_slept" example:"8"`
	CreatedAt               time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt               time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
