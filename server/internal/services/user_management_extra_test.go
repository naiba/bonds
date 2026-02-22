package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func TestUserManagementGet(t *testing.T) {
	svc, _, accountID, userID := setupUserManagementTest(t)

	user, err := svc.Get(userID, accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if user.ID != userID {
		t.Errorf("Expected ID '%s', got '%s'", userID, user.ID)
	}
	if user.Email != "admin-user@example.com" {
		t.Errorf("Expected email 'admin-user@example.com', got '%s'", user.Email)
	}
	if user.FirstName != "Admin" {
		t.Errorf("Expected first name 'Admin', got '%s'", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("Expected last name 'User', got '%s'", user.LastName)
	}
}

func TestUserManagementGetNotFound(t *testing.T) {
	svc, _, accountID, _ := setupUserManagementTest(t)

	_, err := svc.Get("non-existent-user-id", accountID)
	if err != ErrManagedUserNotFound {
		t.Errorf("Expected ErrManagedUserNotFound, got %v", err)
	}
}

func TestUserManagementGetWrongAccount(t *testing.T) {
	svc, _, _, userID := setupUserManagementTest(t)

	_, err := svc.Get(userID, "wrong-account-id")
	if err != ErrManagedUserNotFound {
		t.Errorf("Expected ErrManagedUserNotFound, got %v", err)
	}
}

func TestUserManagementGetAnotherUser(t *testing.T) {
	svc, db, accountID, _ := setupUserManagementTest(t)

	db2 := testutil.SetupTestDB(t)
	svc2 := NewUserManagementService(db2)
	cfg := testutil.TestJWTConfig()
	authSvc2 := NewAuthService(db2, cfg)

	resp2, err := authSvc2.Register(dto.RegisterRequest{
		FirstName: "Second", LastName: "User",
		Email: "second-user@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	managed := createTestUser(t, db, accountID, "managed-user@example.com")

	got, err := svc.Get(managed.ID, accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Email != "managed-user@example.com" {
		t.Errorf("Expected email 'managed-user@example.com', got '%s'", got.Email)
	}

	_, err = svc2.Get(managed.ID, resp2.User.AccountID)
	if err != ErrManagedUserNotFound {
		t.Errorf("Expected ErrManagedUserNotFound for wrong account, got %v", err)
	}
}
