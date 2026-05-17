package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestContactScopedDeleteCascadesSubTasks(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	parent, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "P"})
	child, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:        "C",
		ParentTaskID: &parent.ID,
	})

	if err := svc.Delete(parent.ID, contactID, vaultID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	for _, id := range []uint{parent.ID, child.ID} {
		var task models.ContactTask
		if err := svc.db.First(&task, id).Error; err == nil {
			t.Errorf("task %d should be soft-deleted, still visible to live queries", id)
		}
		var pivotCount int64
		svc.db.Model(&models.TaskContact{}).Where("contact_task_id = ?", id).Count(&pivotCount)
		if pivotCount != 0 {
			t.Errorf("task %d still has %d pivot rows after delete", id, pivotCount)
		}
	}
}

func TestContactScopedRejectsDescendantParent(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	a, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "A"})
	b, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:        "B",
		ParentTaskID: &a.ID,
	})

	_, err := svc.Update(a.ID, contactID, vaultID, dto.UpdateTaskRequest{
		Label:        "A",
		ParentTaskID: dto.NullableUint{Present: true, Valid: true, Value: b.ID},
	})
	if err != ErrInvalidParentTask {
		t.Errorf("expected ErrInvalidParentTask, got %v", err)
	}
}

func TestContactScopedParentTaskIDLeaveUnchanged(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	parent, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "P"})
	child, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:        "C",
		ParentTaskID: &parent.ID,
	})

	updated, err := svc.Update(child.ID, contactID, vaultID, dto.UpdateTaskRequest{
		Label: "C renamed",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.ParentTaskID == nil || *updated.ParentTaskID != parent.ID {
		t.Errorf("parent_task_id should be unchanged, got %v want %d", updated.ParentTaskID, parent.ID)
	}
}

func TestContactScopedParentTaskIDExplicitClear(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	parent, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "P"})
	child, _ := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:        "C",
		ParentTaskID: &parent.ID,
	})

	updated, err := svc.Update(child.ID, contactID, vaultID, dto.UpdateTaskRequest{
		Label:        "C",
		ParentTaskID: dto.NullableUint{Present: true, Valid: false},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.ParentTaskID != nil {
		t.Errorf("parent_task_id should be cleared, got %v", updated.ParentTaskID)
	}
}
