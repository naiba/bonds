package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupUserManagementExtraTest(t *testing.T) (*UserManagementService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Admin",
		LastName:  "User",
		Email:     "user-mgmt-extra-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewUserManagementService(db), resp.User.AccountID, resp.User.ID
}

func TestUserManagementGet(t *testing.T) {
	svc, accountID, userID := setupUserManagementExtraTest(t)

	user, err := svc.Get(userID, accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if user.ID != userID {
		t.Errorf("Expected ID '%s', got '%s'", userID, user.ID)
	}
	if user.Email != "user-mgmt-extra-test@example.com" {
		t.Errorf("Expected email 'user-mgmt-extra-test@example.com', got '%s'", user.Email)
	}
	if user.FirstName != "Admin" {
		t.Errorf("Expected first name 'Admin', got '%s'", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("Expected last name 'User', got '%s'", user.LastName)
	}
}

func TestUserManagementGetNotFound(t *testing.T) {
	svc, accountID, _ := setupUserManagementExtraTest(t)

	_, err := svc.Get("non-existent-user-id", accountID)
	if err != ErrManagedUserNotFound {
		t.Errorf("Expected ErrManagedUserNotFound, got %v", err)
	}
}

func TestUserManagementGetWrongAccount(t *testing.T) {
	svc, _, userID := setupUserManagementExtraTest(t)

	_, err := svc.Get(userID, "wrong-account-id")
	if err != ErrManagedUserNotFound {
		t.Errorf("Expected ErrManagedUserNotFound, got %v", err)
	}
}

func TestUserManagementGetAnotherUser(t *testing.T) {
	svc, accountID, _ := setupUserManagementExtraTest(t)

	db := testutil.SetupTestDB(t)
	svc2 := NewUserManagementService(db)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second-user@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	created, err := svc.Create(accountID, dto.CreateManagedUserRequest{
		Email:     "managed-user@example.com",
		FirstName: "Managed",
		LastName:  "User",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.Get(created.ID, accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Email != "managed-user@example.com" {
		t.Errorf("Expected email 'managed-user@example.com', got '%s'", got.Email)
	}

	_, err = svc2.Get(created.ID, resp.User.AccountID)
	if err != ErrManagedUserNotFound {
		t.Errorf("Expected ErrManagedUserNotFound for wrong account, got %v", err)
	}
}
