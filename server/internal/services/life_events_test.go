package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
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

func createLifeEventTestContact(t *testing.T, svc *LifeEventService, vaultID string, firstName string) string {
	t.Helper()
	contact := models.Contact{
		VaultID:      vaultID,
		FirstName:    strPtrOrNil(firstName),
		CanBeDeleted: true,
		Listed:       true,
	}
	if err := svc.db.Create(&contact).Error; err != nil {
		t.Fatalf("CreateContact %s failed: %v", firstName, err)
	}
	return contact.ID
}

func participantIDs(refs []dto.TaskContactRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}
	return ids
}

func assertParticipantIDs(t *testing.T, refs []dto.TaskContactRef, expected ...string) {
	t.Helper()
	seen := make(map[string]int, len(refs))
	for _, ref := range refs {
		seen[ref.ID]++
	}
	if len(seen) != len(expected) {
		t.Fatalf("expected participant ids %v, got refs=%v", expected, refs)
	}
	for _, id := range expected {
		if seen[id] != 1 {
			t.Fatalf("expected participant %s exactly once, got ids=%v refs=%v", id, participantIDs(refs), refs)
		}
	}
}

func TestTimelineEventParticipantsPersistDedupeAndListForParticipants(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	participantID := createLifeEventTestContact(t, svc, vaultID, "Participant")

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt:    time.Now(),
		Label:        "Shared Timeline",
		Participants: []string{participantID, participantID, contactID},
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}
	assertParticipantIDs(t, te.Participants, contactID, participantID)

	ownerEvents, _, err := svc.ListTimelineEvents(contactID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents owner failed: %v", err)
	}
	if len(ownerEvents) != 1 {
		t.Fatalf("expected owner to see 1 timeline, got %d", len(ownerEvents))
	}
	assertParticipantIDs(t, ownerEvents[0].Participants, contactID, participantID)

	participantEvents, _, err := svc.ListTimelineEvents(participantID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents participant failed: %v", err)
	}
	if len(participantEvents) != 1 || participantEvents[0].ID != te.ID {
		t.Fatalf("expected participant to see timeline %d, got %+v", te.ID, participantEvents)
	}
}

func TestTimelineEventParticipantsRejectDuplicatePivotRows(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Unique timeline participant",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	duplicate := models.TimelineEventParticipant{TimelineEventID: te.ID, ContactID: contactID}
	if err := svc.db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected duplicate timeline participant pivot to fail")
	}
}

func TestLifeEventParticipantsRejectDuplicatePivotRows(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Unique life participant timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}
	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Unique life participant",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	duplicate := models.LifeEventParticipant{LifeEventID: le.ID, ContactID: contactID}
	if err := svc.db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected duplicate life event participant pivot to fail")
	}
}

func TestUpdateLifeEventParticipantsPersistUpdateReplaceAndKeepTimelineParticipants(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	timelineParticipantID := createLifeEventTestContact(t, svc, vaultID, "TimelineParticipant")
	lifeParticipantID := createLifeEventTestContact(t, svc, vaultID, "LifeParticipant")
	replacementID := createLifeEventTestContact(t, svc, vaultID, "Replacement")

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt:    time.Now(),
		Label:        "Timeline",
		Participants: []string{timelineParticipantID},
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Shared life event",
		Participants:    []string{lifeParticipantID, lifeParticipantID},
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}
	assertParticipantIDs(t, le.Participants, contactID, timelineParticipantID, lifeParticipantID)

	lifeParticipantEvents, _, err := svc.ListTimelineEvents(lifeParticipantID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents life participant failed: %v", err)
	}
	if len(lifeParticipantEvents) != 1 || lifeParticipantEvents[0].ID != te.ID {
		t.Fatalf("expected life participant to see timeline %d, got %+v", te.ID, lifeParticipantEvents)
	}
	if len(lifeParticipantEvents[0].LifeEvents) != 1 {
		t.Fatalf("expected listed timeline to include life event, got %+v", lifeParticipantEvents)
	}
	assertParticipantIDs(t, lifeParticipantEvents[0].LifeEvents[0].Participants, contactID, timelineParticipantID, lifeParticipantID)

	updated, err := svc.UpdateLifeEvent(contactID, te.ID, le.ID, vaultID, dto.UpdateLifeEventRequest{
		Summary:      "Updated shared life event",
		Participants: []string{replacementID},
	})
	if err != nil {
		t.Fatalf("UpdateLifeEvent failed: %v", err)
	}
	assertParticipantIDs(t, updated.Participants, contactID, timelineParticipantID, replacementID)

	oldParticipantEvents, _, err := svc.ListTimelineEvents(lifeParticipantID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents old participant failed: %v", err)
	}
	if len(oldParticipantEvents) != 0 {
		t.Fatalf("expected replaced life participant to lose visibility, got %+v", oldParticipantEvents)
	}

	replacementEvents, _, err := svc.ListTimelineEvents(replacementID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents replacement failed: %v", err)
	}
	if len(replacementEvents) != 1 || replacementEvents[0].ID != te.ID {
		t.Fatalf("expected replacement to see timeline %d, got %+v", te.ID, replacementEvents)
	}
}

func TestCreateTimelineEventAndUpdateLifeEventParticipantsRejectCrossVaultAndMissingContacts(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("load test vault failed: %v", err)
	}
	otherVault := models.Vault{AccountID: vault.AccountID, Type: "private", Name: "Other Vault"}
	if err := svc.db.Create(&otherVault).Error; err != nil {
		t.Fatalf("create other vault failed: %v", err)
	}
	otherContactID := createLifeEventTestContact(t, svc, otherVault.ID, "OtherVaultContact")

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	if _, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt:    time.Now(),
		Label:        "Invalid timeline participant",
		Participants: []string{otherContactID},
	}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for cross-vault timeline participant, got %v", err)
	}

	if _, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Invalid life participant",
		Participants:    []string{otherContactID},
	}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for cross-vault life participant, got %v", err)
	}

	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Valid life event",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	if _, err := svc.UpdateLifeEvent(contactID, te.ID, le.ID, vaultID, dto.UpdateLifeEventRequest{
		Summary:      "Invalid update participant",
		Participants: []string{"missing-contact-id"},
	}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for missing update participant, got %v", err)
	}

	if _, err := svc.CreateTimelineEvent(otherContactID, otherVault.ID, dto.CreateTimelineEventRequest{
		StartedAt:    time.Now(),
		Label:        "Invalid reverse participant",
		Participants: []string{contactID},
	}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for reverse cross-vault participant, got %v", err)
	}
}

func TestDeleteLifeEventParticipantPivotsAreCleaned(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	participantID := createLifeEventTestContact(t, svc, vaultID, "Participant")

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{StartedAt: time.Now(), Label: "Timeline"})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}
	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "To delete",
		Participants:    []string{participantID},
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	if err := svc.DeleteLifeEvent(te.ID, le.ID, vaultID); err != nil {
		t.Fatalf("DeleteLifeEvent failed: %v", err)
	}
	var count int64
	if err := svc.db.Model(&models.LifeEventParticipant{}).Where("life_event_id = ?", le.ID).Count(&count).Error; err != nil {
		t.Fatalf("count life event participants failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted life event pivots to be cleaned, got %d", count)
	}
}

func TestDeleteLifeEventRejectsTimelineFromAnotherVault(t *testing.T) {
	svc, _, vaultID := setupLifeEventTest(t)
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("load test vault failed: %v", err)
	}
	otherVault := models.Vault{AccountID: vault.AccountID, Type: "private", Name: "Other Vault"}
	if err := svc.db.Create(&otherVault).Error; err != nil {
		t.Fatalf("create other vault failed: %v", err)
	}
	otherContactID := createLifeEventTestContact(t, svc, otherVault.ID, "OtherVaultContact")

	otherTimeline, err := svc.CreateTimelineEvent(otherContactID, otherVault.ID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Other timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent other vault failed: %v", err)
	}
	otherLifeEvent, err := svc.AddLifeEvent(otherContactID, otherTimeline.ID, otherVault.ID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Protected life event",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent other vault failed: %v", err)
	}

	if err := svc.DeleteLifeEvent(otherTimeline.ID, otherLifeEvent.ID, vaultID); !errors.Is(err, ErrTimelineEventNotFound) {
		t.Fatalf("expected ErrTimelineEventNotFound for cross-vault delete, got %v", err)
	}

	var lifeEventCount int64
	if err := svc.db.Model(&models.LifeEvent{}).Where("id = ?", otherLifeEvent.ID).Count(&lifeEventCount).Error; err != nil {
		t.Fatalf("count protected life event failed: %v", err)
	}
	if lifeEventCount != 1 {
		t.Fatalf("expected cross-vault delete to preserve life event, got count %d", lifeEventCount)
	}
	var participantCount int64
	if err := svc.db.Model(&models.LifeEventParticipant{}).Where("life_event_id = ?", otherLifeEvent.ID).Count(&participantCount).Error; err != nil {
		t.Fatalf("count protected life event participants failed: %v", err)
	}
	if participantCount == 0 {
		t.Fatal("expected cross-vault delete to preserve life event participants")
	}
}

func TestDeleteTimelineEventParticipantPivotsAreCleaned(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	participantID := createLifeEventTestContact(t, svc, vaultID, "Participant")

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt:    time.Now(),
		Label:        "Timeline",
		Participants: []string{participantID},
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}
	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "To delete with timeline",
		Participants:    []string{participantID},
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	if err := svc.DeleteTimelineEvent(te.ID, vaultID); err != nil {
		t.Fatalf("DeleteTimelineEvent failed: %v", err)
	}
	var timelineCount int64
	if err := svc.db.Model(&models.TimelineEventParticipant{}).Where("timeline_event_id = ?", te.ID).Count(&timelineCount).Error; err != nil {
		t.Fatalf("count timeline participants failed: %v", err)
	}
	if timelineCount != 0 {
		t.Fatalf("expected deleted timeline pivots to be cleaned, got %d", timelineCount)
	}
	var lifeCount int64
	if err := svc.db.Model(&models.LifeEventParticipant{}).Where("life_event_id = ?", le.ID).Count(&lifeCount).Error; err != nil {
		t.Fatalf("count life participants failed: %v", err)
	}
	if lifeCount != 0 {
		t.Fatalf("expected deleted timeline life pivots to be cleaned, got %d", lifeCount)
	}
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

	events, meta, err := svc.ListTimelineEvents(contactID, vaultID, 1, 15)
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

func TestAddLifeEventLegacyTimelineWithoutParticipantsIncludesCurrentContact(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	legacyTimeline := models.TimelineEvent{
		VaultID:   vaultID,
		StartedAt: time.Now(),
		Label:     strPtrOrNil("Legacy timeline"),
	}
	if err := svc.db.Create(&legacyTimeline).Error; err != nil {
		t.Fatalf("create legacy timeline failed: %v", err)
	}

	le, err := svc.AddLifeEvent(contactID, legacyTimeline.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Legacy life event",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}
	assertParticipantIDs(t, le.Participants, contactID)

	var participantCount int64
	if err := svc.db.Model(&models.LifeEventParticipant{}).
		Where("life_event_id = ? AND contact_id = ?", le.ID, contactID).
		Count(&participantCount).Error; err != nil {
		t.Fatalf("count life event participant failed: %v", err)
	}
	if participantCount != 1 {
		t.Fatalf("expected current contact persisted once as life event participant, got %d", participantCount)
	}

	events, _, err := svc.ListTimelineEvents(contactID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents failed: %v", err)
	}
	if len(events) != 1 || events[0].ID != legacyTimeline.ID {
		t.Fatalf("expected current contact to see legacy timeline %d via life participant, got %+v", legacyTimeline.ID, events)
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
	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
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

	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Original",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	updated, err := svc.UpdateLifeEvent(contactID, te.ID, le.ID, vaultID, dto.UpdateLifeEventRequest{
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

	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "To delete",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	if err := svc.DeleteLifeEvent(te.ID, le.ID, vaultID); err != nil {
		t.Fatalf("DeleteLifeEvent failed: %v", err)
	}

	_, err = svc.UpdateLifeEvent(contactID, te.ID, le.ID, vaultID, dto.UpdateLifeEventRequest{Summary: "nope"})
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

	if err := svc.DeleteTimelineEvent(te.ID, vaultID); err != nil {
		t.Fatalf("DeleteTimelineEvent failed: %v", err)
	}

	events, _, err := svc.ListTimelineEvents(contactID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 timeline events after delete, got %d", len(events))
	}
}

func TestTimelineEventNotFound(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	err := svc.DeleteTimelineEvent(9999, vaultID)
	if err != ErrTimelineEventNotFound {
		t.Errorf("Expected ErrTimelineEventNotFound, got %v", err)
	}

	_, err = svc.AddLifeEvent(contactID, 9999, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "nope",
	})
	if err != ErrTimelineEventNotFound {
		t.Errorf("Expected ErrTimelineEventNotFound, got %v", err)
	}
}

func TestAddLifeEventWithEmotion(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	var eid uint
	svc.db.Raw("SELECT id FROM emotions LIMIT 1").Scan(&eid)
	if eid == 0 {
		t.Fatal("Expected at least one seeded emotion")
	}

	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "With emotion",
		EmotionID:       &eid,
	})
	if err != nil {
		t.Fatalf("AddLifeEvent with emotion failed: %v", err)
	}
	if le.EmotionID == nil || *le.EmotionID != eid {
		t.Errorf("Expected emotion_id %d, got %v", eid, le.EmotionID)
	}

	updated, err := svc.UpdateLifeEvent(contactID, te.ID, le.ID, vaultID, dto.UpdateLifeEventRequest{
		Summary:   "Updated",
		EmotionID: nil,
	})
	if err != nil {
		t.Fatalf("UpdateLifeEvent emotion to nil failed: %v", err)
	}
	if updated.EmotionID != nil {
		t.Errorf("Expected emotion_id nil after update, got %v", updated.EmotionID)
	}
}

func TestToggleTimelineEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Toggle Test",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	initialCollapsed := te.Collapsed

	if err := svc.ToggleTimelineEvent(te.ID, vaultID); err != nil {
		t.Fatalf("ToggleTimelineEvent failed: %v", err)
	}

	events, _, err := svc.ListTimelineEvents(contactID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].Collapsed == initialCollapsed {
		t.Error("Expected collapsed to be toggled")
	}
}

func TestToggleTimelineEventNotFound(t *testing.T) {
	svc, _, vaultID := setupLifeEventTest(t)

	err := svc.ToggleTimelineEvent(9999, vaultID)
	if err != ErrTimelineEventNotFound {
		t.Errorf("Expected ErrTimelineEventNotFound, got %v", err)
	}
}

func TestToggleLifeEvent(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Now(),
		Summary:         "Toggle life event",
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}

	if err := svc.ToggleLifeEvent(te.ID, le.ID, vaultID); err != nil {
		t.Fatalf("ToggleLifeEvent failed: %v", err)
	}

	events, _, err := svc.ListTimelineEvents(contactID, vaultID, 1, 15)
	if err != nil {
		t.Fatalf("ListTimelineEvents failed: %v", err)
	}
	if len(events) == 0 || len(events[0].LifeEvents) == 0 {
		t.Fatal("Expected at least 1 event with 1 life event")
	}
	if !events[0].LifeEvents[0].Collapsed {
		t.Error("Expected life event collapsed to be true after toggle")
	}
}

func TestToggleLifeEventNotFound(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	err = svc.ToggleLifeEvent(te.ID, 9999, vaultID)
	if err != ErrLifeEventNotFound {
		t.Errorf("Expected ErrLifeEventNotFound, got %v", err)
	}
}
