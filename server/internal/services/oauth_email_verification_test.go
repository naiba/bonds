package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func registerOAuthVictim(t *testing.T, svc *OAuthService, email string) string {
	t.Helper()
	resp, err := svc.authService().Register(dto.RegisterRequest{
		FirstName: "Victim", LastName: "User", Email: email, Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("register victim: %v", err)
	}
	return resp.User.ID
}

// Account takeover regression: an identity whose email the provider has not
// verified must never be bound to the local account that owns that email.
func TestFindOrCreateUser_DoesNotAutoLinkUnverifiedProviderEmail(t *testing.T) {
	svc, _ := setupOAuthTestRequiringEmailVerification(t)
	victimID := registerOAuthVictim(t, svc, "victim@example.com")

	resp, linkInfo, err := svc.FindOrCreateUser("discord", "attacker-identity", "victim@example.com", false, "Attacker", "en")

	if !errors.Is(err, ErrOAuthAccountNotLinked) {
		t.Fatalf("err = %v, want ErrOAuthAccountNotLinked", err)
	}
	if resp != nil {
		t.Fatal("an unverified provider email produced a session for the existing account")
	}
	if linkInfo == nil || linkInfo.EmailVerified {
		t.Fatalf("linkInfo = %+v, want the unverified identity for the binding flow", linkInfo)
	}
	var tokens int64
	if err := svc.db.Model(&models.UserToken{}).Where("user_id = ?", victimID).Count(&tokens).Error; err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 0 {
		t.Fatalf("%d provider identity(ies) were bound to the victim account", tokens)
	}
	var victim models.User
	if err := svc.db.First(&victim, "id = ?", victimID).Error; err != nil {
		t.Fatalf("fetch victim: %v", err)
	}
	if victim.EmailVerifiedAt != nil {
		t.Fatal("the victim's email was marked verified on the attacker's say-so")
	}
}

func TestOAuthEmailVerified_ReadsEachProvidersClaim(t *testing.T) {
	cases := []struct {
		name         string
		providerType string
		rawData      map[string]interface{}
		want         bool
	}{
		{name: "oidc verified", providerType: "oidc", rawData: map[string]interface{}{"email_verified": true}, want: true},
		{name: "oidc verified as string", providerType: "oidc", rawData: map[string]interface{}{"email_verified": "true"}, want: true},
		{name: "oidc unverified", providerType: "oidc", rawData: map[string]interface{}{"email_verified": false}, want: false},
		{name: "oidc without the claim", providerType: "oidc", rawData: map[string]interface{}{"email": "a@example.com"}, want: false},
		{name: "google verified", providerType: "google", rawData: map[string]interface{}{"verified_email": true}, want: true},
		{name: "google unverified", providerType: "google", rawData: map[string]interface{}{"verified_email": false}, want: false},
		{name: "discord verified", providerType: "discord", rawData: map[string]interface{}{"verified": true}, want: true},
		{name: "discord unverified", providerType: "discord", rawData: map[string]interface{}{"verified": false}, want: false},
		{name: "gitlab confirmed", providerType: "gitlab", rawData: map[string]interface{}{"confirmed_at": "2024-01-01T00:00:00Z"}, want: true},
		{name: "gitlab unconfirmed", providerType: "gitlab", rawData: map[string]interface{}{"confirmed_at": nil}, want: false},
		{name: "github only returns verified emails", providerType: "github", rawData: nil, want: true},
		{name: "unknown provider without claims", providerType: "custom", rawData: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := oauthEmailVerified(tc.providerType, tc.rawData); got != tc.want {
				t.Fatalf("oauthEmailVerified(%q, %v) = %v, want %v", tc.providerType, tc.rawData, got, tc.want)
			}
		})
	}
}

func TestEmailVerifiedByProvider_ResolvesTypeFromConfiguredName(t *testing.T) {
	svc := setupOAuthTest(t)
	if err := svc.db.Create(&models.OAuthProvider{Type: "github", Name: "company-sso", ClientID: "id", ClientSecret: "secret"}).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	if !svc.EmailVerifiedByProvider("company-sso", nil) {
		t.Fatal("a provider configured with type github must be resolved through its custom name")
	}
	if svc.EmailVerifiedByProvider("unconfigured-name", nil) {
		t.Fatal("an unknown provider without claims must count as unverified")
	}
}

// setupOAuthTestRequiringEmailVerification mirrors production wiring: the OAuth
// service shares the configured AuthService, verification is required, and the
// bootstrap administrator (always auto-verified) already exists.
func setupOAuthTestRequiringEmailVerification(t *testing.T) (*OAuthService, *SystemSettingService) {
	t.Helper()
	authSvc, db, settings := setupAuthWithEmailVerification(t)
	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Bootstrap", LastName: "Admin", Email: "bootstrap-admin@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register bootstrap admin: %v", err)
	}
	svc := NewOAuthService(db, testutil.TestJWTConfig())
	svc.SetAuthService(authSvc)
	return svc, settings
}

func registerThroughOAuthLink(t *testing.T, svc *OAuthService, info *OAuthLinkInfo, registrationEmail string) models.User {
	t.Helper()
	linkToken, err := svc.GenerateLinkToken(info)
	if err != nil {
		t.Fatalf("GenerateLinkToken: %v", err)
	}
	resp, err := svc.LinkOAuthAndRegister(linkToken, dto.OAuthLinkRegisterRequest{
		LinkToken: linkToken, FirstName: "New", LastName: "User", Email: registrationEmail, Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("LinkOAuthAndRegister: %v", err)
	}
	var user models.User
	if err := svc.db.First(&user, "id = ?", resp.User.ID).Error; err != nil {
		t.Fatalf("fetch registered user: %v", err)
	}
	return user
}

// The registration email comes from the request body. It may only skip
// verification when it is the very address the provider vouched for.
func TestLinkOAuthAndRegister_VerifiesOnlyTheProviderConfirmedEmail(t *testing.T) {
	t.Run("different email stays unverified", func(t *testing.T) {
		svc, _ := setupOAuthTestRequiringEmailVerification(t)
		user := registerThroughOAuthLink(t, svc, &OAuthLinkInfo{
			Provider: "google", ProviderUserID: "goo-1", Email: "idp@example.com", EmailVerified: true,
		}, "someone-else@example.com")
		if user.EmailVerifiedAt != nil {
			t.Fatal("an address the provider never vouched for was marked verified")
		}
	})
	t.Run("unverified provider email stays unverified", func(t *testing.T) {
		svc, _ := setupOAuthTestRequiringEmailVerification(t)
		user := registerThroughOAuthLink(t, svc, &OAuthLinkInfo{
			Provider: "discord", ProviderUserID: "dis-1", Email: "idp@example.com", EmailVerified: false,
		}, "idp@example.com")
		if user.EmailVerifiedAt != nil {
			t.Fatal("an email the provider had not verified was marked verified")
		}
	})
	t.Run("provider confirmed email is verified", func(t *testing.T) {
		svc, _ := setupOAuthTestRequiringEmailVerification(t)
		user := registerThroughOAuthLink(t, svc, &OAuthLinkInfo{
			Provider: "google", ProviderUserID: "goo-2", Email: "IdP@Example.com", EmailVerified: true,
		}, "idp@example.com")
		if user.EmailVerifiedAt == nil {
			t.Fatal("the provider confirmed email should not need a second verification")
		}
	})
}

// Registering through an OAuth link is still a registration: the instance
// switch that closes sign-ups has to apply to it.
func TestLinkOAuthAndRegister_HonoursDisabledRegistration(t *testing.T) {
	svc, settings := setupOAuthTestRequiringEmailVerification(t)
	if err := settings.Set("registration.enabled", "false"); err != nil {
		t.Fatalf("disable registration: %v", err)
	}
	linkToken, err := svc.GenerateLinkToken(&OAuthLinkInfo{
		Provider: "github", ProviderUserID: "gh-closed", Email: "outsider@example.com", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("GenerateLinkToken: %v", err)
	}

	_, err = svc.LinkOAuthAndRegister(linkToken, dto.OAuthLinkRegisterRequest{
		LinkToken: linkToken, FirstName: "Out", LastName: "Sider", Email: "outsider@example.com", Password: "password123",
	}, "en")

	if !errors.Is(err, ErrRegistrationDisabled) {
		t.Fatalf("err = %v, want ErrRegistrationDisabled", err)
	}
	var outsiders int64
	if err := svc.db.Model(&models.User{}).Where("email = ?", "outsider@example.com").Count(&outsiders).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if outsiders != 0 {
		t.Fatal("an account was created although registration is disabled")
	}
}
