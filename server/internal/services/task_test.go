package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupTaskTest(t *testing.T) (*TaskService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "task-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewTaskService(db), contact.ID, vault.ID, resp.User.ID
}

func TestCreateTask(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	task, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:       "Buy groceries",
		Description: "Milk, eggs, bread",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if task.Label != "Buy groceries" {
		t.Errorf("Expected label 'Buy groceries', got '%s'", task.Label)
	}
	if task.Description != "Milk, eggs, bread" {
		t.Errorf("Expected description 'Milk, eggs, bread', got '%s'", task.Description)
	}
	if task.Completed {
		t.Error("Expected task to not be completed")
	}
	if task.ID == 0 {
		t.Error("Expected task ID to be non-zero")
	}
}

func TestListTasks(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	_, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "Task 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "Task 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	tasks, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestUpdateTask(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateTaskRequest{
		Label:       "Updated",
		Description: "New description",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Updated" {
		t.Errorf("Expected label 'Updated', got '%s'", updated.Label)
	}
	if updated.Description != "New description" {
		t.Errorf("Expected description 'New description', got '%s'", updated.Description)
	}
}

func TestToggleTaskCompleted(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "Toggle me"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	toggled, err := svc.ToggleCompleted(created.ID, contactID, vaultID)
	if err != nil {
		t.Fatalf("ToggleCompleted failed: %v", err)
	}
	if !toggled.Completed {
		t.Error("Expected task to be completed after toggle")
	}
	if toggled.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	toggledBack, err := svc.ToggleCompleted(created.ID, contactID, vaultID)
	if err != nil {
		t.Fatalf("ToggleCompleted back failed: %v", err)
	}
	if toggledBack.Completed {
		t.Error("Expected task to not be completed after second toggle")
	}
	if toggledBack.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil after second toggle")
	}
}

func TestDeleteTask(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{Label: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	tasks, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after delete, got %d", len(tasks))
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupTaskTest(t)

	err := svc.Delete(9999, contactID, vaultID)
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateTaskNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupTaskTest(t)

	_, err := svc.Update(9999, contactID, vaultID, dto.UpdateTaskRequest{Label: "nope"})
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestToggleTaskNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupTaskTest(t)

	_, err := svc.ToggleCompleted(9999, contactID, vaultID)
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}
