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
	}, "en")
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

	list, meta, err := svc.List(accountID, 0, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 invitations, got %d", len(list))
	}
	if meta.Total != 3 {
		t.Errorf("Expected meta.Total=3, got %d", meta.Total)
	}
}

func TestListInvitations_Pagination(t *testing.T) {
	svc, accountID, userID := setupInvitationTest(t)

	for i := 0; i < 5; i++ {
		_, err := svc.Create(accountID, userID, dto.CreateInvitationRequest{
			Email:      fmt.Sprintf("page%d@example.com", i),
			Permission: 300,
		})
		if err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	page1, meta1, err := svc.List(accountID, 1, 2)
	if err != nil {
		t.Fatalf("List page1 failed: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("Expected 2 invitations on page 1, got %d", len(page1))
	}
	if meta1.Total != 5 {
		t.Errorf("Expected total=5, got %d", meta1.Total)
	}
	if meta1.TotalPages != 3 {
		t.Errorf("Expected total_pages=3, got %d", meta1.TotalPages)
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

	list, _, err := svc.List(accountID, 0, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected 0 invitations after delete, got %d", len(list))
	}
}

func TestAcceptInvitation_VaultAccessGranted(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Owner",
		LastName:  "User",
		Email:     "vault-owner@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	v1, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault A"}, "en")
	if err != nil {
		t.Fatalf("CreateVault A failed: %v", err)
	}
	v2, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault B"}, "en")
	if err != nil {
		t.Fatalf("CreateVault B failed: %v", err)
	}

	mailer := &NoopMailer{}
	invSvc := NewInvitationService(db, mailer, "http://localhost:8080")

	inv, err := invSvc.Create(resp.User.AccountID, resp.User.ID, dto.CreateInvitationRequest{
		Email:      "invited-vault@example.com",
		Permission: models.PermissionEditor,
	})
	if err != nil {
		t.Fatalf("Create invitation failed: %v", err)
	}

	var invitation models.Invitation
	if err := db.First(&invitation, inv.ID).Error; err != nil {
		t.Fatalf("Load invitation failed: %v", err)
	}

	_, err = invSvc.Accept(dto.AcceptInvitationRequest{
		Token:     invitation.Token,
		FirstName: "Invited",
		LastName:  "Person",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}

	var newUser models.User
	if err := db.Where("email = ?", "invited-vault@example.com").First(&newUser).Error; err != nil {
		t.Fatalf("New user not found: %v", err)
	}

	var userVaults []models.UserVault
	if err := db.Where("user_id = ?", newUser.ID).Find(&userVaults).Error; err != nil {
		t.Fatalf("UserVault query failed: %v", err)
	}
	if len(userVaults) != 2 {
		t.Fatalf("Expected 2 UserVault entries, got %d", len(userVaults))
	}

	vaultIDSet := map[string]bool{v1.ID: false, v2.ID: false}
	for _, uv := range userVaults {
		if _, ok := vaultIDSet[uv.VaultID]; !ok {
			t.Errorf("Unexpected vault ID: %s", uv.VaultID)
		}
		vaultIDSet[uv.VaultID] = true

		if uv.Permission != models.PermissionEditor {
			t.Errorf("Expected PermissionEditor (%d), got %d", models.PermissionEditor, uv.Permission)
		}
		if uv.ContactID == "" {
			t.Errorf("Expected ContactID to be set for vault %s", uv.VaultID)
		}

		var contact models.Contact
		if err := db.First(&contact, "id = ?", uv.ContactID).Error; err != nil {
			t.Fatalf("Self-contact not found for vault %s: %v", uv.VaultID, err)
		}
		if contact.CanBeDeleted {
			t.Error("Self-contact should have CanBeDeleted=false")
		}
		if contact.Listed {
			t.Error("Self-contact should have Listed=false")
		}
	}
	for vid, found := range vaultIDSet {
		if !found {
			t.Errorf("User was not added to vault %s", vid)
		}
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
