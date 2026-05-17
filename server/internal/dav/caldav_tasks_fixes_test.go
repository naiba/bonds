package dav

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
)

func TestListStandaloneTaskInEmptyContactVault(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	uid := "standalone-task-uid"
	task := models.ContactTask{
		VaultID:    vaultID,
		AuthorID:   &userID,
		UUID:       &uid,
		Label:      "Vault-level chore",
		AuthorName: "Test",
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create standalone task: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListCalendarObjects(ctx, path, nil)
	if err != nil {
		t.Fatalf("ListCalendarObjects: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected 1 calendar object, got %d", len(objects))
	}
}

func TestDeleteCalendarObjectAlsoDeletesPivot(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCalDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Pivot", "Test")

	uid := "task-delete-uid"
	task := models.ContactTask{
		VaultID:    vaultID,
		AuthorID:   &userID,
		UUID:       &uid,
		Label:      "delete me",
		AuthorName: "Test",
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := db.Create(&models.TaskContact{ContactTaskID: task.ID, ContactID: contact.ID}).Error; err != nil {
		t.Fatalf("create pivot: %v", err)
	}

	path := "/dav/calendars/" + userID + "/" + vaultID + "/" + uid + ".ics"
	if err := backend.DeleteCalendarObject(ctx, path); err != nil {
		t.Fatalf("DeleteCalendarObject: %v", err)
	}

	var pivotCount int64
	db.Model(&models.TaskContact{}).Where("contact_task_id = ?", task.ID).Count(&pivotCount)
	if pivotCount != 0 {
		t.Errorf("expected pivot rows to be deleted, got %d", pivotCount)
	}
}


