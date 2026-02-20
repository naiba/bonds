package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestRegister(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	req := dto.RegisterRequest{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Password:  "password123",
	}

	resp, err := svc.Register(req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected token to be non-empty")
	}
	if resp.User.Email != "john@example.com" {
		t.Errorf("Expected email john@example.com, got %s", resp.User.Email)
	}
	if resp.User.FirstName != "John" {
		t.Errorf("Expected first name John, got %s", resp.User.FirstName)
	}
	if resp.User.ID == "" {
		t.Error("Expected user ID to be non-empty")
	}
	if resp.User.AccountID == "" {
		t.Error("Expected account ID to be non-empty")
	}
}

func TestRegisterSeedsDefaultData(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	resp, err := svc.Register(dto.RegisterRequest{
		FirstName: "Seed",
		LastName:  "Test",
		Email:     "seed@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	accountID := resp.User.AccountID

	tests := []struct {
		name     string
		model    interface{}
		minCount int64
	}{
		{"genders", &[]models.Gender{}, 3},
		{"pronouns", &[]models.Pronoun{}, 7},
		{"address_types", &[]models.AddressType{}, 5},
		{"pet_categories", &[]models.PetCategory{}, 10},
		{"contact_info_types", &[]models.ContactInformationType{}, 12},
		{"relationship_group_types", &[]models.RelationshipGroupType{}, 4},
		{"call_reason_types", &[]models.CallReasonType{}, 2},
		{"religions", &[]models.Religion{}, 9},
		{"group_types", &[]models.GroupType{}, 5},
		{"emotions", &[]models.Emotion{}, 3},
		{"gift_occasions", &[]models.GiftOccasion{}, 5},
		{"gift_states", &[]models.GiftState{}, 5},
		{"post_templates", &[]models.PostTemplate{}, 2},
	}

	for _, tt := range tests {
		var count int64
		if err := db.Where("account_id = ?", accountID).Find(tt.model).Count(&count).Error; err != nil {
			t.Errorf("%s: query failed: %v", tt.name, err)
			continue
		}
		if count < tt.minCount {
			t.Errorf("%s: expected at least %d records, got %d", tt.name, tt.minCount, count)
		}
	}

	var emailType models.ContactInformationType
	if err := db.Where("account_id = ? AND type = ?", accountID, "email").First(&emailType).Error; err != nil {
		t.Fatalf("email contact info type not found: %v", err)
	}
	if emailType.CanBeDeleted {
		t.Error("email contact info type should have can_be_deleted=false")
	}

	var phoneType models.ContactInformationType
	if err := db.Where("account_id = ? AND type = ?", accountID, "phone").First(&phoneType).Error; err != nil {
		t.Fatalf("phone contact info type not found: %v", err)
	}
	if phoneType.CanBeDeleted {
		t.Error("phone contact info type should have can_be_deleted=false")
	}

	var loveGroup models.RelationshipGroupType
	if err := db.Where("account_id = ? AND name = ?", accountID, "Love").First(&loveGroup).Error; err != nil {
		t.Fatalf("Love relationship group not found: %v", err)
	}
	if loveGroup.CanBeDeleted {
		t.Error("Love relationship group should have can_be_deleted=false")
	}

	var relTypes []models.RelationshipType
	if err := db.Where("relationship_group_type_id = ?", loveGroup.ID).Find(&relTypes).Error; err != nil {
		t.Fatalf("relationship types query failed: %v", err)
	}
	if len(relTypes) != 6 {
		t.Errorf("Love group: expected 6 relationship types, got %d", len(relTypes))
	}

	var callReasons []models.CallReason
	if err := db.Find(&callReasons).Error; err != nil {
		t.Fatalf("call reasons query failed: %v", err)
	}
	if len(callReasons) != 7 {
		t.Errorf("expected 7 call reasons, got %d", len(callReasons))
	}

	var channel models.UserNotificationChannel
	if err := db.Where("user_id = ? AND type = ?", resp.User.ID, "email").First(&channel).Error; err != nil {
		t.Fatalf("notification channel not found: %v", err)
	}
	if channel.Content != "seed@example.com" {
		t.Errorf("expected channel content 'seed@example.com', got '%s'", channel.Content)
	}
	if !channel.Active {
		t.Error("notification channel should have active=true")
	}
	if channel.VerifiedAt == nil {
		t.Error("notification channel should have verified_at set")
	}

	var tmpl models.Template
	if err := db.Where("account_id = ? AND name = ?", accountID, "Default template").First(&tmpl).Error; err != nil {
		t.Fatalf("default template not found: %v", err)
	}
	if tmpl.CanBeDeleted {
		t.Error("default template should have can_be_deleted=false")
	}
	var pages []models.TemplatePage
	if err := db.Where("template_id = ?", tmpl.ID).Find(&pages).Error; err != nil {
		t.Fatalf("template pages query failed: %v", err)
	}
	if len(pages) != 5 {
		t.Errorf("expected 5 template pages, got %d", len(pages))
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	req := dto.RegisterRequest{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Password:  "password123",
	}

	_, err := svc.Register(req)
	if err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	_, err = svc.Register(req)
	if err != ErrEmailExists {
		t.Errorf("Expected ErrEmailExists, got %v", err)
	}
}

func TestLoginScenarios(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	_, err := svc.Register(dto.RegisterRequest{
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tests := []struct {
		name      string
		email     string
		password  string
		wantErr   error
		wantToken bool
	}{
		{"valid credentials", "jane@example.com", "password123", nil, true},
		{"invalid password", "jane@example.com", "wrongpassword", ErrInvalidCredentials, false},
		{"nonexistent user", "nobody@example.com", "password123", ErrInvalidCredentials, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.Login(dto.LoginRequest{Email: tt.email, Password: tt.password})
			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantToken && (resp == nil || resp.Token == "") {
				t.Error("expected non-empty token")
			}
		})
	}
}

func TestRegisterFirstUserIsInstanceAdmin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	first, err := svc.Register(dto.RegisterRequest{
		FirstName: "First",
		LastName:  "User",
		Email:     "first@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register first user failed: %v", err)
	}
	if !first.User.IsInstanceAdministrator {
		t.Error("expected first user to be instance administrator")
	}

	second, err := svc.Register(dto.RegisterRequest{
		FirstName: "Second",
		LastName:  "User",
		Email:     "second@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register second user failed: %v", err)
	}
	if second.User.IsInstanceAdministrator {
		t.Error("expected second user to NOT be instance administrator")
	}
}

func TestLoginDisabledUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	resp, err := svc.Register(dto.RegisterRequest{
		FirstName: "Disabled",
		LastName:  "User",
		Email:     "disabled@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	db.Model(&models.User{}).Where("id = ?", resp.User.ID).Update("disabled", true)

	_, err = svc.Login(dto.LoginRequest{Email: "disabled@example.com", Password: "password123"})
	if err != ErrUserDisabled {
		t.Errorf("expected ErrUserDisabled, got %v", err)
	}
}

func TestRefreshTokenDisabledUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)

	resp, err := svc.Register(dto.RegisterRequest{
		FirstName: "Refresh",
		LastName:  "Disabled",
		Email:     "refresh-disabled@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	db.Model(&models.User{}).Where("id = ?", resp.User.ID).Update("disabled", true)

	_, err = svc.RefreshToken(&middleware.JWTClaims{
		UserID:    resp.User.ID,
		AccountID: resp.User.AccountID,
		Email:     resp.User.Email,
	})
	if err != ErrUserDisabled {
		t.Errorf("expected ErrUserDisabled, got %v", err)
	}
}
