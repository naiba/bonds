package services

import (
	"fmt"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupUserManagementTest(t *testing.T) (*UserManagementService, *gorm.DB, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Admin", LastName: "User",
		Email: "admin-user@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewUserManagementService(db), db, resp.User.AccountID, resp.User.ID
}

func createTestUser(t *testing.T, db *gorm.DB, accountID, email string) models.User {
	t.Helper()
	firstName := "Test"
	password := "$2a$10$eImDhVHVc96dqKMpMfyMruPLaGrGPR6caDyqnCVq1G1u5IUXY1C5e" // bcrypt("password123")
	user := models.User{
		AccountID: accountID,
		FirstName: &firstName,
		Email:     email,
		Password:  &password,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestUserManagementList(t *testing.T) {
	svc, _, accountID, _ := setupUserManagementTest(t)

	users, meta, err := svc.List(accountID, 0, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
	if meta.Total != 1 {
		t.Errorf("Expected meta.Total=1, got %d", meta.Total)
	}
}

func TestUserManagementList_Pagination(t *testing.T) {
	svc, db, accountID, _ := setupUserManagementTest(t)

	for i := 0; i < 4; i++ {
		createTestUser(t, db, accountID, fmt.Sprintf("page-user%d@example.com", i))
	}

	page1, meta1, err := svc.List(accountID, 1, 2)
	if err != nil {
		t.Fatalf("List page1 failed: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("Expected 2 users on page 1, got %d", len(page1))
	}
	if meta1.Total != 5 {
		t.Errorf("Expected total=5, got %d", meta1.Total)
	}
	if meta1.TotalPages != 3 {
		t.Errorf("Expected total_pages=3, got %d", meta1.TotalPages)
	}
}

func TestUserManagementUpdate(t *testing.T) {
	svc, db, accountID, _ := setupUserManagementTest(t)

	user := createTestUser(t, db, accountID, "update-user@example.com")

	updated, err := svc.Update(user.ID, accountID, dto.UpdateManagedUserRequest{FirstName: "New", IsAdmin: true})
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
	svc, _, accountID, userID := setupUserManagementTest(t)

	err := svc.Delete(userID, accountID, userID)
	if err != ErrCannotDeleteSelf {
		t.Errorf("Expected ErrCannotDeleteSelf, got %v", err)
	}
}

func TestUserManagementDelete(t *testing.T) {
	svc, db, accountID, userID := setupUserManagementTest(t)

	user := createTestUser(t, db, accountID, "delete-user@example.com")

	if err := svc.Delete(user.ID, accountID, userID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	users, _, _ := svc.List(accountID, 0, 0)
	if len(users) != 1 {
		t.Errorf("Expected 1 user after delete, got %d", len(users))
	}
}
