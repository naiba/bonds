package dto

import "time"

type CreateReminderRequest struct {
	Label           string `json:"label" validate:"required"`
	Day             *int   `json:"day"`
	Month           *int   `json:"month"`
	Year            *int   `json:"year"`
	CalendarType    string `json:"calendar_type"`
	OriginalDay     *int   `json:"original_day"`
	OriginalMonth   *int   `json:"original_month"`
	OriginalYear    *int   `json:"original_year"`
	Type            string `json:"type" validate:"required"`
	FrequencyNumber *int   `json:"frequency_number"`
}

type UpdateReminderRequest struct {
	Label           string `json:"label" validate:"required"`
	Day             *int   `json:"day"`
	Month           *int   `json:"month"`
	Year            *int   `json:"year"`
	CalendarType    string `json:"calendar_type"`
	OriginalDay     *int   `json:"original_day"`
	OriginalMonth   *int   `json:"original_month"`
	OriginalYear    *int   `json:"original_year"`
	Type            string `json:"type" validate:"required"`
	FrequencyNumber *int   `json:"frequency_number"`
}

type ReminderResponse struct {
	ID                   uint       `json:"id"`
	ContactID            string     `json:"contact_id"`
	Label                string     `json:"label"`
	Day                  *int       `json:"day"`
	Month                *int       `json:"month"`
	Year                 *int       `json:"year"`
	CalendarType         string     `json:"calendar_type"`
	OriginalDay          *int       `json:"original_day"`
	OriginalMonth        *int       `json:"original_month"`
	OriginalYear         *int       `json:"original_year"`
	Type                 string     `json:"type"`
	FrequencyNumber      *int       `json:"frequency_number"`
	LastTriggeredAt      *time.Time `json:"last_triggered_at"`
	NumberTimesTriggered int        `json:"number_times_triggered"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
