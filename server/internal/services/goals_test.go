package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupGoalTest(t *testing.T) (*GoalService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "goals-test@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewGoalService(db), contact.ID, vault.ID
}

func TestCreateGoal(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	goal, err := svc.Create(contactID, vaultID, dto.CreateGoalRequest{
		Name: "Learn Go",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if goal.Name != "Learn Go" {
		t.Errorf("Expected name 'Learn Go', got '%s'", goal.Name)
	}
	if goal.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, goal.ContactID)
	}
	if goal.Active {
		t.Error("Expected goal to be inactive by default")
	}
	if goal.ID == 0 {
		t.Error("Expected goal ID to be non-zero")
	}
}

func TestListGoals(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreateGoalRequest{Name: "Goal 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreateGoalRequest{Name: "Goal 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	goals, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(goals) != 2 {
		t.Errorf("Expected 2 goals, got %d", len(goals))
	}
}

func TestGetGoal(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateGoalRequest{Name: "Get Goal"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.Get(created.ID, contactID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Get Goal" {
		t.Errorf("Expected name 'Get Goal', got '%s'", got.Name)
	}
	if got.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestUpdateGoal(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateGoalRequest{Name: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	active := true
	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateGoalRequest{
		Name:   "Updated",
		Active: &active,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", updated.Name)
	}
	if !updated.Active {
		t.Error("Expected goal to be active after update")
	}
}

func TestAddStreak(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateGoalRequest{Name: "Streak Goal"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	happenedAt := time.Now()
	got, err := svc.AddStreak(created.ID, contactID, vaultID, dto.AddStreakRequest{HappenedAt: happenedAt})
	if err != nil {
		t.Fatalf("AddStreak failed: %v", err)
	}
	if len(got.Streaks) != 1 {
		t.Fatalf("Expected 1 streak, got %d", len(got.Streaks))
	}
	if got.Streaks[0].GoalID != created.ID {
		t.Errorf("Expected streak goal_id %d, got %d", created.ID, got.Streaks[0].GoalID)
	}
}

func TestDeleteGoal(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateGoalRequest{Name: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	goals, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(goals) != 0 {
		t.Errorf("Expected 0 goals after delete, got %d", len(goals))
	}
}

func TestDeleteGoalNotFound(t *testing.T) {
	svc, contactID, vaultID := setupGoalTest(t)

	err := svc.Delete(9999, contactID, vaultID)
	if err != ErrGoalNotFound {
		t.Errorf("Expected ErrGoalNotFound, got %v", err)
	}
}
