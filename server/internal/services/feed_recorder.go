package services

import (
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// Feed action constants
const (
	ActionContactCreated    = "contact_created"
	ActionContactUpdated    = "contact_updated"
	ActionContactDeleted    = "contact_deleted"
	ActionNoteCreated       = "note_created"
	ActionNoteUpdated       = "note_updated"
	ActionNoteDeleted       = "note_deleted"
	ActionReminderCreated   = "reminder_created"
	ActionCallLogged        = "call_logged"
	ActionTaskCreated       = "task_created"
	ActionTaskCompleted     = "task_completed"
	ActionAddressAdded      = "address_added"
	ActionLifeEventCreated  = "life_event_created"
	ActionFileUploaded      = "file_uploaded"
	ActionLoanCreated       = "loan_created"
	ActionRelationshipAdded = "relationship_added"
)

type FeedRecorder struct {
	db *gorm.DB
}

func NewFeedRecorder(db *gorm.DB) *FeedRecorder {
	return &FeedRecorder{db: db}
}

// Record creates a ContactFeedItem. feedableID/feedableType are optional (for polymorphic reference).
func (r *FeedRecorder) Record(contactID, authorID, action string, description string, feedableID *uint, feedableType *string) error {
	item := models.ContactFeedItem{
		ContactID:    contactID,
		Action:       action,
		FeedableID:   feedableID,
		FeedableType: feedableType,
	}
	if authorID != "" {
		item.AuthorID = &authorID
	}
	if description != "" {
		item.Description = &description
	}
	return r.db.Create(&item).Error
}
