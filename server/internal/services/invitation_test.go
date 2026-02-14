package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupInvitationTest(t *testing.T) (*InvitationService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "invite-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	mailer := &NoopMailer{}
	svc := NewInvitationService(db, mailer, "http://localhost:8080")
	return svc, resp.User.AccountID, resp.User.ID
}

func TestCreateInvitation(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	inv, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
		Email:      "invited@example.com",
		Permission: 300,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if inv.Email != "invited@example.com" {
		t.Errorf("Expected email 'invited@example.com', got '%s'", inv.Email)
	}
	if inv.Permission != 300 {
		t.Errorf("Expected permission 300, got %d", inv.Permission)
	}
	if inv.ID == 0 {
		t.Error("Expected non-zero ID")
	}
	if inv.AcceptedAt != nil {
		t.Error("Expected AcceptedAt to be nil")
	}
	if inv.ExpiresAt.Before(time.Now()) {
		t.Error("Expected ExpiresAt to be in the future")
	}
}

func TestAcceptInvitation(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	inv, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
		Email:      "accept@example.com",
		Permission: 200,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var invitation models.Invitation
	if err := svc.db.First(&invitation, inv.ID).Error; err != nil {
		t.Fatalf("Failed to load invitation: %v", err)
	}

	accepted, err := svc.Accept(dto.AcceptInvitationRequest{
		Token:     invitation.Token,
		FirstName: "New",
		LastName:  "User",
		Password:  "newpassword123",
	})
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	if accepted.AcceptedAt == nil {
		t.Error("Expected AcceptedAt to be set")
	}

	var user models.User
	if err := svc.db.Where("email = ?", "accept@example.com").First(&user).Error; err != nil {
		t.Fatalf("Expected user to be created: %v", err)
	}
	if user.AccountID != accountID {
		t.Errorf("Expected account_id '%s', got '%s'", accountID, user.AccountID)
	}
	if user.InvitationCode == nil || *user.InvitationCode != invitation.Token {
		t.Error("Expected InvitationCode to match token")
	}
	if user.InvitationAcceptedAt == nil {
		t.Error("Expected InvitationAcceptedAt to be set")
	}
}

func TestAcceptExpiredInvitation(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	inv, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
		Email:      "expired@example.com",
		Permission: 300,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	svc.db.Model(&models.Invitation{}).Where("id = ?", inv.ID).
		Update("expires_at", time.Now().Add(-1*time.Hour))

	var invitation models.Invitation
	svc.db.First(&invitation, inv.ID)

	_, err = svc.Accept(dto.AcceptInvitationRequest{
		Token:     invitation.Token,
		FirstName: "Late",
		Password:  "password123",
	})
	if err != ErrInvitationExpired {
		t.Fatalf("Expected ErrInvitationExpired, got: %v", err)
	}
}

func TestAcceptAlreadyAccepted(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	inv, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
		Email:      "double@example.com",
		Permission: 300,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var invitation models.Invitation
	svc.db.First(&invitation, inv.ID)

	_, err = svc.Accept(dto.AcceptInvitationRequest{
		Token:     invitation.Token,
		FirstName: "First",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("First accept failed: %v", err)
	}

	_, err = svc.Accept(dto.AcceptInvitationRequest{
		Token:     invitation.Token,
		FirstName: "Second",
		Password:  "password123",
	})
	if err != ErrInvitationNotFound {
		t.Fatalf("Expected ErrInvitationNotFound, got: %v", err)
	}
}

func TestListInvitations(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	for i := 0; i < 3; i++ {
		_, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
			Email:      fmt.Sprintf("list%d@example.com", i),
			Permission: 300,
		})
		if err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	list, err := svc.List(accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 invitations, got %d", len(list))
	}
}

func TestDeleteInvitation(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	inv, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
		Email:      "delete@example.com",
		Permission: 300,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(inv.ID, accountID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, err := svc.List(accountID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected 0 invitations after delete, got %d", len(list))
	}
}

func TestCreateInvitationDuplicateEmail(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	_, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
		Email:      "invite-test@example.com",
		Permission: 300,
	})
	if err != ErrUserAlreadyExists {
		t.Fatalf("Expected ErrUserAlreadyExists, got: %v", err)
	}
}
