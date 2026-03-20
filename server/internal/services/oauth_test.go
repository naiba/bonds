package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupOAuthTest(t *testing.T) *OAuthService {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	return NewOAuthService(db, cfg, "http://localhost:8080")
}

func TestFindOrCreateUserNew_ReturnsNotLinked(t *testing.T) {
	svc := setupOAuthTest(t)

	resp, linkInfo, err := svc.FindOrCreateUser("github", "gh-12345", "newuser@example.com", "John Doe", "en")
	if !errors.Is(err, ErrOAuthAccountNotLinked) {
		t.Fatalf("Expected ErrOAuthAccountNotLinked, got %v", err)
	}
	if resp != nil {
		t.Error("Expected nil auth response for unlinked account")
	}
	if linkInfo == nil {
		t.Fatal("Expected non-nil link info")
	}
	if linkInfo.Provider != "github" {
		t.Errorf("Expected provider github, got %s", linkInfo.Provider)
	}
	if linkInfo.ProviderUserID != "gh-12345" {
		t.Errorf("Expected provider user ID gh-12345, got %s", linkInfo.ProviderUserID)
	}
	if linkInfo.Email != "newuser@example.com" {
		t.Errorf("Expected email newuser@example.com, got %s", linkInfo.Email)
	}
	if linkInfo.Name != "John Doe" {
		t.Errorf("Expected name John Doe, got %s", linkInfo.Name)
	}

	var tokenCount int64
	svc.db.Model(&models.UserToken{}).Count(&tokenCount)
	if tokenCount != 0 {
		t.Errorf("Expected no UserToken created, got %d", tokenCount)
	}
}

func TestFindOrCreateUserExisting(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	regResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Existing",
		LastName:  "User",
		Email:     "existing@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	resp1, _, err := svc.FindOrCreateUser("github", "gh-12345", "existing@example.com", "Existing User", "en")
	if err != nil {
		t.Fatalf("First FindOrCreateUser failed: %v", err)
	}
	if resp1.User.ID != regResp.User.ID {
		t.Errorf("Expected user ID %s, got %s", regResp.User.ID, resp1.User.ID)
	}

	resp2, _, err := svc.FindOrCreateUser("github", "gh-12345", "existing@example.com", "Existing User", "en")
	if err != nil {
		t.Fatalf("Second FindOrCreateUser failed: %v", err)
	}
	if resp2.User.ID != resp1.User.ID {
		t.Errorf("Expected same user ID %s, got %s", resp1.User.ID, resp2.User.ID)
	}
}

func TestFindOrCreateUserLinkEmail(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	regResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Link",
		LastName:  "User",
		Email:     "link@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	existingUserID := regResp.User.ID

	resp, _, err := svc.FindOrCreateUser("google", "goo-99999", "link@example.com", "Link User", "en")
	if err != nil {
		t.Fatalf("FindOrCreateUser failed: %v", err)
	}

	if resp.User.ID != existingUserID {
		t.Errorf("Expected to link to existing user %s, got %s", existingUserID, resp.User.ID)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "google", "goo-99999").First(&token).Error; err != nil {
		t.Fatalf("Expected UserToken to be created: %v", err)
	}
	if token.UserID != existingUserID {
		t.Errorf("Expected token linked to user %s, got %s", existingUserID, token.UserID)
	}
}

func TestGenerateAndParseLinkToken(t *testing.T) {
	svc := setupOAuthTest(t)

	info := &OAuthLinkInfo{
		Provider:       "github",
		ProviderUserID: "gh-12345",
		Email:          "test@example.com",
		Name:           "Test User",
	}

	linkToken, err := svc.GenerateLinkToken(info)
	if err != nil {
		t.Fatalf("GenerateLinkToken failed: %v", err)
	}
	if linkToken == "" {
		t.Fatal("Expected non-empty link token")
	}

	parsed, err := svc.ParseLinkToken(linkToken)
	if err != nil {
		t.Fatalf("ParseLinkToken failed: %v", err)
	}
	if parsed.Provider != info.Provider {
		t.Errorf("Expected provider %s, got %s", info.Provider, parsed.Provider)
	}
	if parsed.ProviderUserID != info.ProviderUserID {
		t.Errorf("Expected provider user ID %s, got %s", info.ProviderUserID, parsed.ProviderUserID)
	}
	if parsed.Email != info.Email {
		t.Errorf("Expected email %s, got %s", info.Email, parsed.Email)
	}
	if parsed.Name != info.Name {
		t.Errorf("Expected name %s, got %s", info.Name, parsed.Name)
	}
}

func TestParseLinkToken_InvalidToken(t *testing.T) {
	svc := setupOAuthTest(t)

	_, err := svc.ParseLinkToken("invalid-token")
	if !errors.Is(err, ErrOAuthLinkTokenInvalid) {
		t.Errorf("Expected ErrOAuthLinkTokenInvalid, got %v", err)
	}
}

func TestLinkOAuthToUser(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	regResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Link",
		LastName:  "Target",
		Email:     "target@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	info := &OAuthLinkInfo{
		Provider:       "github",
		ProviderUserID: "gh-link-test",
		Email:          "oauth-user@github.com",
		Name:           "OAuth User",
	}
	linkToken, err := svc.GenerateLinkToken(info)
	if err != nil {
		t.Fatalf("GenerateLinkToken failed: %v", err)
	}

	resp, err := svc.LinkOAuthToUser(linkToken, regResp.User.ID)
	if err != nil {
		t.Fatalf("LinkOAuthToUser failed: %v", err)
	}
	if resp.User.ID != regResp.User.ID {
		t.Errorf("Expected user ID %s, got %s", regResp.User.ID, resp.User.ID)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "github", "gh-link-test").First(&token).Error; err != nil {
		t.Fatalf("Expected UserToken to be created: %v", err)
	}
	if token.UserID != regResp.User.ID {
		t.Errorf("Expected token user ID %s, got %s", regResp.User.ID, token.UserID)
	}
}

func TestLinkOAuthToUser_InvalidToken(t *testing.T) {
	svc := setupOAuthTest(t)

	_, err := svc.LinkOAuthToUser("bad-token", "some-user-id")
	if !errors.Is(err, ErrOAuthLinkTokenInvalid) {
		t.Errorf("Expected ErrOAuthLinkTokenInvalid, got %v", err)
	}
}

func TestLinkOAuthToUser_AlreadyLinkedToOtherUser(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	reg1, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "User",
		LastName:  "One",
		Email:     "user1@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register user1 failed: %v", err)
	}

	reg2, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "User",
		LastName:  "Two",
		Email:     "user2@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register user2 failed: %v", err)
	}

	info := &OAuthLinkInfo{
		Provider:       "github",
		ProviderUserID: "gh-shared",
		Email:          "shared@github.com",
		Name:           "Shared",
	}

	linkToken1, _ := svc.GenerateLinkToken(info)
	_, err = svc.LinkOAuthToUser(linkToken1, reg1.User.ID)
	if err != nil {
		t.Fatalf("First LinkOAuthToUser failed: %v", err)
	}

	linkToken2, _ := svc.GenerateLinkToken(info)
	_, err = svc.LinkOAuthToUser(linkToken2, reg2.User.ID)
	if !errors.Is(err, ErrOAuthAlreadyLinked) {
		t.Errorf("Expected ErrOAuthAlreadyLinked, got %v", err)
	}
}

func TestLinkOAuthAndRegister(t *testing.T) {
	svc := setupOAuthTest(t)

	info := &OAuthLinkInfo{
		Provider:       "google",
		ProviderUserID: "goo-register",
		Email:          "oauth-new@google.com",
		Name:           "New User",
	}
	linkToken, err := svc.GenerateLinkToken(info)
	if err != nil {
		t.Fatalf("GenerateLinkToken failed: %v", err)
	}

	req := dto.OAuthLinkRegisterRequest{
		LinkToken: linkToken,
		FirstName: "New",
		LastName:  "User",
		Email:     "newaccount@example.com",
		Password:  "password123",
	}

	resp, err := svc.LinkOAuthAndRegister(linkToken, req, "en")
	if err != nil {
		t.Fatalf("LinkOAuthAndRegister failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected non-empty token")
	}
	if resp.User.Email != "newaccount@example.com" {
		t.Errorf("Expected email newaccount@example.com, got %s", resp.User.Email)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "google", "goo-register").First(&token).Error; err != nil {
		t.Fatalf("Expected UserToken to be created: %v", err)
	}
	if token.UserID != resp.User.ID {
		t.Errorf("Expected token user ID %s, got %s", resp.User.ID, token.UserID)
	}
}

func TestLinkOAuthAndRegister_DuplicateEmail(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	_, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Existing",
		LastName:  "User",
		Email:     "taken@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	info := &OAuthLinkInfo{
		Provider:       "google",
		ProviderUserID: "goo-dup",
		Email:          "oauth@google.com",
		Name:           "Dup User",
	}
	linkToken, _ := svc.GenerateLinkToken(info)

	req := dto.OAuthLinkRegisterRequest{
		LinkToken: linkToken,
		FirstName: "Dup",
		LastName:  "User",
		Email:     "taken@example.com",
		Password:  "password123",
	}

	_, err = svc.LinkOAuthAndRegister(linkToken, req, "en")
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("Expected ErrEmailExists, got %v", err)
	}
}

func TestSaveTokenCreate(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	regResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Save",
		LastName:  "User",
		Email:     "save@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = svc.SaveToken(regResp.User.ID, "github", "gh-save", "access-token-123", "refresh-token-456", 3600)
	if err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "github", "gh-save").First(&token).Error; err != nil {
		t.Fatalf("Token not found: %v", err)
	}
	if token.Token != "access-token-123" {
		t.Errorf("Expected access token 'access-token-123', got '%s'", token.Token)
	}
}

func TestSaveTokenUpdate(t *testing.T) {
	svc := setupOAuthTest(t)

	authSvc := NewAuthService(svc.db, svc.jwt)
	regResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Update",
		LastName:  "User",
		Email:     "update@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = svc.SaveToken(regResp.User.ID, "github", "gh-update", "token-v1", "", 0)
	if err != nil {
		t.Fatalf("First SaveToken failed: %v", err)
	}

	err = svc.SaveToken(regResp.User.ID, "github", "gh-update", "token-v2", "refresh-v2", 7200)
	if err != nil {
		t.Fatalf("Second SaveToken failed: %v", err)
	}

	var token models.UserToken
	if err := svc.db.Where("driver = ? AND driver_id = ?", "github", "gh-update").First(&token).Error; err != nil {
		t.Fatalf("Token not found: %v", err)
	}
	if token.Token != "token-v2" {
		t.Errorf("Expected access token 'token-v2', got '%s'", token.Token)
	}
}
