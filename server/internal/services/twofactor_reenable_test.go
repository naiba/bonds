package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
	"github.com/pquerna/otp/totp"
)

// Enable issues a fresh secret and clears two_factor_confirmed_at. If it ran
// for an account whose 2FA is already confirmed, a stolen session could switch
// 2FA off (or re-point it at the attacker's authenticator) without ever
// presenting a code. Re-enrolment has to go through Disable, which demands one.
func TestEnable2FA_RejectsAccountWithConfirmedTwoFactor(t *testing.T) {
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

	second, err := tc.twoFactorSvc.Enable(tc.userID)

	if !errors.Is(err, ErrTwoFactorAlreadyEnabled) {
		t.Fatalf("second Enable: err = %v, want ErrTwoFactorAlreadyEnabled", err)
	}
	if second != nil {
		t.Fatal("second Enable returned a new setup payload")
	}
	var user models.User
	if err := tc.db.First(&user, "id = ?", tc.userID).Error; err != nil {
		t.Fatalf("fetch user: %v", err)
	}
	if user.TwoFactorConfirmedAt == nil {
		t.Fatal("two_factor_confirmed_at was cleared: 2FA is effectively disabled")
	}
	if user.TwoFactorSecret == nil || *user.TwoFactorSecret != setup.Secret {
		t.Fatal("the confirmed TOTP secret was replaced")
	}
}

// An abandoned enrolment (never confirmed) must stay restartable.
func TestEnable2FA_AllowsRestartingUnconfirmedSetup(t *testing.T) {
	tc := setupTwoFactorTest(t)
	first, err := tc.twoFactorSvc.Enable(tc.userID)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	second, err := tc.twoFactorSvc.Enable(tc.userID)

	if err != nil {
		t.Fatalf("restarting an unconfirmed setup failed: %v", err)
	}
	if second.Secret == "" || second.Secret == first.Secret {
		t.Fatalf("restart must issue a new secret, got %q (first %q)", second.Secret, first.Secret)
	}
}
