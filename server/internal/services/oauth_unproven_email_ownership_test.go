package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

// An instance that sends no verification mail lets anyone register with any
// address. These tests cover what a provider sign-in does when it meets such an
// account, or one from before email verification existed.

func setupOAuthTestWithoutEmailVerification(t *testing.T) (*OAuthService, *SystemSettingService) {
	t.Helper()
	svc, settings := setupOAuthTestRequiringEmailVerification(t)
	if err := settings.Set("auth.require_email_verification", "false"); err != nil {
		t.Fatalf("switch off email verification: %v", err)
	}
	return svc, settings
}

func registerWithPassword(t *testing.T, svc *OAuthService, email, password string) *dto.AuthResponse {
	t.Helper()
	resp, err := svc.authService().Register(dto.RegisterRequest{FirstName: "First", LastName: "Registrant", Email: email, Password: password}, "en")
	if err != nil {
		t.Fatalf("Register %s: %v", email, err)
	}
	return resp
}

func countRows(t *testing.T, svc *OAuthService, model interface{}, userID string) int64 {
	t.Helper()
	var count int64
	if err := svc.db.Model(model).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	return count
}

// The first registrant can bind a provider identity of their own to the account
// before the real owner arrives. Clearing only the password left that identity
// as a way back in.
func TestFindOrCreateUser_TakeoverRemovesProviderIdentitiesBoundByEarlierRegistrant(t *testing.T) {
	svc, _ := setupOAuthTestRequiringEmailVerification(t)
	registrant := registerThroughOAuthLink(t, svc, &OAuthLinkInfo{
		Provider: "github", ProviderUserID: "gh-first-registrant", Email: "first-registrant@example.com", EmailVerified: true,
	}, "victim@example.com")
	if registrant.EmailVerifiedAt != nil {
		t.Fatal("test setup: the account must start unverified")
	}
	if got := countRows(t, svc, &models.UserToken{}, registrant.ID); got != 1 {
		t.Fatalf("test setup: registrant has %d provider identities, want 1", got)
	}

	ownerSession, _, err := svc.FindOrCreateUser("google", "google-real-owner", "victim@example.com", true, "Real Owner", "en")
	if err != nil || ownerSession == nil || ownerSession.User.ID != registrant.ID {
		t.Fatalf("owner sign-in = %+v, %v; want a session for the existing account", ownerSession, err)
	}

	resp, _, err := svc.FindOrCreateUser("github", "gh-first-registrant", "first-registrant@example.com", true, "First Registrant", "en")
	if !errors.Is(err, ErrOAuthAccountNotLinked) || resp != nil {
		t.Fatalf("first registrant's provider sign-in = %+v, %v; want ErrOAuthAccountNotLinked and no session", resp, err)
	}
	var remaining []models.UserToken
	if err := svc.db.Where("user_id = ?", registrant.ID).Find(&remaining).Error; err != nil {
		t.Fatalf("list provider identities: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Driver != "google" || remaining[0].DriverID != "google-real-owner" {
		t.Fatalf("provider identities after takeover = %+v, want only the owner's", remaining)
	}
}

// Without verification mail the account is marked verified at registration, so
// "verified" proves nothing about who owns the address. Auto-linking it handed
// the real owner an account whose password the first registrant still knew.
func TestFindOrCreateUser_DoesNotAutoLinkAccountRegisteredWithoutVerification(t *testing.T) {
	svc, _ := setupOAuthTestWithoutEmailVerification(t)
	registrant := registerWithPassword(t, svc, "victim@example.com", "registrant-password")

	resp, linkInfo, err := svc.FindOrCreateUser("google", "google-real-owner", "victim@example.com", true, "Real Owner", "en")

	if !errors.Is(err, ErrOAuthAccountNotLinked) || resp != nil {
		t.Fatalf("FindOrCreateUser = %+v, %v; want ErrOAuthAccountNotLinked and no session", resp, err)
	}
	if linkInfo == nil || linkInfo.ProviderUserID != "google-real-owner" || !linkInfo.EmailVerified {
		t.Fatalf("linkInfo = %+v, want the owner's identity for the manual binding flow", linkInfo)
	}
	if got := countRows(t, svc, &models.UserToken{}, registrant.User.ID); got != 0 {
		t.Fatalf("%d provider identities were bound without proof of ownership", got)
	}
}

// Accounts created before email verification existed have no verification time.
// On an instance that does not require verification they work normally, and
// their owners must not lose password, tokens or passkeys by signing in through
// a provider for the first time.
func TestFindOrCreateUser_LegacyUnverifiedAccountKeepsItsCredentials(t *testing.T) {
	svc, _ := setupOAuthTestWithoutEmailVerification(t)
	legacy := registerWithPassword(t, svc, "legacy@example.com", "legacy-password")
	if err := svc.db.Model(&models.User{}).Where("id = ?", legacy.User.ID).Update("email_verified_at", nil).Error; err != nil {
		t.Fatalf("make the account look like a legacy one: %v", err)
	}
	if err := svc.db.Create(&models.PersonalAccessToken{UserID: legacy.User.ID, AccountID: legacy.User.AccountID, Name: "dav client", TokenHash: "hash-of-legacy-dav-token", TokenHint: "bonds_..."}).Error; err != nil {
		t.Fatalf("create PAT: %v", err)
	}

	resp, _, err := svc.FindOrCreateUser("google", "google-legacy-owner", "legacy@example.com", true, "Legacy Owner", "en")

	if !errors.Is(err, ErrOAuthAccountNotLinked) || resp != nil {
		t.Fatalf("FindOrCreateUser = %+v, %v; want the manual binding flow", resp, err)
	}
	if _, err := svc.authService().Login(dto.LoginRequest{Email: "legacy@example.com", Password: "legacy-password"}); err != nil {
		t.Errorf("login with the legacy password after the provider sign-in attempt: %v", err)
	}
	if got := countRows(t, svc, &models.PersonalAccessToken{}, legacy.User.ID); got != 1 {
		t.Errorf("personal access tokens = %d, want the existing one kept", got)
	}
}

// With password sign-in switched off nobody can "sign in first, then bind", so
// the account goes to the owner the provider vouches for, minus old credentials.
func TestFindOrCreateUser_TakesOverUnprovenAccountWhenPasswordSignInIsOff(t *testing.T) {
	svc, settings := setupOAuthTestWithoutEmailVerification(t)
	registrant := registerWithPassword(t, svc, "victim@example.com", "registrant-password")
	if err := settings.Set("auth.password.enabled", "false"); err != nil {
		t.Fatalf("switch off password sign-in: %v", err)
	}

	resp, _, err := svc.FindOrCreateUser("google", "google-real-owner", "victim@example.com", true, "Real Owner", "en")

	if err != nil || resp == nil || resp.User.ID != registrant.User.ID {
		t.Fatalf("FindOrCreateUser = %+v, %v; want a session for the existing account", resp, err)
	}
	var owner models.User
	if err := svc.db.First(&owner, "id = ?", registrant.User.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if owner.Password != nil {
		t.Error("the first registrant's password survived the takeover")
	}
	if owner.EmailOwnershipUnproven {
		t.Error("ownership is still recorded as unproven after the provider vouched for it")
	}
}

// Signing in with the password and then binding a provider identity that
// vouches for the same address proves ownership; later providers auto-link.
func TestLinkOAuthToUser_ProvesOwnershipForLaterProviders(t *testing.T) {
	svc, _ := setupOAuthTestWithoutEmailVerification(t)
	owner := registerWithPassword(t, svc, "owner@example.com", "owner-password")
	_, linkInfo, err := svc.FindOrCreateUser("google", "google-owner", "owner@example.com", true, "Owner", "en")
	if !errors.Is(err, ErrOAuthAccountNotLinked) {
		t.Fatalf("first provider sign-in err = %v, want ErrOAuthAccountNotLinked", err)
	}
	linkToken, err := svc.GenerateLinkToken(linkInfo)
	if err != nil {
		t.Fatalf("GenerateLinkToken: %v", err)
	}

	if _, err := svc.LinkOAuthToUser(linkToken, owner.User.ID); err != nil {
		t.Fatalf("LinkOAuthToUser: %v", err)
	}

	resp, _, err := svc.FindOrCreateUser("github", "gh-owner", "owner@example.com", true, "Owner", "en")
	if err != nil || resp == nil || resp.User.ID != owner.User.ID {
		t.Fatalf("second provider sign-in = %+v, %v; want it linked automatically", resp, err)
	}
	if _, err := svc.authService().Login(dto.LoginRequest{Email: "owner@example.com", Password: "owner-password"}); err != nil {
		t.Errorf("the owner's password stopped working after binding: %v", err)
	}
}
