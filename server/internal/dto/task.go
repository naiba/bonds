package dto

import "time"

// TaskContactRef is the lightweight contact view embedded in task responses
// so the UI can render every assignee without a follow-up request per contact.
type TaskContactRef struct {
	ID   string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name string `json:"name" example:"Jane Doe"`
}

type CreateTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear let the UI
	// record a lunar due date. The server overwrites DueAt with the Gregorian
	// projection computed from these fields, so the kanban sort order stays
	// consistent regardless of the picker mode used to enter the date.
	CalendarType  string `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int   `json:"original_day" example:"15"`
	OriginalMonth *int   `json:"original_month" example:"8"`
	OriginalYear  *int   `json:"original_year" example:"2026"`
	Status        string `json:"status" example:"todo"`
	// ContactIDs — additional assignees besides the contact in the URL. The
	// URL's contact is always included; extras here add a multi-person task.
	ContactIDs []string `json:"contact_ids" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	// ParentTaskID — optional. When set, this task is a sub-task of the
	// given parent task. Parent must live in the same vault.
	ParentTaskID *uint `json:"parent_task_id" example:"42"`
}

type UpdateTaskRequest struct {
	Label       string     `json:"label" validate:"required" example:"Buy birthday gift"`
	Description string     `json:"description" example:"Get a nice book from the bookstore"`
	DueAt       *time.Time `json:"due_at" example:"2026-01-15T10:30:00Z"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear — see
	// CreateTaskRequest. Sending an empty CalendarType (or "gregorian") clears
	// Original* and pins the row to gregorian.
	CalendarType  string `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int   `json:"original_day" example:"15"`
	OriginalMonth *int   `json:"original_month" example:"8"`
	OriginalYear  *int   `json:"original_year" example:"2026"`
	Status        string `json:"status" example:"todo"`
	// ContactIDs — when nil, assignees are left untouched. When provided
	// (even as an empty slice), the assignee set is replaced verbatim. The
	// contact in the URL path is always re-added to keep the task visible
	// from this contact's task list.
	ContactIDs *[]string `json:"contact_ids" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	// ParentTaskID — tri-state: omitted = leave unchanged, null = clear,
	// number = set. See NullableUint.
	ParentTaskID NullableUint `json:"parent_task_id,omitempty" swaggertype:"integer" example:"42"`
}

type UpdateTaskStatusRequest struct {
	Status string `json:"status" validate:"required" example:"in_progress"`
}

type UpdateTaskPositionRequest struct {
	Position int    `json:"position" example:"3"`
	Status   string `json:"status" example:"in_progress"`
}

type TaskResponse struct {
	ID            uint             `json:"id" example:"1"`
	VaultID       string           `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID      string           `json:"author_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label         string           `json:"label" example:"Buy birthday gift"`
	Description   string           `json:"description" example:"Get a nice book from the bookstore"`
	Status        string           `json:"status" example:"todo"`
	Position      int              `json:"position" example:"0"`
	Completed     bool             `json:"completed" example:"false"`
	CompletedAt   *time.Time       `json:"completed_at" example:"2026-01-15T10:30:00Z"`
	DueAt         *time.Time       `json:"due_at" example:"2026-01-15T10:30:00Z"`
	CalendarType  string           `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int             `json:"original_day" example:"15"`
	OriginalMonth *int             `json:"original_month" example:"8"`
	OriginalYear  *int             `json:"original_year" example:"2026"`
	ParentTaskID  *uint            `json:"parent_task_id" example:"42"`
	Contacts      []TaskContactRef `json:"contacts"`
	CreatedAt     time.Time        `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt     time.Time        `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
