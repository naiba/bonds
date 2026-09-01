package dto

import "time"

// ActivityUpsertRequest is the single write contract for point-in-time events,
// ongoing experiences, completed periods, and milestones. PrimaryContactID is
// set by contact-detail flows; ParticipantIDs contains optional ordinary-contact
// participants selected by the user.
type ActivityUpsertRequest struct {
	PrimaryContactID  string     `json:"primary_contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ParticipantIDs    *[]string  `json:"participant_ids" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	ParentID          *uint      `json:"parent_id" example:"42"`
	ActivityTypeID    uint       `json:"activity_type_id" validate:"required" example:"1"`
	Title             string     `json:"title" validate:"required" example:"Product designer at Acme"`
	Description       string     `json:"description" example:"Worked with @[Alice](contact:550e8400-e29b-41d4-a716-446655440000)"`
	DescriptionFormat string     `json:"description_format" example:"markdown" enums:"plain,markdown"`
	StartDate         *time.Time `json:"start_date" example:"2026-01-15T00:00:00Z"`
	StartPrecision    string     `json:"start_precision" example:"day"`
	EndDate           *time.Time `json:"end_date" example:"2026-08-31T00:00:00Z"`
	EndPrecision      string     `json:"end_precision" example:"month"`
	EndStatus         string     `json:"end_status" example:"known"`

	CalendarType  string `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int   `json:"original_day" example:"15"`
	OriginalMonth *int   `json:"original_month" example:"6"`
	OriginalYear  *int   `json:"original_year" example:"1990"`

	EmotionID         *uint  `json:"emotion_id" example:"1"`
	Costs             *int   `json:"costs" example:"50"`
	CurrencyID        *uint  `json:"currency_id" example:"1"`
	DurationInMinutes *int   `json:"duration_in_minutes" example:"30"`
	Distance          *int   `json:"distance" example:"100"`
	DistanceUnit      string `json:"distance_unit" example:"km"`
	FromPlace         string `json:"from_place" example:"San Francisco"`
	ToPlace           string `json:"to_place" example:"Los Angeles"`
	Place             string `json:"place" example:"Downtown Office"`
}

type ActivityResponse struct {
	ID                   uint                  `json:"id" example:"1"`
	VaultID              string                `json:"vault_id"`
	SubjectUserID        string                `json:"subject_user_id,omitempty"`
	SubjectUserName      string                `json:"subject_user_name,omitempty"`
	SubjectIsCurrentUser bool                  `json:"subject_is_current_user"`
	ParentID             *uint                 `json:"parent_id"`
	ActivityTypeID       *uint                 `json:"activity_type_id"`
	ActivityType         *ActivityTypeResponse `json:"activity_type,omitempty"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	DescriptionFormat    string                `json:"description_format"`
	RenderedDescription  string                `json:"rendered_description"`
	StartDate            *time.Time            `json:"start_date"`
	StartPrecision       string                `json:"start_precision"`
	EndDate              *time.Time            `json:"end_date"`
	EndPrecision         string                `json:"end_precision"`
	EndStatus            string                `json:"end_status"`
	CalendarType         string                `json:"calendar_type"`
	OriginalDay          *int                  `json:"original_day"`
	OriginalMonth        *int                  `json:"original_month"`
	OriginalYear         *int                  `json:"original_year"`
	EmotionID            *uint                 `json:"emotion_id"`
	Costs                *int                  `json:"costs"`
	CurrencyID           *uint                 `json:"currency_id"`
	DurationInMinutes    *int                  `json:"duration_in_minutes"`
	Distance             *int                  `json:"distance"`
	DistanceUnit         string                `json:"distance_unit"`
	FromPlace            string                `json:"from_place"`
	ToPlace              string                `json:"to_place"`
	Place                string                `json:"place"`
	Participants         []TaskContactRef      `json:"participants"`
	MentionedContacts    []TaskContactRef      `json:"mentioned_contacts"`
	Milestones           []ActivityResponse    `json:"milestones,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}
