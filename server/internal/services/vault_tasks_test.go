package services

import (
	"reflect"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultTaskTest(t *testing.T) (*VaultTaskService, *TaskService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-tasks-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewVaultTaskService(db), NewTaskService(db), vault.ID, contact.ID, resp.User.ID
}

func hasAssignee(task *dto.VaultTaskResponse, contactID string) bool {
	for _, c := range task.Contacts {
		if c.ID == contactID {
			return true
		}
	}
	return false
}

func createTaskStatusForVaultTaskTest(t *testing.T, svc *VaultTaskService, vaultID, name string) string {
	t.Helper()

	var vault models.Vault
	if err := svc.db.Where("id = ?", vaultID).First(&vault).Error; err != nil {
		t.Fatalf("load vault failed: %v", err)
	}

	resp, err := NewPersonalizeService(svc.db).Create(vault.AccountID, "task-statuses", dto.PersonalizeEntityRequest{Name: name})
	if err != nil {
		t.Fatalf("create task status failed: %v", err)
	}
	if resp.Slug == "" {
		t.Fatalf("created task status has empty slug: %+v", resp)
	}
	return resp.Slug
}

func TestVaultTaskListEmpty(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	tasks, err := svc.List(vaultID, VaultTaskFilters{}, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestVaultTaskListWithTasks(t *testing.T) {
	svc, taskSvc, vaultID, contactID, userID := setupVaultTaskTest(t)

	_, err := taskSvc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "Task 1"})
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}
	_, err = taskSvc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "Task 2"})
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	tasks, err := svc.List(vaultID, VaultTaskFilters{}, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestVaultTaskCreateStandalone(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	task, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label: "Personal todo (no contact)",
	})
	if err != nil {
		t.Fatalf("Create standalone failed: %v", err)
	}
	if len(task.Contacts) != 0 {
		t.Errorf("standalone task should have no assignees, got %d", len(task.Contacts))
	}
	if task.VaultID != vaultID {
		t.Errorf("VaultID mismatch: got %q, want %q", task.VaultID, vaultID)
	}
	if task.Status != models.TaskStatusTodo {
		t.Errorf("default status should be 'todo', got %q", task.Status)
	}
}

func TestVaultTaskCreateWithContact(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)

	task, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:      "Birthday card",
		ContactIDs: []string{contactID},
		Status:     models.TaskStatusInProgress,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !hasAssignee(task, contactID) {
		t.Errorf("expected contact %q in assignees, got %+v", contactID, task.Contacts)
	}
	if task.Status != models.TaskStatusInProgress {
		t.Errorf("Status mismatch: got %q, want 'in_progress'", task.Status)
	}
	if task.Contacts[0].Name == "" {
		t.Errorf("contact name should be populated, got empty")
	}
}

func TestVaultTaskCreateMultiContact(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)
	db := svc.db
	contactSvc := NewContactService(db)
	second, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Mary"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	task, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:      "Plan dinner",
		ContactIDs: []string{contactID, second.ID},
	})
	if err != nil {
		t.Fatalf("Create multi failed: %v", err)
	}
	if len(task.Contacts) != 2 {
		t.Fatalf("expected 2 assignees, got %d", len(task.Contacts))
	}
	if !hasAssignee(task, contactID) || !hasAssignee(task, second.ID) {
		t.Errorf("missing one of the expected assignees: %+v", task.Contacts)
	}
}

func TestVaultTaskCreateRejectsInvalidStatus(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	_, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:  "Task",
		Status: "garbage",
	})
	if err != ErrInvalidTaskStatus {
		t.Errorf("expected ErrInvalidTaskStatus, got %v", err)
	}
}

func TestVaultTaskCreateRejectsCrossVaultContact(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	_, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:      "Bad",
		ContactIDs: []string{"00000000-0000-0000-0000-000000000000"},
	})
	if err == nil {
		t.Errorf("expected error for nonexistent contact, got nil")
	}
}

func TestVaultTaskUpdateStatusToDoneSetsCompleted(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "x"})
	updated, err := svc.UpdateStatus(task.ID, vaultID, dto.UpdateTaskStatusRequest{
		Status: models.TaskStatusDone,
	}, userID)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if updated.Status != models.TaskStatusDone {
		t.Errorf("status not updated, got %q", updated.Status)
	}
	if !updated.Completed {
		t.Errorf("moving to Done should set Completed=true")
	}
	if updated.CompletedAt == nil {
		t.Errorf("moving to Done should set CompletedAt")
	}
}

func TestVaultTaskUpdateStatusFromDoneClearsCompleted(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label: "x", Status: models.TaskStatusDone,
	})
	_, _ = svc.UpdateStatus(task.ID, vaultID, dto.UpdateTaskStatusRequest{Status: models.TaskStatusDone}, userID)
	updated, err := svc.UpdateStatus(task.ID, vaultID, dto.UpdateTaskStatusRequest{
		Status: models.TaskStatusInProgress,
	}, userID)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if updated.Completed {
		t.Errorf("moving away from Done should clear Completed")
	}
}

func TestVaultTaskUpdateStatusRejectsInvalid(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)
	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "x"})

	_, err := svc.UpdateStatus(task.ID, vaultID, dto.UpdateTaskStatusRequest{Status: "neon"}, userID)
	if err != ErrInvalidTaskStatus {
		t.Errorf("expected ErrInvalidTaskStatus, got %v", err)
	}
}

func TestVaultTaskUpdateStatusAcceptsCustomStatus(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)
	customStatus := createTaskStatusForVaultTaskTest(t, svc, vaultID, "Waiting Review")
	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "x"})

	updated, err := svc.UpdateStatus(task.ID, vaultID, dto.UpdateTaskStatusRequest{Status: customStatus}, userID)
	if err != nil {
		t.Fatalf("UpdateStatus rejected account task status %q: %v", customStatus, err)
	}
	if updated.Status != customStatus {
		t.Errorf("status mismatch: got %q, want %q", updated.Status, customStatus)
	}
}

func TestVaultTaskUpdatePositionMovesAcrossColumns(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "x"})
	updated, err := svc.UpdatePosition(task.ID, vaultID, dto.UpdateTaskPositionRequest{
		Position: 5,
		Status:   models.TaskStatusInProgress,
	}, userID)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}
	if updated.Position != 0 {
		t.Errorf("position should be normalized within destination column, got %d want 0", updated.Position)
	}
	if updated.Status != models.TaskStatusInProgress {
		t.Errorf("status not updated, got %q", updated.Status)
	}
}

func TestVaultTaskUpdatePositionResequencesDestinationColumn(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	taskA, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "A", Status: models.TaskStatusTodo})
	taskB, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "B", Status: models.TaskStatusTodo})
	taskC, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "C", Status: models.TaskStatusTodo})
	_, _ = svc.UpdatePosition(taskA.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 0, Status: models.TaskStatusTodo}, userID)
	_, _ = svc.UpdatePosition(taskB.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 1, Status: models.TaskStatusTodo}, userID)
	_, _ = svc.UpdatePosition(taskC.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 2, Status: models.TaskStatusTodo}, userID)

	_, err := svc.UpdatePosition(taskC.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 0, Status: models.TaskStatusTodo}, userID)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	tasks, err := svc.List(vaultID, VaultTaskFilters{Status: &[]string{models.TaskStatusTodo}[0]}, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	gotLabels := []string{tasks[0].Label, tasks[1].Label, tasks[2].Label}
	wantLabels := []string{"C", "A", "B"}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("order mismatch: got %v, want %v", gotLabels, wantLabels)
	}
	for i, task := range tasks {
		if task.Position != i {
			t.Fatalf("task %s position mismatch: got %d, want %d", task.Label, task.Position, i)
		}
	}
}

func TestVaultTaskUpdatePositionResequencesSourceAndDestinationColumns(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	taskA, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "A", Status: models.TaskStatusTodo})
	taskB, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "B", Status: models.TaskStatusTodo})
	taskC, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "C", Status: models.TaskStatusInProgress})
	_, _ = svc.UpdatePosition(taskA.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 0, Status: models.TaskStatusTodo}, userID)
	_, _ = svc.UpdatePosition(taskB.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 1, Status: models.TaskStatusTodo}, userID)
	_, _ = svc.UpdatePosition(taskC.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 0, Status: models.TaskStatusInProgress}, userID)

	_, err := svc.UpdatePosition(taskB.ID, vaultID, dto.UpdateTaskPositionRequest{Position: 0, Status: models.TaskStatusInProgress}, userID)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	todoTasks, err := svc.List(vaultID, VaultTaskFilters{Status: &[]string{models.TaskStatusTodo}[0]}, userID)
	if err != nil {
		t.Fatalf("List todo failed: %v", err)
	}
	if len(todoTasks) != 1 || todoTasks[0].Label != "A" || todoTasks[0].Position != 0 {
		t.Fatalf("source column not resequenced: %+v", todoTasks)
	}

	wipTasks, err := svc.List(vaultID, VaultTaskFilters{Status: &[]string{models.TaskStatusInProgress}[0]}, userID)
	if err != nil {
		t.Fatalf("List in_progress failed: %v", err)
	}
	gotLabels := []string{wipTasks[0].Label, wipTasks[1].Label}
	wantLabels := []string{"B", "C"}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("destination order mismatch: got %v, want %v", gotLabels, wantLabels)
	}
	for i, task := range wipTasks {
		if task.Position != i {
			t.Fatalf("destination task %s position mismatch: got %d, want %d", task.Label, task.Position, i)
		}
	}
}

func TestVaultTaskListFilterByStatus(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)
	_, _ = svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "todo task"})
	t2, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "wip task"})
	_, _ = svc.UpdateStatus(t2.ID, vaultID, dto.UpdateTaskStatusRequest{Status: models.TaskStatusInProgress}, userID)

	wip := models.TaskStatusInProgress
	tasks, err := svc.List(vaultID, VaultTaskFilters{Status: &wip}, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 in_progress task, got %d", len(tasks))
	}
}

func TestVaultTaskUpdateAllFields(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)
	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "old"})

	assignees := []string{contactID}
	updated, err := svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:       "new label",
		Description: "richer description",
		ContactIDs:  &assignees,
		Status:      models.TaskStatusInProgress,
	}, userID)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "new label" || updated.Description != "richer description" {
		t.Errorf("fields didn't update: %+v", updated)
	}
	if !hasAssignee(updated, contactID) {
		t.Errorf("expected contact %q in assignees, got %+v", contactID, updated.Contacts)
	}
	if updated.Status != models.TaskStatusInProgress {
		t.Errorf("Status didn't update: got %q", updated.Status)
	}
}

func TestVaultTaskUpdateClearsContactWhenEmpty(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)
	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label: "x", ContactIDs: []string{contactID},
	})
	if !hasAssignee(task, contactID) {
		t.Fatalf("setup: expected assignee, got %+v", task.Contacts)
	}

	empty := []string{}
	updated, err := svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:      "x",
		ContactIDs: &empty,
	}, userID)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(updated.Contacts) != 0 {
		t.Errorf("expected no assignees, got %+v", updated.Contacts)
	}
}

func TestVaultTaskUpdateLeavesAssigneesAloneWhenNil(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)
	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label: "x", ContactIDs: []string{contactID},
	})

	updated, err := svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:      "renamed",
		ContactIDs: nil, // intentionally untouched
	}, userID)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if !hasAssignee(updated, contactID) {
		t.Errorf("nil ContactIDs should leave assignees untouched, got %+v", updated.Contacts)
	}
}

func TestVaultTaskListFilterStandaloneOnly(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)
	_, _ = svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "personal"})
	_, _ = svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "for contact", ContactIDs: []string{contactID}})

	noContact := ""
	tasks, err := svc.List(vaultID, VaultTaskFilters{ContactID: &noContact}, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 standalone task, got %d", len(tasks))
	}
	if len(tasks[0].Contacts) != 0 {
		t.Errorf("standalone task should have no assignees, got %+v", tasks[0].Contacts)
	}
}

func TestVaultTaskListFilterByContact(t *testing.T) {
	svc, _, vaultID, contactID, userID := setupVaultTaskTest(t)
	db := svc.db
	contactSvc := NewContactService(db)
	other, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Other"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, _ = svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "first", ContactIDs: []string{contactID}})
	_, _ = svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "second", ContactIDs: []string{other.ID}})
	_, _ = svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "both", ContactIDs: []string{contactID, other.ID}})

	cid := contactID
	tasks, err := svc.List(vaultID, VaultTaskFilters{ContactID: &cid}, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks assigned to contact, got %d", len(tasks))
	}
}

func TestVaultTaskSubTaskHierarchy(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	parent, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "Plan trip"})
	if err != nil {
		t.Fatalf("Create parent failed: %v", err)
	}
	if parent.ParentTaskID != nil {
		t.Errorf("parent task should have nil ParentTaskID, got %v", parent.ParentTaskID)
	}

	child, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "Book flights",
		ParentTaskID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("Create child failed: %v", err)
	}
	if child.ParentTaskID == nil || *child.ParentTaskID != parent.ID {
		t.Errorf("child ParentTaskID = %v, want %d", child.ParentTaskID, parent.ID)
	}
}

func TestVaultTaskSubTaskRejectsMissingParent(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	bogus := uint(99999)
	_, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:        "Orphan",
		ParentTaskID: &bogus,
	})
	if err != ErrInvalidParentTask {
		t.Errorf("expected ErrInvalidParentTask, got %v", err)
	}
}

func TestVaultTaskSubTaskRejectsSelfParent(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	task, _ := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "x"})
	_, err := svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
		Label:        "x",
		ParentTaskID: dto.NullableUint{Present: true, Valid: true, Value: task.ID},
	}, userID)
	if err != ErrInvalidParentTask {
		t.Errorf("expected ErrInvalidParentTask, got %v", err)
	}
}
