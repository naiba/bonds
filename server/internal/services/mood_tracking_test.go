package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupMoodTrackingTest(t *testing.T) (*MoodTrackingService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "mood-tracking-test@example.com",
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

	return NewMoodTrackingService(db), contact.ID, vault.ID
}

func TestCreateMoodTrackingEvent(t *testing.T) {
	svc, contactID, vaultID := setupMoodTrackingTest(t)

	hoursSlept := 8
	ratedAt := time.Now()
	event, err := svc.Create(contactID, vaultID, dto.CreateMoodTrackingEventRequest{
		MoodTrackingParameterID: 1,
		RatedAt:                 ratedAt,
		Note:                    "Feeling great",
		NumberOfHoursSlept:      &hoursSlept,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if event.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, event.ContactID)
	}
	if event.MoodTrackingParameterID != 1 {
		t.Errorf("Expected mood_tracking_parameter_id 1, got %d", event.MoodTrackingParameterID)
	}
	if event.Note != "Feeling great" {
		t.Errorf("Expected note 'Feeling great', got '%s'", event.Note)
	}
	if event.NumberOfHoursSlept == nil || *event.NumberOfHoursSlept != 8 {
		t.Errorf("Expected number_of_hours_slept 8, got %v", event.NumberOfHoursSlept)
	}
	if event.ID == 0 {
		t.Error("Expected event ID to be non-zero")
	}
}

func TestListMoodTrackingEvents(t *testing.T) {
	svc, contactID, vaultID := setupMoodTrackingTest(t)

	ratedAt := time.Now()
	_, err := svc.Create(contactID, vaultID, dto.CreateMoodTrackingEventRequest{
		MoodTrackingParameterID: 1,
		RatedAt:                 ratedAt,
		Note:                    "Event 1",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreateMoodTrackingEventRequest{
		MoodTrackingParameterID: 1,
		RatedAt:                 ratedAt,
		Note:                    "Event 2",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	events, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestCreateMoodTrackingEventWithoutNote(t *testing.T) {
	svc, contactID, vaultID := setupMoodTrackingTest(t)

	ratedAt := time.Now()
	event, err := svc.Create(contactID, vaultID, dto.CreateMoodTrackingEventRequest{
		MoodTrackingParameterID: 2,
		RatedAt:                 ratedAt,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if event.Note != "" {
		t.Errorf("Expected empty note, got '%s'", event.Note)
	}
	if event.NumberOfHoursSlept != nil {
		t.Errorf("Expected nil number_of_hours_slept, got %v", event.NumberOfHoursSlept)
	}
}

func TestCreateMoodTrackingEventFieldValues(t *testing.T) {
	svc, contactID, vaultID := setupMoodTrackingTest(t)

	hoursSlept := 6
	ratedAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	event, err := svc.Create(contactID, vaultID, dto.CreateMoodTrackingEventRequest{
		MoodTrackingParameterID: 3,
		RatedAt:                 ratedAt,
		Note:                    "Tired",
		NumberOfHoursSlept:      &hoursSlept,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if event.MoodTrackingParameterID != 3 {
		t.Errorf("Expected mood_tracking_parameter_id 3, got %d", event.MoodTrackingParameterID)
	}
	if event.NumberOfHoursSlept == nil || *event.NumberOfHoursSlept != 6 {
		t.Errorf("Expected number_of_hours_slept 6, got %v", event.NumberOfHoursSlept)
	}
}

func TestListMoodTrackingEventsEmpty(t *testing.T) {
	svc, contactID, vaultID := setupMoodTrackingTest(t)

	events, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}
