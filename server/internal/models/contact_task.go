package models

import (
	"time"

	"gorm.io/gorm"
)

// Task status constants — used for kanban-style boards.
// New rows default to TaskStatusTodo via gorm:"default:'todo'".
// Existing pre-migration rows have an empty string and are treated as TaskStatusTodo on read.
const (
	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusDone       = "done"
	TaskStatusBlocked    = "blocked"
	TaskStatusCancelled  = "cancelled"
)

// IsValidTaskStatus returns true if the given status is a recognized task status.
func IsValidTaskStatus(s string) bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusBlocked, TaskStatusCancelled:
		return true
	}
	return false
}

// NormalizeTaskStatus returns the given status, or TaskStatusTodo if empty/invalid.
// Used as a read-side fallback for existing rows that pre-date the Status field.
func NormalizeTaskStatus(s string) string {
	if IsValidTaskStatus(s) {
		return s
	}
	return TaskStatusTodo
}

// ContactTask represents a vault-scoped task. It can be assigned to any number
// of contacts via the task_contacts pivot (zero contacts = standalone task)
// and can have one parent task (ParentTaskID) to form a sub-task hierarchy.
type ContactTask struct {
	ID uint `json:"id" gorm:"primaryKey;autoIncrement"`
	// VaultID — required. Tasks always belong to a vault.
	VaultID string `json:"vault_id" gorm:"type:text;not null;index"`
	// ParentTaskID — optional. Non-NULL means this row is a sub-task of the
	// referenced task. Sub-tasks live in the same vault as their parent.
	ParentTaskID *uint          `json:"parent_task_id" gorm:"index"`
	AuthorID     *string        `json:"author_id" gorm:"type:text;index"`
	UUID         *string        `json:"uuid" gorm:"type:text;index"`
	Vcalendar    *string        `json:"vcalendar" gorm:"type:text"`
	DistantUUID  *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag  *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI   *string        `json:"distant_uri" gorm:"size:2096"`
	AuthorName   string         `json:"author_name" gorm:"not null"`
	Label        string         `json:"label" gorm:"not null"`
	Description  *string        `json:"description" gorm:"type:text"`
	Status       string         `json:"status" gorm:"type:text;default:'todo';index"`
	Position     int            `json:"position" gorm:"default:0"`
	Completed    bool           `json:"completed" gorm:"default:false"`
	CompletedAt  *time.Time     `json:"completed_at"`
	DueAt        *time.Time     `json:"due_at"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear preserve the
	// user's input when they set a due date in a non-Gregorian calendar
	// (e.g. lunar). DueAt always stores the Gregorian projection so existing
	// sort/filter on the kanban keep working unchanged; the Original* triple
	// lets the UI render the lunar label and lets edits re-display the user's
	// original input instead of a back-converted value that may have drifted
	// by a day.
	// Defaults to "gregorian" so legacy rows pre-dating the column read as
	// gregorian without a backfill needing to touch them on every boot.
	CalendarType  string `json:"calendar_type" gorm:"default:'gregorian'"`
	OriginalDay   *int   `json:"original_day"`
	OriginalMonth *int   `json:"original_month"`
	OriginalYear  *int   `json:"original_year"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	Author     *User         `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	ParentTask *ContactTask  `json:"parent_task,omitempty" gorm:"foreignKey:ParentTaskID"`
	SubTasks   []ContactTask `json:"sub_tasks,omitempty" gorm:"foreignKey:ParentTaskID"`
	Contacts   []Contact     `json:"contacts,omitempty" gorm:"many2many:task_contacts;joinForeignKey:ContactTaskID;joinReferences:ContactID"`
}

// TaskContact is the explicit pivot for the ContactTask <-> Contact m2m. Kept
// as a real struct (rather than the GORM-auto-generated join) so timestamps
// are tracked and per-assignment metadata can be added later (assigned_at,
// role, etc.) without another migration.
type TaskContact struct {
	ContactTaskID uint      `json:"contact_task_id" gorm:"primaryKey"`
	ContactID     string    `json:"contact_id" gorm:"type:text;primaryKey"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (TaskContact) TableName() string {
	return "task_contacts"
}
