package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

type twoFactorTestContext struct {
	db           *gorm.DB
	twoFactorSvc *TwoFactorService
	authSvc      *AuthService
	userID       string
	userEmail    string
}

func setupTwoFactorTest(t *testing.T) *twoFactorTestContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	twoFactorSvc := NewTwoFactorService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "TwoFactor",
		LastName:  "User",
		Email:     "2fa@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return &twoFactorTestContext{
		db:           db,
		twoFactorSvc: twoFactorSvc,
		authSvc:      authSvc,
		userID:       resp.User.ID,
		userEmail:    resp.User.Email,
	}
}

func TestEnable2FA(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	if setup.Secret == "" {
		t.Error("Expected secret to be non-empty")
	}
	if setup.QRCodeURL == "" {
		t.Error("Expected QR code URL to be non-empty")
	}
	if len(setup.RecoveryCodes) != 8 {
		t.Errorf("Expected 8 recovery codes, got %d", len(setup.RecoveryCodes))
	}
	for i, code := range setup.RecoveryCodes {
		if len(code) != 8 {
			t.Errorf("Recovery code %d: expected length 8, got %d", i, len(code))
		}
	}

	var user models.User
	if err := tc.db.First(&user, "id = ?", tc.userID).Error; err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}
	if user.TwoFactorSecret == nil {
		t.Error("Expected TwoFactorSecret to be set")
	}
	if user.TwoFactorConfirmedAt != nil {
		t.Error("Expected TwoFactorConfirmedAt to be nil before confirmation")
	}
	if user.TwoFactorRecoveryCodes == nil {
		t.Error("Expected TwoFactorRecoveryCodes to be set")
	}
}

func TestConfirm2FA(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	var user models.User
	if err := tc.db.First(&user, "id = ?", tc.userID).Error; err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}
	if user.TwoFactorConfirmedAt == nil {
		t.Error("Expected TwoFactorConfirmedAt to be set after confirmation")
	}
}

func TestConfirm2FAInvalidCode(t *testing.T) {
	tc := setupTwoFactorTest(t)

	if _, err := tc.twoFactorSvc.Enable(tc.userID); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	err := tc.twoFactorSvc.Confirm(tc.userID, "000000")
	if err != ErrInvalidTOTPCode {
		t.Errorf("Expected ErrInvalidTOTPCode, got %v", err)
	}
}

func TestDisable2FA(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	code, err = totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if err := tc.twoFactorSvc.Disable(tc.userID, code); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	var user models.User
	if err := tc.db.First(&user, "id = ?", tc.userID).Error; err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}
	if user.TwoFactorSecret != nil {
		t.Error("Expected TwoFactorSecret to be nil after disable")
	}
	if user.TwoFactorRecoveryCodes != nil {
		t.Error("Expected TwoFactorRecoveryCodes to be nil after disable")
	}
	if user.TwoFactorConfirmedAt != nil {
		t.Error("Expected TwoFactorConfirmedAt to be nil after disable")
	}
}

func TestValidateTOTP(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	code, err = totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	valid, err := tc.twoFactorSvc.Validate(tc.userID, code)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !valid {
		t.Error("Expected TOTP code to be valid")
	}
}

func TestValidateRecoveryCode(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	recoveryCode := setup.RecoveryCodes[0]
	valid, err := tc.twoFactorSvc.Validate(tc.userID, recoveryCode)
	if err != nil {
		t.Fatalf("Validate with recovery code failed: %v", err)
	}
	if !valid {
		t.Error("Expected recovery code to be valid")
	}

	var user models.User
	if err := tc.db.First(&user, "id = ?", tc.userID).Error; err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}
	var remainingCodes []string
	if err := json.Unmarshal([]byte(*user.TwoFactorRecoveryCodes), &remainingCodes); err != nil {
		t.Fatalf("Failed to unmarshal recovery codes: %v", err)
	}
	if len(remainingCodes) != 7 {
		t.Errorf("Expected 7 remaining recovery codes, got %d", len(remainingCodes))
	}
	for _, c := range remainingCodes {
		if c == recoveryCode {
			t.Error("Used recovery code should have been removed")
		}
	}
}

func TestValidateInvalidCode(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	valid, err := tc.twoFactorSvc.Validate(tc.userID, "invalidcode")
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if valid {
		t.Error("Expected invalid code to return false")
	}
}

func TestLoginWith2FA(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	resp, err := tc.authSvc.Login(dto.LoginRequest{
		Email:    tc.userEmail,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login without TOTP code failed: %v", err)
	}
	if !resp.RequiresTwoFactor {
		t.Error("Expected RequiresTwoFactor to be true")
	}
	if resp.TempToken == "" {
		t.Error("Expected TempToken to be non-empty")
	}
	if resp.Token != "" {
		t.Error("Expected Token to be empty when 2FA is required")
	}

	code, err = totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	resp, err = tc.authSvc.Login(dto.LoginRequest{
		Email:    tc.userEmail,
		Password: "password123",
		TOTPCode: code,
	})
	if err != nil {
		t.Fatalf("Login with TOTP code failed: %v", err)
	}
	if resp.RequiresTwoFactor {
		t.Error("Expected RequiresTwoFactor to be false")
	}
	if resp.Token == "" {
		t.Error("Expected Token to be non-empty")
	}
}

func TestLoginWith2FAInvalidCode(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	_, err = tc.authSvc.Login(dto.LoginRequest{
		Email:    tc.userEmail,
		Password: "password123",
		TOTPCode: "000000",
	})
	if err != ErrInvalidTOTPCode {
		t.Errorf("Expected ErrInvalidTOTPCode, got %v", err)
	}
}

func TestLoginWith2FARecoveryCode(t *testing.T) {
	tc := setupTwoFactorTest(t)

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	resp, err := tc.authSvc.Login(dto.LoginRequest{
		Email:    tc.userEmail,
		Password: "password123",
		TOTPCode: setup.RecoveryCodes[0],
	})
	if err != nil {
		t.Fatalf("Login with recovery code failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected Token to be non-empty")
	}
	if resp.RequiresTwoFactor {
		t.Error("Expected RequiresTwoFactor to be false")
	}
}

func TestIsEnabled(t *testing.T) {
	tc := setupTwoFactorTest(t)

	enabled, err := tc.twoFactorSvc.IsEnabled(tc.userID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("Expected 2FA to be disabled initially")
	}

	setup, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	enabled, err = tc.twoFactorSvc.IsEnabled(tc.userID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("Expected 2FA to not be enabled before confirmation")
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}
	if err := tc.twoFactorSvc.Confirm(tc.userID, code); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	enabled, err = tc.twoFactorSvc.IsEnabled(tc.userID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("Expected 2FA to be enabled after confirmation")
	}
}
