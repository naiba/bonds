package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupLifeEventTest(t *testing.T) (*LifeEventService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "life-events-test@example.com",
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

	return NewLifeEventService(db), contact.ID, vault.ID
}

func TestCreateTimelineEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	startedAt := time.Now()
	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: startedAt,
		Label:     "My Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}
	if te.VaultID != vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", vaultID, te.VaultID)
	}
	if te.Label != "My Timeline" {
		t.Errorf("Expected label 'My Timeline', got '%s'", te.Label)
	}
	if te.ID == 0 {
		t.Error("Expected timeline event ID to be non-zero")
	}
}

func TestListTimelineEvents(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	_, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Event 1",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}
	_, err = svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Event 2",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	events, meta, err := svc.ListTimelineEvents(contactID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("Expected 2 timeline events, got %d", len(events))
	}
	if meta.Total != 2 {
		t.Errorf("Expected total 2, got %d", meta.Total)
	}
}

func TestAddLifeEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	happenedAt := time.Now()
	le, err := svc.AddLifeEvent(te.ID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      happenedAt,
		Summary:         "Got promoted",
		Description:     "Big promotion at work",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}
	if le.TimelineEventID != te.ID {
		t.Errorf("Expected timeline_event_id %d, got %d", te.ID, le.TimelineEventID)
	}
	if le.Summary != "Got promoted" {
		t.Errorf("Expected summary 'Got promoted', got '%s'", le.Summary)
	}
	if le.Description != "Big promotion at work" {
		t.Errorf("Expected description 'Big promotion at work', got '%s'", le.Description)
	}
	if le.ID == 0 {
		t.Error("Expected life event ID to be non-zero")
	}
}

func TestUpdateLifeEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	le, err := svc.AddLifeEvent(te.ID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Original",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	updated, err := svc.UpdateLifeEvent(te.ID, le.ID, dto.UpdateLifeEventRequest{
		Summary:     "Updated summary",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("UpdateLifeEvent failed: %v", err)
	}
	if updated.Summary != "Updated summary" {
		t.Errorf("Expected summary 'Updated summary', got '%s'", updated.Summary)
	}
	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
	}
}

func TestDeleteLifeEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	le, err := svc.AddLifeEvent(te.ID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "To delete",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	if err := svc.DeleteLifeEvent(te.ID, le.ID); err != nil {
		t.Fatalf("DeleteLifeEvent failed: %v", err)
	}

	_, err = svc.UpdateLifeEvent(te.ID, le.ID, dto.UpdateLifeEventRequest{Summary: "nope"})
	if err != ErrLifeEventNotFound {
		t.Errorf("Expected ErrLifeEventNotFound, got %v", err)
	}
}

func TestDeleteTimelineEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "To delete",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	if err := svc.DeleteTimelineEvent(te.ID); err != nil {
		t.Fatalf("DeleteTimelineEvent failed: %v", err)
	}

	events, _, err := svc.ListTimelineEvents(contactID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 timeline events after delete, got %d", len(events))
	}
}

func TestTimelineEventNotFound(t *testing.T) {
	svc, _, _ := setupLifeEventTest(t)

	err := svc.DeleteTimelineEvent(9999)
	if err != ErrTimelineEventNotFound {
		t.Errorf("Expected ErrTimelineEventNotFound, got %v", err)
	}

	_, err = svc.AddLifeEvent(9999, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "nope",
	})
	if err != ErrTimelineEventNotFound {
		t.Errorf("Expected ErrTimelineEventNotFound, got %v", err)
	}
}
