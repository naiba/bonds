package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

// An account with 2FA switched on must be asked for its code no matter how the
// first factor was presented. OAuth sign-in used to hand out a full session
// straight away, so a compromised provider account skipped 2FA entirely.
func TestFindOrCreateUser_AsksForSecondFactorWhenAccountHasTwoFactor(t *testing.T) {
	svc := setupOAuthTest(t)
	authSvc := NewAuthService(svc.db, svc.jwt)
	registered, err := authSvc.Register(dto.RegisterRequest{FirstName: "Two", LastName: "Factor", Email: "two-factor@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, _, err := svc.FindOrCreateUser("github", "gh-two-factor", "two-factor@example.com", true, "Two Factor", "en"); err != nil {
		t.Fatalf("link provider identity: %v", err)
	}
	confirmedAt := time.Now()
	if err := svc.db.Model(&models.User{}).Where("id = ?", registered.User.ID).Update("two_factor_confirmed_at", confirmedAt).Error; err != nil {
		t.Fatalf("enable 2FA: %v", err)
	}

	resp, _, err := svc.FindOrCreateUser("github", "gh-two-factor", "two-factor@example.com", true, "Two Factor", "en")

	if err != nil {
		t.Fatalf("FindOrCreateUser: %v", err)
	}
	if resp.Token != "" {
		t.Error("OAuth sign-in issued a full session token to an account protected by 2FA")
	}
	if !resp.RequiresTwoFactor || resp.TempToken == "" {
		t.Errorf("requires_two_factor = %v, temp token present = %v; want a pending 2FA challenge", resp.RequiresTwoFactor, resp.TempToken != "")
	}
}

// Anyone can register with somebody else's address; until the address is
// verified the account proves nothing. When the real owner later signs in
// through a provider that vouches for the address, the account is linked and
// marked verified, so the password chosen by the first registrant has to stop
// working, together with every session and token created with it.
func TestFindOrCreateUser_LinkingUnverifiedAccountRevokesItsEarlierCredentials(t *testing.T) {
	svc := setupOAuthTest(t)
	authSvc := NewAuthService(svc.db, svc.jwt)
	registered, err := authSvc.Register(dto.RegisterRequest{FirstName: "Not", LastName: "Owner", Email: "victim@example.com", Password: "registrant-password"}, "en")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc.db.Model(&models.User{}).Where("id = ?", registered.User.ID).Update("email_verified_at", nil).Error; err != nil {
		t.Fatalf("mark account unverified: %v", err)
	}
	if err := svc.db.Create(&models.PersonalAccessToken{UserID: registered.User.ID, AccountID: registered.User.AccountID, Name: "created before verification", TokenHash: "hash-created-before-verification", TokenHint: "bonds_..."}).Error; err != nil {
		t.Fatalf("create PAT: %v", err)
	}
	if _, err := authSvc.Login(dto.LoginRequest{Email: "victim@example.com", Password: "registrant-password"}); err != nil && !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("test setup: registrant password should be valid before linking, got %v", err)
	}

	if _, _, err := svc.FindOrCreateUser("google", "google-real-owner", "victim@example.com", true, "Real Owner", "en"); err != nil {
		t.Fatalf("FindOrCreateUser: %v", err)
	}

	if _, err := authSvc.Login(dto.LoginRequest{Email: "victim@example.com", Password: "registrant-password"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("login with the pre-verification password = %v, want ErrInvalidCredentials", err)
	}
	var linked models.User
	if err := svc.db.First(&linked, "id = ?", registered.User.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if linked.AuthVersion == 0 {
		t.Error("sessions issued before verification were not revoked")
	}
	var remainingTokens int64
	if err := svc.db.Model(&models.PersonalAccessToken{}).Where("user_id = ?", registered.User.ID).Count(&remainingTokens).Error; err != nil {
		t.Fatalf("count PATs: %v", err)
	}
	if remainingTokens != 0 {
		t.Errorf("personal access tokens created before verification = %d, want 0", remainingTokens)
	}
}

// The owner of an already verified account keeps the password after linking.
func TestFindOrCreateUser_LinkingVerifiedAccountKeepsItsPassword(t *testing.T) {
	svc := setupOAuthTest(t)
	authSvc := NewAuthService(svc.db, svc.jwt)
	if _, err := authSvc.Register(dto.RegisterRequest{FirstName: "Real", LastName: "Owner", Email: "owner@example.com", Password: "owner-password"}, "en"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, _, err := svc.FindOrCreateUser("google", "google-owner", "owner@example.com", true, "Real Owner", "en"); err != nil {
		t.Fatalf("FindOrCreateUser: %v", err)
	}

	if _, err := authSvc.Login(dto.LoginRequest{Email: "owner@example.com", Password: "owner-password"}); err != nil {
		t.Errorf("login with the owner's password after linking: %v", err)
	}
}

// A provider that says the address is unverified in one claim must not be
// overruled by an unrelated truthy claim in the same payload.
func TestOAuthEmailVerified_ExplicitUnverifiedClaimWins(t *testing.T) {
	for name, rawData := range map[string]map[string]interface{}{
		"unverified email, verified profile flag": {"email_verified": false, "verified": true},
		"string false, later true":                {"email_verified": "false", "verified_email": "true"},
	} {
		if oauthEmailVerified("oidc", rawData) {
			t.Errorf("%s: treated as verified", name)
		}
	}
}
