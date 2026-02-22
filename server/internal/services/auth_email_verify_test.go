package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupAuthWithEmailVerification(t *testing.T) (*AuthService, *gorm.DB, *SystemSettingService) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)
	settings := NewSystemSettingService(db)
	svc.SetSystemSettings(settings)
	svc.SetMailer(&NoopMailer{})
	settings.Set("auth.require_email_verification", "true")
	settings.Set("smtp.host", "test-smtp")
	return svc, db, settings
}

func TestRegisterFirstUserAutoVerified(t *testing.T) {
	svc, db, _ := setupAuthWithEmailVerification(t)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	var user models.User
	if err := db.First(&user, "email = ?", "first@example.com").Error; err != nil {
		t.Fatalf("User not found: %v", err)
	}
	if user.EmailVerifiedAt == nil {
		t.Error("expected first user (instance admin) to have EmailVerifiedAt set")
	}
}

func TestRegisterSecondUserNotVerified(t *testing.T) {
	svc, db, _ := setupAuthWithEmailVerification(t)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	_, err = svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var user models.User
	if err := db.First(&user, "email = ?", "second@example.com").Error; err != nil {
		t.Fatalf("User not found: %v", err)
	}
	if user.EmailVerifiedAt != nil {
		t.Error("expected second user to have EmailVerifiedAt nil when verification is required")
	}
	if user.EmailVerificationToken == nil {
		t.Error("expected second user to have EmailVerificationToken set")
	}
}

func TestRegisterSkipsVerificationWhenDisabled(t *testing.T) {
	svc, db, settings := setupAuthWithEmailVerification(t)
	settings.Set("auth.require_email_verification", "false")

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	_, err = svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var user models.User
	if err := db.First(&user, "email = ?", "second@example.com").Error; err != nil {
		t.Fatalf("User not found: %v", err)
	}
	if user.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set when verification is disabled")
	}
}

func TestRegisterSkipsVerificationWhenNoSMTP(t *testing.T) {
	svc, db, settings := setupAuthWithEmailVerification(t)
	settings.Set("smtp.host", "")

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	_, err = svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var user models.User
	if err := db.First(&user, "email = ?", "second@example.com").Error; err != nil {
		t.Fatalf("User not found: %v", err)
	}
	if user.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set when SMTP is not configured")
	}
}

func TestVerifyEmail(t *testing.T) {
	svc, db, _ := setupAuthWithEmailVerification(t)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	_, err = svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var user models.User
	if err := db.First(&user, "email = ?", "second@example.com").Error; err != nil {
		t.Fatalf("User not found: %v", err)
	}
	if user.EmailVerificationToken == nil {
		t.Fatal("expected EmailVerificationToken to be set")
	}

	resp, err := svc.VerifyEmail(*user.EmailVerificationToken)
	if err != nil {
		t.Fatalf("VerifyEmail failed: %v", err)
	}
	if resp.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set in response")
	}

	var verified models.User
	if err := db.First(&verified, "email = ?", "second@example.com").Error; err != nil {
		t.Fatalf("User not found after verify: %v", err)
	}
	if verified.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set in DB")
	}
	if verified.EmailVerificationToken != nil {
		t.Error("expected EmailVerificationToken to be nil after verification")
	}
}

func TestVerifyEmailInvalidToken(t *testing.T) {
	svc, _, _ := setupAuthWithEmailVerification(t)

	_, err := svc.VerifyEmail("invalid-token-12345")
	if err != ErrInvalidEmailVerificationToken {
		t.Errorf("expected ErrInvalidEmailVerificationToken, got %v", err)
	}
}

func TestVerifyEmailAlreadyVerified(t *testing.T) {
	svc, db, _ := setupAuthWithEmailVerification(t)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	_, err = svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var user models.User
	db.First(&user, "email = ?", "second@example.com")
	token := *user.EmailVerificationToken

	_, err = svc.VerifyEmail(token)
	if err != nil {
		t.Fatalf("First VerifyEmail failed: %v", err)
	}

	_, err = svc.VerifyEmail(token)
	if err != ErrInvalidEmailVerificationToken {
		t.Errorf("expected ErrInvalidEmailVerificationToken (token cleared after verify), got %v", err)
	}
}

func TestResendVerification(t *testing.T) {
	svc, db, _ := setupAuthWithEmailVerification(t)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	resp, err := svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var userBefore models.User
	db.First(&userBefore, "email = ?", "second@example.com")
	oldToken := userBefore.EmailVerificationToken

	err = svc.ResendVerification(resp.User.ID)
	if err != nil {
		t.Fatalf("ResendVerification failed: %v", err)
	}

	var userAfter models.User
	db.First(&userAfter, "email = ?", "second@example.com")
	if userAfter.EmailVerificationToken == nil {
		t.Error("expected EmailVerificationToken to be set after resend")
	}
	if oldToken != nil && userAfter.EmailVerificationToken != nil && *oldToken == *userAfter.EmailVerificationToken {
		t.Error("expected new token to differ from old token after resend")
	}
}

func TestResendVerificationAlreadyVerified(t *testing.T) {
	svc, db, _ := setupAuthWithEmailVerification(t)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}

	resp, err := svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}

	var user models.User
	db.First(&user, "email = ?", "second@example.com")
	svc.VerifyEmail(*user.EmailVerificationToken)

	err = svc.ResendVerification(resp.User.ID)
	if err != ErrEmailAlreadyVerified {
		t.Errorf("expected ErrEmailAlreadyVerified, got %v", err)
	}
}
