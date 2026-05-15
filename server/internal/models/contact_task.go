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

type ContactTask struct {
	ID uint `json:"id" gorm:"primaryKey;autoIncrement"`
	// VaultID — required. Tasks always belong to a vault, even when not tied to a contact.
	VaultID string `json:"vault_id" gorm:"type:text;not null;index"`
	// ContactID — optional. NULL means a standalone vault-level task.
	ContactID   *string        `json:"contact_id" gorm:"type:text;index"`
	AuthorID    *string        `json:"author_id" gorm:"type:text;index"`
	UUID        *string        `json:"uuid" gorm:"type:text;index"`
	Vcalendar   *string        `json:"vcalendar" gorm:"type:text"`
	DistantUUID *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI  *string        `json:"distant_uri" gorm:"size:2096"`
	AuthorName  string         `json:"author_name" gorm:"not null"`
	Label       string         `json:"label" gorm:"not null"`
	Description *string        `json:"description" gorm:"type:text"`
	Status      string         `json:"status" gorm:"type:text;default:'todo';index"`
	Position    int            `json:"position" gorm:"default:0"`
	Completed   bool           `json:"completed" gorm:"default:false"`
	CompletedAt *time.Time     `json:"completed_at"`
	DueAt       *time.Time     `json:"due_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	Contact *Contact `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Author  *User    `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
}
