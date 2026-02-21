package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactSortTest(t *testing.T) (*ContactSortService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "sort-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewContactSortService(db), resp.User.ID
}

func TestContactSortUpdate(t *testing.T) {
	svc, userID := setupContactSortTest(t)

	err := svc.UpdateSort(userID, dto.UpdateContactSortRequest{SortOrder: "first_name"})
	if err != nil {
		t.Fatalf("UpdateSort failed: %v", err)
	}

	var user models.User
	svc.db.First(&user, "id = ?", userID)
	if user.ContactSortOrder != "first_name" {
		t.Errorf("Expected sort order 'first_name', got '%s'", user.ContactSortOrder)
	}
}

func TestContactSortUpdateUserNotFound(t *testing.T) {
	svc, _ := setupContactSortTest(t)

	err := svc.UpdateSort("nonexistent-user", dto.UpdateContactSortRequest{SortOrder: "first_name"})
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
