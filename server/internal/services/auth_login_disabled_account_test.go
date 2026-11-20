package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

// "This account is disabled" confirms that the account exists. It may only be
// said to someone who has proven they know the password; anyone else gets the
// same answer as for an unknown address.
func TestLogin_RevealsDisabledStateOnlyAfterTheCorrectPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Disabled", LastName: "User", Email: "disabled-login@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := db.Model(&models.User{}).Where("email = ?", "disabled-login@example.com").Update("disabled", true).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	_, wrongPasswordErr := authSvc.Login(dto.LoginRequest{Email: "disabled-login@example.com", Password: "not-the-password"})
	_, unknownAccountErr := authSvc.Login(dto.LoginRequest{Email: "nobody@example.com", Password: "not-the-password"})
	_, correctPasswordErr := authSvc.Login(dto.LoginRequest{Email: "disabled-login@example.com", Password: "password123"})

	if !errors.Is(wrongPasswordErr, ErrInvalidCredentials) {
		t.Errorf("disabled account, wrong password: err = %v, want ErrInvalidCredentials", wrongPasswordErr)
	}
	if !errors.Is(unknownAccountErr, ErrInvalidCredentials) {
		t.Errorf("unknown account: err = %v, want ErrInvalidCredentials", unknownAccountErr)
	}
	if !errors.Is(correctPasswordErr, ErrUserDisabled) {
		t.Errorf("disabled account, correct password: err = %v, want ErrUserDisabled", correctPasswordErr)
	}
}
