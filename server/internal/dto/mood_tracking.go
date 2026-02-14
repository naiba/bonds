package dto

import "time"

type CreateMoodTrackingEventRequest struct {
	MoodTrackingParameterID uint      `json:"mood_tracking_parameter_id" validate:"required"`
	RatedAt                 time.Time `json:"rated_at" validate:"required"`
	Note                    string    `json:"note"`
	NumberOfHoursSlept      *int      `json:"number_of_hours_slept"`
}

type MoodTrackingEventResponse struct {
	ID                      uint      `json:"id"`
	ContactID               string    `json:"contact_id"`
	MoodTrackingParameterID uint      `json:"mood_tracking_parameter_id"`
	RatedAt                 time.Time `json:"rated_at"`
	Note                    string    `json:"note"`
	NumberOfHoursSlept      *int      `json:"number_of_hours_slept"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}
