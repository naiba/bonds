package services

import (
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

	users, err := svc.List(accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
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

	users, _ := svc.List(accountID)
	if len(users) != 1 {
		t.Errorf("Expected 1 user after delete, got %d", len(users))
	}
}
