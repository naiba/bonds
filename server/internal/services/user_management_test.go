package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupUserManagementTest(t *testing.T) (*UserManagementService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Admin", LastName: "User",
		Email: "admin-user@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewUserManagementService(db), resp.User.AccountID, resp.User.ID
}

func TestUserManagementList(t *testing.T) {
	svc, accountID, _ := setupUserManagementTest(t)

	users, err := svc.List(accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestUserManagementCreate(t *testing.T) {
	svc, accountID, _ := setupUserManagementTest(t)

	user, err := svc.Create(accountID, dto.CreateManagedUserRequest{
		Email: "new-user@example.com", FirstName: "New", LastName: "User",
		Password: "password123", IsAdmin: false,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.Email != "new-user@example.com" {
		t.Errorf("Expected email 'new-user@example.com', got '%s'", user.Email)
	}
}

func TestUserManagementCreateDuplicate(t *testing.T) {
	svc, accountID, _ := setupUserManagementTest(t)

	_, err := svc.Create(accountID, dto.CreateManagedUserRequest{
		Email: "admin-user@example.com", Password: "password123",
	})
	if err != ErrEmailExists {
		t.Errorf("Expected ErrEmailExists, got %v", err)
	}
}

func TestUserManagementUpdate(t *testing.T) {
	svc, accountID, _ := setupUserManagementTest(t)

	created, err := svc.Create(accountID, dto.CreateManagedUserRequest{
		Email: "update-user@example.com", FirstName: "Old", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, accountID, dto.UpdateManagedUserRequest{FirstName: "New", IsAdmin: true})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.FirstName != "New" {
		t.Errorf("Expected first name 'New', got '%s'", updated.FirstName)
	}
	if !updated.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestUserManagementDeleteCannotDeleteSelf(t *testing.T) {
	svc, accountID, userID := setupUserManagementTest(t)

	err := svc.Delete(userID, accountID, userID)
	if err != ErrCannotDeleteSelf {
		t.Errorf("Expected ErrCannotDeleteSelf, got %v", err)
	}
}

func TestUserManagementDelete(t *testing.T) {
	svc, accountID, userID := setupUserManagementTest(t)

	created, err := svc.Create(accountID, dto.CreateManagedUserRequest{
		Email: "delete-user@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, accountID, userID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	users, _ := svc.List(accountID)
	if len(users) != 1 {
		t.Errorf("Expected 1 user after delete, got %d", len(users))
	}
}
