package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupCallTest(t *testing.T) (*CallService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "call-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewCallService(db), contact.ID, resp.User.ID
}

func TestCreateCall(t *testing.T) {
	svc, contactID, userID := setupCallTest(t)

	calledAt := time.Now()
	call, err := svc.Create(contactID, userID, dto.CreateCallRequest{
		CalledAt:     calledAt,
		Type:         "phone",
		WhoInitiated: "me",
		Description:  "Discussed project",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if call.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, call.ContactID)
	}
	if call.Type != "phone" {
		t.Errorf("Expected type 'phone', got '%s'", call.Type)
	}
	if call.WhoInitiated != "me" {
		t.Errorf("Expected who_initiated 'me', got '%s'", call.WhoInitiated)
	}
	if call.Description != "Discussed project" {
		t.Errorf("Expected description 'Discussed project', got '%s'", call.Description)
	}
	if !call.Answered {
		t.Error("Expected call to be answered by default")
	}
	if call.ID == 0 {
		t.Error("Expected call ID to be non-zero")
	}
}

func TestListCalls(t *testing.T) {
	svc, contactID, userID := setupCallTest(t)

	calledAt := time.Now()
	_, err := svc.Create(contactID, userID, dto.CreateCallRequest{CalledAt: calledAt, Type: "phone", WhoInitiated: "me"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, userID, dto.CreateCallRequest{CalledAt: calledAt, Type: "video", WhoInitiated: "them"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	calls, meta, err := svc.List(contactID, 1, 15)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(calls) != 2 {
		t.Errorf("Expected 2 calls, got %d", len(calls))
	}
	if meta.Total != 2 {
		t.Errorf("Expected total 2, got %d", meta.Total)
	}
}

func TestUpdateCall(t *testing.T) {
	svc, contactID, userID := setupCallTest(t)

	calledAt := time.Now()
	created, err := svc.Create(contactID, userID, dto.CreateCallRequest{
		CalledAt:     calledAt,
		Type:         "phone",
		WhoInitiated: "me",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	newCalledAt := time.Now()
	answered := false
	updated, err := svc.Update(created.ID, contactID, dto.UpdateCallRequest{
		CalledAt:     newCalledAt,
		Type:         "video",
		WhoInitiated: "them",
		Description:  "Updated description",
		Answered:     &answered,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Type != "video" {
		t.Errorf("Expected type 'video', got '%s'", updated.Type)
	}
	if updated.WhoInitiated != "them" {
		t.Errorf("Expected who_initiated 'them', got '%s'", updated.WhoInitiated)
	}
	if updated.Answered {
		t.Error("Expected call to not be answered after update")
	}
}

func TestDeleteCall(t *testing.T) {
	svc, contactID, userID := setupCallTest(t)

	calledAt := time.Now()
	created, err := svc.Create(contactID, userID, dto.CreateCallRequest{
		CalledAt:     calledAt,
		Type:         "phone",
		WhoInitiated: "me",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	calls, _, err := svc.List(contactID, 1, 15)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("Expected 0 calls after delete, got %d", len(calls))
	}
}

func TestCallNotFound(t *testing.T) {
	svc, contactID, _ := setupCallTest(t)

	calledAt := time.Now()
	_, err := svc.Update(9999, contactID, dto.UpdateCallRequest{
		CalledAt:     calledAt,
		Type:         "phone",
		WhoInitiated: "me",
	})
	if err != ErrCallNotFound {
		t.Errorf("Expected ErrCallNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID)
	if err != ErrCallNotFound {
		t.Errorf("Expected ErrCallNotFound, got %v", err)
	}
}
