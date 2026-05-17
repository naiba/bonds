package services

import (
	"sync"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestVaultTaskRejectsDescendantParent(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	a, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "A"})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "B",
		ParentTaskID: &a.ID,
	})
	if err != nil {
		t.Fatalf("create B as child of A: %v", err)
	}

	_, err = svc.Update(a.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:        "A",
		ParentTaskID: dto.NullableUint{Present: true, Valid: true, Value: b.ID},
	})
	if err != ErrInvalidParentTask {
		t.Errorf("setting parent to descendant should fail, got %v", err)
	}
}

func TestVaultTaskRejectsThreeNodeCycle(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	a, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "A"})
	b, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "B", ParentTaskID: &a.ID})
	c, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "C", ParentTaskID: &b.ID})

	_, err := svc.Update(a.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:        "A",
		ParentTaskID: dto.NullableUint{Present: true, Valid: true, Value: c.ID},
	})
	if err != ErrInvalidParentTask {
		t.Errorf("expected ErrInvalidParentTask for A->C cycle, got %v", err)
	}
}

func TestVaultTaskParentTaskIDLeaveUnchanged(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	parent, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "P"})
	child, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "C",
		ParentTaskID: &parent.ID,
	})

	updated, err := svc.Update(child.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label: "C renamed",
	})
	if err != nil {
		t.Fatalf("update without parent_task_id: %v", err)
	}
	if updated.ParentTaskID == nil || *updated.ParentTaskID != parent.ID {
		t.Errorf("parent_task_id should be unchanged, got %v want %d", updated.ParentTaskID, parent.ID)
	}
}

func TestVaultTaskParentTaskIDExplicitClear(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	parent, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "P"})
	child, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "C",
		ParentTaskID: &parent.ID,
	})

	updated, err := svc.Update(child.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:        "C",
		ParentTaskID: dto.NullableUint{Present: true, Valid: false},
	})
	if err != nil {
		t.Fatalf("update with cleared parent_task_id: %v", err)
	}
	if updated.ParentTaskID != nil {
		t.Errorf("parent_task_id should be cleared, got %v", updated.ParentTaskID)
	}
}

func TestVaultTaskDeleteCascadesSubTasks(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	parent, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "P"})
	child, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "C",
		ParentTaskID: &parent.ID,
	})
	grand, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "G",
		ParentTaskID: &child.ID,
	})

	if err := svc.Delete(parent.ID, vaultID); err != nil {
		t.Fatalf("delete parent: %v", err)
	}

	for _, id := range []uint{parent.ID, child.ID, grand.ID} {
		var task models.ContactTask
		if err := svc.db.First(&task, id).Error; err == nil {
			t.Errorf("task %d should be soft-deleted, still visible to live queries", id)
		}
	}

	var pivotCount int64
	svc.db.Model(&models.TaskContact{}).
		Where("contact_task_id IN ?", []uint{parent.ID, child.ID, grand.ID}).
		Count(&pivotCount)
	if pivotCount != 0 {
		t.Errorf("expected no orphan pivots, got %d", pivotCount)
	}
}

func TestVaultTaskConcurrentAssigneeReplaceDoesNotDuplicate(t *testing.T) {
	svc, _, vaultID, c1ID, userID := setupVaultTaskTest(t)

	contactSvc := NewContactService(svc.db)
	c2, _ := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Two"})

	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:      "shared",
		ContactIDs: []string{c1ID},
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ids := []string{c1ID, c2.ID}
		svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
			Label:      "shared",
			ContactIDs: &ids,
		})
	}()
	go func() {
		defer wg.Done()
		ids := []string{c2.ID}
		svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
			Label:      "shared",
			ContactIDs: &ids,
		})
	}()
	wg.Wait()

	var rows []models.TaskContact
	if err := svc.db.Where("contact_task_id = ?", task.ID).Find(&rows).Error; err != nil {
		t.Fatalf("read pivots: %v", err)
	}
	seen := make(map[string]int, len(rows))
	for _, r := range rows {
		seen[r.ContactID]++
		if seen[r.ContactID] > 1 {
			t.Errorf("contact %q appears %d times in pivot", r.ContactID, seen[r.ContactID])
		}
	}
	if len(rows) == 0 {
		t.Errorf("expected at least one assignee survives, got 0")
	}
}
