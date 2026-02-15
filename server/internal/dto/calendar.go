package dto

import "time"

type CalendarResponse struct {
	ImportantDates []CalendarDateItem     `json:"important_dates"`
	Reminders      []CalendarReminderItem `json:"reminders"`
}

type CalendarDateItem struct {
	ID            uint   `json:"id"`
	ContactID     string `json:"contact_id"`
	Label         string `json:"label"`
	Day           *int   `json:"day"`
	Month         *int   `json:"month"`
	Year          *int   `json:"year"`
	CalendarType  string `json:"calendar_type"`
	OriginalDay   *int   `json:"original_day"`
	OriginalMonth *int   `json:"original_month"`
	OriginalYear  *int   `json:"original_year"`
}

type CalendarReminderItem struct {
	ID            uint      `json:"id"`
	ContactID     string    `json:"contact_id"`
	Label         string    `json:"label"`
	Day           *int      `json:"day"`
	Month         *int      `json:"month"`
	Year          *int      `json:"year"`
	CalendarType  string    `json:"calendar_type"`
	OriginalDay   *int      `json:"original_day"`
	OriginalMonth *int      `json:"original_month"`
	OriginalYear  *int      `json:"original_year"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created_at"`
}
