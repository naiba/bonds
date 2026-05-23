package dto

import "time"

type VaultTaskResponse struct {
	ID           uint             `json:"id" example:"1"`
	VaultID      string           `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorName   string           `json:"author_name" example:"John Doe"`
	Label        string           `json:"label" example:"Buy birthday gift"`
	Description  string           `json:"description" example:"Get a nice book from the bookstore"`
	Status       string           `json:"status" example:"todo"`
	Position     int              `json:"position" example:"0"`
	Completed    bool             `json:"completed" example:"false"`
	CompletedAt  *time.Time       `json:"completed_at" example:"2026-01-15T10:30:00Z"`
	DueAt        *time.Time       `json:"due_at" example:"2026-01-15T10:30:00Z"`
	ParentTaskID *uint            `json:"parent_task_id" example:"42"`
	Contacts     []TaskContactRef `json:"contacts"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear preserve the
	// alternative-calendar anchor (lunar etc.) that produced DueAt. Stored
	// alongside DueAt so the frontend can re-render the original date in the
	// user's preferred calendar even after gregorian projection.
	CalendarType  string    `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int      `json:"original_day" example:"15"`
	OriginalMonth *int      `json:"original_month" example:"8"`
	OriginalYear  *int      `json:"original_year" example:"2026"`
	CreatedAt     time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type CreateVaultTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	Status      string     `json:"status" example:"todo"`
	// ContactIDs — assignees. Empty slice or omitted = standalone task. All
	// IDs must belong to the same vault.
	ContactIDs []string `json:"contact_ids" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	// ParentTaskID — optional sub-task pointer. Parent must live in the same vault.
	ParentTaskID *uint `json:"parent_task_id" example:"42"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear let the UI
	// pick a lunar (or other) due date — the server projects it to a
	// gregorian DueAt and persists the alternative-calendar anchor so the
	// reminder scheduler can re-resolve future recurrences correctly.
	CalendarType  string `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int   `json:"original_day" example:"15"`
	OriginalMonth *int   `json:"original_month" example:"8"`
	OriginalYear  *int   `json:"original_year" example:"2026"`
}

// UpdateVaultTaskRequest replaces the editable fields of a vault task in one
// call. Status is validated against the known set; an empty Status means
// "leave unchanged". ContactIDs replaces the assignee set verbatim when
// provided; nil means "leave assignees untouched". ParentTaskID is tri-state
// (omitted = leave unchanged; null = clear; number = set), see NullableUint.
type UpdateVaultTaskRequest struct {
	Label        string       `json:"label" validate:"required" example:"Buy birthday gift"`
	Description  string       `json:"description" example:"Get a nice book from the bookstore"`
	DueAt        *time.Time   `json:"due_at" example:"2026-01-15T10:30:00Z"`
	Status       string       `json:"status" example:"in_progress"`
	ContactIDs   *[]string    `json:"contact_ids" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	ParentTaskID NullableUint `json:"parent_task_id,omitempty" swaggertype:"integer" example:"42"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear — see
	// CreateVaultTaskRequest. Sending an empty CalendarType (or "gregorian")
	// clears any prior lunar anchor; a nil DueAt also clears them because a
	// dateless task has no calendar semantics.
	CalendarType  string `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int   `json:"original_day" example:"15"`
	OriginalMonth *int   `json:"original_month" example:"8"`
	OriginalYear  *int   `json:"original_year" example:"2026"`
}
