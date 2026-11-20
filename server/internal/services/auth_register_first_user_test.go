package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// "Is this the first user?" decides who becomes instance administrator and
// whether the registration switch applies. A failed count must abort the
// registration; treating it as "zero users" hands out instance admin.
func TestRegister_FailedUserCountDoesNotGrantInstanceAdmin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "First", LastName: "Admin", Email: "first-admin@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register first user: %v", err)
	}

	countFailure := errors.New("database is locked")
	failNextCount := true
	callbackName := "test:fail_user_count_once"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if failNextCount && tx.Statement.Table == "users" {
			failNextCount = false
			_ = tx.AddError(countFailure)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	_, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Second", LastName: "User", Email: "second-user@example.com", Password: "password123",
	}, "en")

	if !errors.Is(err, countFailure) {
		t.Fatalf("Register err = %v, want the count failure to be returned", err)
	}
	var instanceAdmins int64
	if err := db.Model(&models.User{}).Where("is_instance_administrator = ?", true).Count(&instanceAdmins).Error; err != nil {
		t.Fatalf("count instance admins: %v", err)
	}
	if instanceAdmins != 1 {
		t.Fatalf("instance administrators = %d, want only the genuine first user", instanceAdmins)
	}
	var secondUsers int64
	if err := db.Model(&models.User{}).Where("email = ?", "second-user@example.com").Count(&secondUsers).Error; err != nil {
		t.Fatalf("count second user: %v", err)
	}
	if secondUsers != 0 {
		t.Fatal("the registration went through despite the failed count")
	}
}
