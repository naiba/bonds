package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactTest(t *testing.T) (*ContactService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewContactService(db), vault.ID, resp.User.ID, resp.User.AccountID
}

func setupContactWithFirstMetThrough(t *testing.T) (*ContactService, string, string, *dto.ContactResponse, *dto.ContactResponse) {
	t.Helper()
	svc, vaultID, userID, _ := setupContactTest(t)

	introducer, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Introducer"})
	if err != nil {
		t.Fatalf("Create introducer failed: %v", err)
	}

	firstMetAt := time.Date(2026, 3, 12, 9, 15, 0, 0, time.UTC)
	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "Met",
		FirstMetAt:               &firstMetAt,
		FirstMetThroughContactID: &introducer.ID,
	})
	if err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	return svc, vaultID, userID, introducer, contact
}

func TestCreateContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "JD",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.FirstName != "John" {
		t.Errorf("Expected first_name 'John', got '%s'", contact.FirstName)
	}
	if contact.LastName != "Doe" {
		t.Errorf("Expected last_name 'Doe', got '%s'", contact.LastName)
	}
	if contact.Nickname != "JD" {
		t.Errorf("Expected nickname 'JD', got '%s'", contact.Nickname)
	}
	if contact.ID == "" {
		t.Error("Expected contact ID to be non-empty")
	}
	if contact.VaultID != vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", vaultID, contact.VaultID)
	}
}

func TestCreateContactWithNicknameOnly(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		Nickname: "Nickname Only",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.FirstName != "" {
		t.Fatalf("expected empty first_name, got %q", contact.FirstName)
	}
	if contact.Nickname != "Nickname Only" {
		t.Fatalf("expected nickname-only contact, got %q", contact.Nickname)
	}
}

func TestCreateContactRejectsBlankFirstNameAndNickname(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "  ",
		Nickname:  "\t",
	})
	if !errors.Is(err, ErrContactNameRequired) {
		t.Fatalf("expected ErrContactNameRequired, got %v", err)
	}
}

func TestCreateContactStayInTouchFields(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	lastTalkedTo := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	frequencyDays := 30

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "Stay",
		LastTalkedTo:             &lastTalkedTo,
		StayInTouchFrequencyDays: &frequencyDays,
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	if contact.LastTalkedTo == nil || !contact.LastTalkedTo.Equal(lastTalkedTo) {
		t.Fatalf("expected last_talked_to %v, got %v", lastTalkedTo, contact.LastTalkedTo)
	}
	if contact.StayInTouchFrequencyDays == nil || *contact.StayInTouchFrequencyDays != frequencyDays {
		t.Fatalf("expected stay_in_touch_frequency_days %d, got %v", frequencyDays, contact.StayInTouchFrequencyDays)
	}
	wantTriggerDate := lastTalkedTo.AddDate(0, 0, frequencyDays)
	if contact.StayInTouchTriggerDate == nil || !contact.StayInTouchTriggerDate.Equal(wantTriggerDate) {
		t.Fatalf("expected stay_in_touch_trigger_date %v, got %v", wantTriggerDate, contact.StayInTouchTriggerDate)
	}
}

func TestCreateContactFirstMetFields(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	introducedBy, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Bob",
		LastName:  "Introducer",
	})
	if err != nil {
		t.Fatalf("Create introducer contact failed: %v", err)
	}
	firstMetAt := time.Date(2026, 3, 12, 9, 15, 0, 0, time.UTC)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "Alice",
		FirstMetAt:               &firstMetAt,
		FirstMetThroughContactID: &introducedBy.ID,
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.FirstMetAt == nil || !contact.FirstMetAt.Equal(firstMetAt) {
		t.Fatalf("expected first_met_at %v, got %v", firstMetAt, contact.FirstMetAt)
	}
	if contact.FirstMetThroughContactID == nil || *contact.FirstMetThroughContactID != introducedBy.ID {
		t.Fatalf("expected first_met_through_contact_id %s, got %v", introducedBy.ID, contact.FirstMetThroughContactID)
	}
	if contact.FirstMetThroughContact == nil {
		t.Fatal("expected first_met_through_contact brief response")
	}
	if contact.FirstMetThroughContact.ID != introducedBy.ID {
		t.Fatalf("expected first_met_through_contact id %s, got %s", introducedBy.ID, contact.FirstMetThroughContact.ID)
	}
	if contact.FirstMetThroughContact.Name != "Bob Introducer" {
		t.Fatalf("expected first_met_through_contact name 'Bob Introducer', got %q", contact.FirstMetThroughContact.Name)
	}

	got, err := svc.GetContact(contact.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if got.FirstMetThroughContact == nil || got.FirstMetThroughContact.ID != introducedBy.ID {
		t.Fatalf("expected GetContact to return met-through brief contact, got %+v", got.FirstMetThroughContact)
	}
}

func TestCreateContactFirstMetThroughRejectsDifferentVault(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)
	otherVault, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Other Vault"}, "en")
	if err != nil {
		t.Fatalf("Create other vault failed: %v", err)
	}
	otherVaultContact, err := svc.CreateContact(otherVault.ID, userID, dto.CreateContactRequest{FirstName: "Other", LastName: "Vault"})
	if err != nil {
		t.Fatalf("Create other vault contact failed: %v", err)
	}

	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "BlockedCreate",
		FirstMetThroughContactID: &otherVaultContact.ID,
	})
	if err != ErrContactNotFound {
		t.Fatalf("expected ErrContactNotFound for cross-vault met-through contact, got %v", err)
	}

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "BlockedCreate", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Fatalf("expected rejected contact not to persist, got %+v", contacts)
	}
}

func TestCreateContactFirstMetThroughRejectsNonexistentContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	nonexistentContactID := "00000000-0000-0000-0000-000000000000"

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "MissingIntroducer",
		FirstMetThroughContactID: &nonexistentContactID,
	})
	if err != ErrContactNotFound {
		t.Fatalf("expected ErrContactNotFound for nonexistent met-through contact, got %v", err)
	}

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "MissingIntroducer", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Fatalf("expected rejected contact not to persist, got %+v", contacts)
	}
}

func TestUpdateContactStayInTouchFields(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	lastTalkedTo := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	frequencyDays := 14
	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:                "Updated",
		LastTalkedTo:             &lastTalkedTo,
		StayInTouchFrequencyDays: &frequencyDays,
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	wantTriggerDate := lastTalkedTo.AddDate(0, 0, frequencyDays)
	if updated.StayInTouchTriggerDate == nil || !updated.StayInTouchTriggerDate.Equal(wantTriggerDate) {
		t.Fatalf("expected stay_in_touch_trigger_date %v, got %v", wantTriggerDate, updated.StayInTouchTriggerDate)
	}

	clearedFrequency, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:    "OnlyLastTalkedTo",
		LastTalkedTo: &lastTalkedTo,
	})
	if err != nil {
		t.Fatalf("UpdateContact without frequency failed: %v", err)
	}
	if clearedFrequency.LastTalkedTo == nil || !clearedFrequency.LastTalkedTo.Equal(lastTalkedTo) {
		t.Fatalf("expected last_talked_to %v after partial stay-in-touch update, got %v", lastTalkedTo, clearedFrequency.LastTalkedTo)
	}
	if clearedFrequency.StayInTouchFrequencyDays != nil {
		t.Fatalf("expected frequency to be nil when omitted, got %v", clearedFrequency.StayInTouchFrequencyDays)
	}
	if clearedFrequency.StayInTouchTriggerDate != nil {
		t.Fatalf("expected trigger date to be nil when frequency is omitted, got %v", clearedFrequency.StayInTouchTriggerDate)
	}
}

func TestUpdateContactFirstMetFieldsChangesAndClears(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	firstIntroducer, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "First", LastName: "Introducer"})
	if err != nil {
		t.Fatalf("Create first introducer failed: %v", err)
	}
	secondIntroducer, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Second", LastName: "Introducer"})
	if err != nil {
		t.Fatalf("Create second introducer failed: %v", err)
	}

	firstMetAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:                "Updated",
		FirstMetAt:               &firstMetAt,
		FirstMetThroughContactID: &firstIntroducer.ID,
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	if updated.FirstMetAt == nil || !updated.FirstMetAt.Equal(firstMetAt) {
		t.Fatalf("expected first_met_at %v, got %v", firstMetAt, updated.FirstMetAt)
	}
	if updated.FirstMetThroughContact == nil || updated.FirstMetThroughContact.Name != "First Introducer" {
		t.Fatalf("expected first introducer brief contact, got %+v", updated.FirstMetThroughContact)
	}

	changedFirstMetAt := time.Date(2026, 5, 2, 13, 30, 0, 0, time.UTC)
	changed, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:                "Changed",
		FirstMetAt:               &changedFirstMetAt,
		FirstMetThroughContactID: &secondIntroducer.ID,
	})
	if err != nil {
		t.Fatalf("UpdateContact change failed: %v", err)
	}
	if changed.FirstMetAt == nil || !changed.FirstMetAt.Equal(changedFirstMetAt) {
		t.Fatalf("expected changed first_met_at %v, got %v", changedFirstMetAt, changed.FirstMetAt)
	}
	if changed.FirstMetThroughContactID == nil || *changed.FirstMetThroughContactID != secondIntroducer.ID {
		t.Fatalf("expected changed met-through id %s, got %v", secondIntroducer.ID, changed.FirstMetThroughContactID)
	}
	if changed.FirstMetThroughContact == nil || changed.FirstMetThroughContact.Name != "Second Introducer" {
		t.Fatalf("expected second introducer brief contact, got %+v", changed.FirstMetThroughContact)
	}

	cleared, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{FirstName: "Cleared"})
	if err != nil {
		t.Fatalf("UpdateContact clear failed: %v", err)
	}
	if cleared.FirstMetAt != nil {
		t.Fatalf("expected first_met_at to clear on full update, got %v", cleared.FirstMetAt)
	}
	if cleared.FirstMetThroughContactID != nil {
		t.Fatalf("expected first_met_through_contact_id to clear on full update, got %v", *cleared.FirstMetThroughContactID)
	}
	if cleared.FirstMetThroughContact != nil {
		t.Fatalf("expected first_met_through_contact to clear on full update, got %+v", cleared.FirstMetThroughContact)
	}
}

func TestUpdateContactFirstMetThroughRejectsDifferentVault(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)
	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	otherVault, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Other Vault"}, "en")
	if err != nil {
		t.Fatalf("Create other vault failed: %v", err)
	}
	otherVaultContact, err := svc.CreateContact(otherVault.ID, userID, dto.CreateContactRequest{FirstName: "Other", LastName: "Vault"})
	if err != nil {
		t.Fatalf("Create other vault contact failed: %v", err)
	}

	_, err = svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:                "BlockedUpdate",
		FirstMetThroughContactID: &otherVaultContact.ID,
	})
	if err != ErrContactNotFound {
		t.Fatalf("expected ErrContactNotFound for cross-vault met-through contact, got %v", err)
	}

	reloaded, err := svc.GetContact(created.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if reloaded.FirstName != "Original" {
		t.Fatalf("expected rejected update not to modify first_name, got %q", reloaded.FirstName)
	}
	if reloaded.FirstMetThroughContactID != nil || reloaded.FirstMetThroughContact != nil {
		t.Fatalf("expected rejected update not to persist met-through data, got id=%v brief=%+v", reloaded.FirstMetThroughContactID, reloaded.FirstMetThroughContact)
	}
}

func TestUpdateContactFirstMetThroughRejectsNonexistentContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	nonexistentContactID := "00000000-0000-0000-0000-000000000000"

	_, err = svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:                "MissingIntroducerUpdate",
		FirstMetThroughContactID: &nonexistentContactID,
	})
	if err != ErrContactNotFound {
		t.Fatalf("expected ErrContactNotFound for nonexistent met-through contact, got %v", err)
	}

	reloaded, err := svc.GetContact(created.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if reloaded.FirstName != "Original" {
		t.Fatalf("expected rejected update not to modify first_name, got %q", reloaded.FirstName)
	}
	if reloaded.FirstMetThroughContactID != nil || reloaded.FirstMetThroughContact != nil {
		t.Fatalf("expected rejected update not to persist met-through data, got id=%v brief=%+v", reloaded.FirstMetThroughContactID, reloaded.FirstMetThroughContact)
	}
}

func TestListContacts(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contacts, meta, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}
	if meta.Total != 2 {
		t.Errorf("Expected total 2, got %d", meta.Total)
	}
}

func TestListContactsReturnsFirstMetThroughBriefContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	introducedBy, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Casey", LastName: "Connector"})
	if err != nil {
		t.Fatalf("Create introducer contact failed: %v", err)
	}
	firstMetAt := time.Date(2026, 6, 3, 18, 45, 0, 0, time.UTC)
	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "Dana",
		FirstMetAt:               &firstMetAt,
		FirstMetThroughContactID: &introducedBy.ID,
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "Dana", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 filtered contact, got %d", len(contacts))
	}
	if contacts[0].ID != created.ID {
		t.Fatalf("expected contact %s, got %s", created.ID, contacts[0].ID)
	}
	if contacts[0].FirstMetAt == nil || !contacts[0].FirstMetAt.Equal(firstMetAt) {
		t.Fatalf("expected first_met_at %v, got %v", firstMetAt, contacts[0].FirstMetAt)
	}
	if contacts[0].FirstMetThroughContact == nil {
		t.Fatal("expected list response to include met-through brief contact")
	}
	if contacts[0].FirstMetThroughContact.ID != introducedBy.ID || contacts[0].FirstMetThroughContact.Name != "Casey Connector" {
		t.Fatalf("expected met-through brief Casey Connector, got %+v", contacts[0].FirstMetThroughContact)
	}
}

func TestListContactsSortByFirstName(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie"})
	svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 3 {
		t.Fatalf("Expected 3 contacts, got %d", len(contacts))
	}
	if contacts[0].FirstName != "Alice" {
		t.Errorf("Expected first contact 'Alice', got '%s'", contacts[0].FirstName)
	}
	if contacts[1].FirstName != "Bob" {
		t.Errorf("Expected second contact 'Bob', got '%s'", contacts[1].FirstName)
	}
	if contacts[2].FirstName != "Charlie" {
		t.Errorf("Expected third contact 'Charlie', got '%s'", contacts[2].FirstName)
	}
}

func TestListContactsSortByFirstMetAtPutsDatedContactsBeforeNil(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	olderMetAt := time.Date(2025, 1, 10, 10, 0, 0, 0, time.UTC)
	newerMetAt := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	if _, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "NoDate"}); err != nil {
		t.Fatalf("Create no-date contact failed: %v", err)
	}
	if _, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Older", FirstMetAt: &olderMetAt}); err != nil {
		t.Fatalf("Create older contact failed: %v", err)
	}
	if _, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Newer", FirstMetAt: &newerMetAt}); err != nil {
		t.Fatalf("Create newer contact failed: %v", err)
	}
	if _, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "AlsoNoDate"}); err != nil {
		t.Fatalf("Create second no-date contact failed: %v", err)
	}

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "first_met_at", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 4 {
		t.Fatalf("expected 4 contacts, got %d", len(contacts))
	}
	wantNames := []string{"Newer", "Older", "AlsoNoDate", "NoDate"}
	for i, wantName := range wantNames {
		if contacts[i].FirstName != wantName {
			t.Fatalf("expected contact %d to be %s, got %s", i, wantName, contacts[i].FirstName)
		}
	}
	if contacts[0].FirstMetAt == nil || contacts[1].FirstMetAt == nil {
		t.Fatalf("expected dated contacts first, got %+v", contacts[:2])
	}
	if contacts[2].FirstMetAt != nil || contacts[3].FirstMetAt != nil {
		t.Fatalf("expected nil first_met_at contacts last, got %+v", contacts[2:])
	}
}

func TestListContactsByLabelFirstMetResponseAndSort(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	label := models.Label{VaultID: vaultID, Name: "MetAtConference", Slug: "met-at-conference"}
	if err := svc.db.Create(&label).Error; err != nil {
		t.Fatalf("Create label failed: %v", err)
	}
	introducedBy, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Evan", LastName: "Host"})
	if err != nil {
		t.Fatalf("Create introducer contact failed: %v", err)
	}
	olderMetAt := time.Date(2025, 7, 4, 10, 0, 0, 0, time.UTC)
	newerMetAt := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	older, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "OlderLabel", FirstMetAt: &olderMetAt})
	if err != nil {
		t.Fatalf("Create older label contact failed: %v", err)
	}
	newer, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "NewerLabel",
		FirstMetAt:               &newerMetAt,
		FirstMetThroughContactID: &introducedBy.ID,
	})
	if err != nil {
		t.Fatalf("Create newer label contact failed: %v", err)
	}
	withoutDate, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "NoDateLabel"})
	if err != nil {
		t.Fatalf("Create no-date label contact failed: %v", err)
	}
	for _, contactID := range []string{older.ID, newer.ID, withoutDate.ID} {
		contactLabel := models.ContactLabel{LabelID: label.ID, ContactID: contactID}
		if err := svc.db.Create(&contactLabel).Error; err != nil {
			t.Fatalf("Create contact label for %s failed: %v", contactID, err)
		}
	}

	contacts, meta, err := svc.ListContactsByLabel(vaultID, userID, label.ID, 1, 15, "first_met_at", "")
	if err != nil {
		t.Fatalf("ListContactsByLabel failed: %v", err)
	}
	if meta.Total != 3 || len(contacts) != 3 {
		t.Fatalf("expected 3 labeled contacts, got len=%d total=%d", len(contacts), meta.Total)
	}
	wantNames := []string{"NewerLabel", "OlderLabel", "NoDateLabel"}
	for i, wantName := range wantNames {
		if contacts[i].FirstName != wantName {
			t.Fatalf("expected labeled contact %d to be %s, got %s", i, wantName, contacts[i].FirstName)
		}
	}
	if contacts[0].FirstMetThroughContact == nil {
		t.Fatal("expected labeled list response to include met-through brief contact")
	}
	if contacts[0].FirstMetThroughContact.ID != introducedBy.ID || contacts[0].FirstMetThroughContact.Name != "Evan Host" {
		t.Fatalf("expected met-through brief Evan Host, got %+v", contacts[0].FirstMetThroughContact)
	}
	if contacts[2].FirstMetAt != nil {
		t.Fatalf("expected nil first_met_at labeled contact last, got %+v", contacts[2])
	}
}

func TestUpdateContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName: "Updated",
		LastName:  "Name",
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	if updated.FirstName != "Updated" {
		t.Errorf("Expected first_name 'Updated', got '%s'", updated.FirstName)
	}
	if updated.LastName != "Name" {
		t.Errorf("Expected last_name 'Name', got '%s'", updated.LastName)
	}
}

func TestUpdateContactWithNicknameOnly(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		Nickname: "Updated Nickname",
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	if updated.FirstName != "" {
		t.Fatalf("expected empty first_name, got %q", updated.FirstName)
	}
	if updated.Nickname != "Updated Nickname" {
		t.Fatalf("expected nickname-only update, got %q", updated.Nickname)
	}
}

func TestUpdateContactRejectsBlankFirstNameAndNickname(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	_, err = svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName: "  ",
		Nickname:  "\n",
	})
	if !errors.Is(err, ErrContactNameRequired) {
		t.Fatalf("expected ErrContactNameRequired, got %v", err)
	}

	reloaded, err := svc.GetContact(created.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if reloaded.FirstName != "Original" {
		t.Fatalf("expected rejected update not to modify first_name, got %q", reloaded.FirstName)
	}
}

func TestDeleteContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "ToDelete"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	if err := svc.DeleteContact(created.ID, vaultID); err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "", "", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts after delete, got %d", len(contacts))
	}
}

func TestDeleteContact_FirstMetThroughSoftDeleteSetsNull(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	introducer, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Soft", LastName: "Introducer"})
	if err != nil {
		t.Fatalf("Create introducer failed: %v", err)
	}
	introduced, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "Introduced",
		FirstMetThroughContactID: &introducer.ID,
	})
	if err != nil {
		t.Fatalf("Create introduced contact failed: %v", err)
	}

	if err := svc.DeleteContact(introducer.ID, vaultID); err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	var reloaded models.Contact
	if err := svc.db.First(&reloaded, "id = ?", introduced.ID).Error; err != nil {
		t.Fatalf("Reload introduced contact failed: %v", err)
	}
	if reloaded.FirstMetThroughContactID != nil {
		t.Fatalf("expected soft delete to clear first_met_through_contact_id, got %v", *reloaded.FirstMetThroughContactID)
	}

	resp, err := svc.GetContact(introduced.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if resp.FirstMetThroughContactID != nil || resp.FirstMetThroughContact != nil {
		t.Fatalf("expected response to omit deleted introducer, got id=%v contact=%+v", resp.FirstMetThroughContactID, resp.FirstMetThroughContact)
	}
}

func TestDeleteContact_FirstMetThroughHardDeleteSetsNull(t *testing.T) {
	db := testutil.SetupTestDBWithFKConstraints(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "FirstMet",
		LastName:  "User",
		Email:     "first-met-delete@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "First Met Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	alice, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("Create Alice failed: %v", err)
	}
	bob, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("Create Bob failed: %v", err)
	}
	if err := db.Model(&models.Contact{}).
		Where("id = ?", alice.ID).
		Update("first_met_through_contact_id", bob.ID).Error; err != nil {
		t.Fatalf("Set first_met_through_contact_id failed: %v", err)
	}

	var bobModel models.Contact
	if err := db.First(&bobModel, "id = ?", bob.ID).Error; err != nil {
		t.Fatalf("Load Bob failed: %v", err)
	}
	// Monica imports can create first_met_through self-references. Hard-delete
	// paths must not be blocked by that optional historical pointer.
	if err := db.Unscoped().Delete(&bobModel).Error; err != nil {
		t.Fatalf("Hard delete referenced contact failed: %v", err)
	}

	var reloaded models.Contact
	if err := db.First(&reloaded, "id = ?", alice.ID).Error; err != nil {
		t.Fatalf("Reload Alice failed: %v", err)
	}
	if reloaded.FirstMetThroughContactID != nil {
		t.Fatalf("expected first_met_through_contact_id to be cleared, got %v", *reloaded.FirstMetThroughContactID)
	}
}

func TestToggleArchive(t *testing.T) {
	svc, vaultID, _, introducer, created := setupContactWithFirstMetThrough(t)

	toggled, err := svc.ToggleArchive(created.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}
	if !toggled.IsArchived {
		t.Error("Expected contact to be archived after toggle")
	}
	if toggled.FirstMetThroughContactID == nil || *toggled.FirstMetThroughContactID != introducer.ID {
		t.Fatalf("expected first_met_through_contact_id %s, got %v", introducer.ID, toggled.FirstMetThroughContactID)
	}
	if toggled.FirstMetThroughContact == nil || toggled.FirstMetThroughContact.ID != introducer.ID {
		t.Fatalf("expected first_met_through_contact to be preserved, got %+v", toggled.FirstMetThroughContact)
	}

	toggledBack, err := svc.ToggleArchive(created.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive back failed: %v", err)
	}
	if toggledBack.IsArchived {
		t.Error("Expected contact to not be archived after second toggle")
	}
	if toggledBack.FirstMetThroughContactID == nil || *toggledBack.FirstMetThroughContactID != introducer.ID {
		t.Fatalf("expected first_met_through_contact_id %s after second toggle, got %v", introducer.ID, toggledBack.FirstMetThroughContactID)
	}
	if toggledBack.FirstMetThroughContact == nil || toggledBack.FirstMetThroughContact.ID != introducer.ID {
		t.Fatalf("expected first_met_through_contact after second toggle, got %+v", toggledBack.FirstMetThroughContact)
	}
}

func TestToggleFavorite(t *testing.T) {
	svc, vaultID, userID, introducer, created := setupContactWithFirstMetThrough(t)

	toggled, err := svc.ToggleFavorite(created.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	if !toggled.IsFavorite {
		t.Error("Expected contact to be favorite after toggle")
	}
	if toggled.FirstMetThroughContactID == nil || *toggled.FirstMetThroughContactID != introducer.ID {
		t.Fatalf("expected first_met_through_contact_id %s, got %v", introducer.ID, toggled.FirstMetThroughContactID)
	}
	if toggled.FirstMetThroughContact == nil || toggled.FirstMetThroughContact.ID != introducer.ID {
		t.Fatalf("expected first_met_through_contact to be preserved, got %+v", toggled.FirstMetThroughContact)
	}

	toggledBack, err := svc.ToggleFavorite(created.ID, userID, vaultID)
	if err != nil {
		t.Fatalf("ToggleFavorite back failed: %v", err)
	}
	if toggledBack.IsFavorite {
		t.Error("Expected contact to not be favorite after second toggle")
	}
	if toggledBack.FirstMetThroughContactID == nil || *toggledBack.FirstMetThroughContactID != introducer.ID {
		t.Fatalf("expected first_met_through_contact_id %s after second toggle, got %v", introducer.ID, toggledBack.FirstMetThroughContactID)
	}
	if toggledBack.FirstMetThroughContact == nil || toggledBack.FirstMetThroughContact.ID != introducer.ID {
		t.Fatalf("expected first_met_through_contact after second toggle, got %+v", toggledBack.FirstMetThroughContact)
	}
}

func TestMarkCaughtUpPreservesFirstMetThroughMetadata(t *testing.T) {
	svc, vaultID, _, introducer, created := setupContactWithFirstMetThrough(t)

	marked, err := svc.MarkCaughtUp(created.ID, vaultID)
	if err != nil {
		t.Fatalf("MarkCaughtUp failed: %v", err)
	}
	if marked.LastTalkedTo == nil {
		t.Fatal("expected last_talked_to to be set after marking caught up")
	}
	if marked.FirstMetThroughContactID == nil || *marked.FirstMetThroughContactID != introducer.ID {
		t.Fatalf("expected first_met_through_contact_id %s, got %v", introducer.ID, marked.FirstMetThroughContactID)
	}
	if marked.FirstMetThroughContact == nil || marked.FirstMetThroughContact.ID != introducer.ID {
		t.Fatalf("expected first_met_through_contact to be preserved, got %+v", marked.FirstMetThroughContact)
	}
}

func TestListCatchUpPromptsFiltersAndSortsByPriority(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	now := time.Now()

	createPromptContact := func(firstName string, lastTalkedTo time.Time, frequencyDays int) *dto.ContactResponse {
		contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
			FirstName:                firstName,
			LastTalkedTo:             &lastTalkedTo,
			StayInTouchFrequencyDays: &frequencyDays,
		})
		if err != nil {
			t.Fatalf("CreateContact %s failed: %v", firstName, err)
		}
		return contact
	}

	veryOverdue := createPromptContact("VeryOverdue", now.AddDate(0, 0, -90), 30)
	lessOverdue := createPromptContact("LessOverdue", now.AddDate(0, 0, -40), 20)
	createPromptContact("NotDue", now.AddDate(0, 0, -5), 30)
	createPromptContact("NoFrequency", now.AddDate(0, 0, -100), 0)
	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "NoLastTalkedTo"})
	if err != nil {
		t.Fatalf("CreateContact without last_talked_to failed: %v", err)
	}
	archived := createPromptContact("Archived", now.AddDate(0, 0, -100), 10)
	if _, err := svc.ToggleArchive(archived.ID, vaultID); err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}

	shadow := models.Contact{
		VaultID:                  vaultID,
		FirstName:                strPtrOrNil("Shadow"),
		LastTalkedTo:             ptrTime(now.AddDate(0, 0, -100)),
		StayInTouchFrequencyDays: ptrInt(10),
		StayInTouchTriggerDate:   ptrTime(now.AddDate(0, 0, -90)),
	}
	if err := svc.db.Create(&shadow).Error; err != nil {
		t.Fatalf("Create shadow contact failed: %v", err)
	}
	if err := svc.db.Model(&shadow).Updates(map[string]interface{}{"can_be_deleted": false, "listed": false}).Error; err != nil {
		t.Fatalf("Update shadow contact failed: %v", err)
	}

	prompts, err := svc.ListCatchUpPrompts(vaultID)
	if err != nil {
		t.Fatalf("ListCatchUpPrompts failed: %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("expected 2 catch-up prompts, got %d: %+v", len(prompts), prompts)
	}
	if prompts[0].ContactID != veryOverdue.ID || prompts[1].ContactID != lessOverdue.ID {
		t.Fatalf("expected prompts sorted by priority VeryOverdue then LessOverdue, got %+v", prompts)
	}
	if prompts[0].DaysOverdue < 59 || prompts[0].DaysOverdue > 60 {
		t.Fatalf("expected VeryOverdue days_overdue near 60, got %d", prompts[0].DaysOverdue)
	}
	if prompts[0].PriorityScore < 1.96 || prompts[0].PriorityScore > 2.01 {
		t.Fatalf("expected VeryOverdue priority score near 2.0, got %f", prompts[0].PriorityScore)
	}
	if prompts[1].PriorityScore >= prompts[0].PriorityScore {
		t.Fatalf("expected second prompt priority lower than first, got %f >= %f", prompts[1].PriorityScore, prompts[0].PriorityScore)
	}
}

func TestMarkCaughtUpRecomputesStayInTouchTrigger(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	lastTalkedTo := time.Now().AddDate(0, 0, -45)
	frequencyDays := 30
	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:                "CatchUp",
		LastTalkedTo:             &lastTalkedTo,
		StayInTouchFrequencyDays: &frequencyDays,
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	before := time.Now().Add(-time.Second)
	updated, err := svc.MarkCaughtUp(created.ID, vaultID)
	if err != nil {
		t.Fatalf("MarkCaughtUp failed: %v", err)
	}
	after := time.Now().Add(time.Second)

	if updated.LastTalkedTo == nil || updated.LastTalkedTo.Before(before) || updated.LastTalkedTo.After(after) {
		t.Fatalf("expected last_talked_to to be near now, got %v", updated.LastTalkedTo)
	}
	wantTriggerDate := updated.LastTalkedTo.AddDate(0, 0, frequencyDays)
	if updated.StayInTouchTriggerDate == nil || !updated.StayInTouchTriggerDate.Equal(wantTriggerDate) {
		t.Fatalf("expected recomputed trigger date %v, got %v", wantTriggerDate, updated.StayInTouchTriggerDate)
	}

	prompts, err := svc.ListCatchUpPrompts(vaultID)
	if err != nil {
		t.Fatalf("ListCatchUpPrompts failed: %v", err)
	}
	for _, prompt := range prompts {
		if prompt.ContactID == created.ID {
			t.Fatalf("expected caught-up contact to no longer be due, got %+v", prompt)
		}
	}
}

func TestContactNotFound(t *testing.T) {
	svc, _, _, _ := setupContactTest(t)

	_, err := svc.GetContact("nonexistent-id", "some-user", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.UpdateContact("nonexistent-id", "some-vault", dto.UpdateContactRequest{FirstName: "nope"})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	err = svc.DeleteContact("nonexistent-id", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.ToggleArchive("nonexistent-id", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.ToggleFavorite("nonexistent-id", "some-user", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func ptrBool(b bool) *bool           { return &b }
func ptrUint(u uint) *uint           { return &u }
func ptrInt(i int) *int              { return &i }
func ptrTime(t time.Time) *time.Time { return &t }

func TestCreateContact_WithAllFields(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)

	db := svc.db
	gender := models.Gender{AccountID: accountID, Name: strPtrOrNil("Custom")}
	db.Create(&gender)
	pronoun := models.Pronoun{AccountID: accountID, Name: strPtrOrNil("they/them/custom")}
	db.Create(&pronoun)

	var tmpl models.Template
	db.Where("account_id = ?", accountID).First(&tmpl)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:  "John",
		LastName:   "Doe",
		MiddleName: "Michael",
		Nickname:   "Johnny",
		MaidenName: "Smith",
		Prefix:     "Mr.",
		Suffix:     "Jr.",
		GenderID:   ptrUint(gender.ID),
		PronounID:  ptrUint(pronoun.ID),
		TemplateID: ptrUint(tmpl.ID),
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.FirstName != "John" {
		t.Errorf("Expected first_name 'John', got '%s'", contact.FirstName)
	}
	if contact.LastName != "Doe" {
		t.Errorf("Expected last_name 'Doe', got '%s'", contact.LastName)
	}
	if contact.MiddleName != "Michael" {
		t.Errorf("Expected middle_name 'Michael', got '%s'", contact.MiddleName)
	}
	if contact.Nickname != "Johnny" {
		t.Errorf("Expected nickname 'Johnny', got '%s'", contact.Nickname)
	}
	if contact.MaidenName != "Smith" {
		t.Errorf("Expected maiden_name 'Smith', got '%s'", contact.MaidenName)
	}
	if contact.Prefix != "Mr." {
		t.Errorf("Expected prefix 'Mr.', got '%s'", contact.Prefix)
	}
	if contact.Suffix != "Jr." {
		t.Errorf("Expected suffix 'Jr.', got '%s'", contact.Suffix)
	}
	if contact.GenderID == nil || *contact.GenderID != gender.ID {
		t.Errorf("Expected gender_id %d, got %v", gender.ID, contact.GenderID)
	}
	if contact.PronounID == nil || *contact.PronounID != pronoun.ID {
		t.Errorf("Expected pronoun_id %d, got %v", pronoun.ID, contact.PronounID)
	}
	if contact.TemplateID == nil || *contact.TemplateID != tmpl.ID {
		t.Errorf("Expected template_id %d, got %v", tmpl.ID, contact.TemplateID)
	}
	if !contact.Listed {
		t.Error("Expected listed to be true by default")
	}
	if contact.IsArchived {
		t.Error("Expected is_archived to be false by default")
	}
}

func TestCreateContact_WithListedFalse(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Unlisted",
		Listed:    ptrBool(false),
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.Listed {
		t.Error("Expected listed to be false")
	}
	if !contact.IsArchived {
		t.Error("Expected is_archived to be true when listed is false")
	}
}

func TestUpdateContact_WithAllFields(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Original",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	db := svc.db
	gender := models.Gender{AccountID: accountID, Name: strPtrOrNil("NonBinary")}
	db.Create(&gender)
	pronoun := models.Pronoun{AccountID: accountID, Name: strPtrOrNil("ze/zir")}
	db.Create(&pronoun)

	var tmpl models.Template
	db.Where("account_id = ?", accountID).First(&tmpl)

	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:  "Updated",
		LastName:   "NewLast",
		MiddleName: "NewMiddle",
		Nickname:   "NewNick",
		MaidenName: "NewMaiden",
		Prefix:     "Dr.",
		Suffix:     "III",
		GenderID:   ptrUint(gender.ID),
		PronounID:  ptrUint(pronoun.ID),
		TemplateID: ptrUint(tmpl.ID),
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	if updated.FirstName != "Updated" {
		t.Errorf("Expected first_name 'Updated', got '%s'", updated.FirstName)
	}
	if updated.LastName != "NewLast" {
		t.Errorf("Expected last_name 'NewLast', got '%s'", updated.LastName)
	}
	if updated.MiddleName != "NewMiddle" {
		t.Errorf("Expected middle_name 'NewMiddle', got '%s'", updated.MiddleName)
	}
	if updated.Nickname != "NewNick" {
		t.Errorf("Expected nickname 'NewNick', got '%s'", updated.Nickname)
	}
	if updated.MaidenName != "NewMaiden" {
		t.Errorf("Expected maiden_name 'NewMaiden', got '%s'", updated.MaidenName)
	}
	if updated.Prefix != "Dr." {
		t.Errorf("Expected prefix 'Dr.', got '%s'", updated.Prefix)
	}
	if updated.Suffix != "III" {
		t.Errorf("Expected suffix 'III', got '%s'", updated.Suffix)
	}
	if updated.GenderID == nil || *updated.GenderID != gender.ID {
		t.Errorf("Expected gender_id %d, got %v", gender.ID, updated.GenderID)
	}
	if updated.PronounID == nil || *updated.PronounID != pronoun.ID {
		t.Errorf("Expected pronoun_id %d, got %v", pronoun.ID, updated.PronounID)
	}
	if updated.TemplateID == nil || *updated.TemplateID != tmpl.ID {
		t.Errorf("Expected template_id %d, got %v", tmpl.ID, updated.TemplateID)
	}
}

func TestQuickSearch(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice", LastName: "Johnson"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob", LastName: "Smith"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie", LastName: "Johnson"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	results, err := svc.QuickSearch(vaultID, "Johnson")
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if results[0].ID == "" {
		t.Error("Expected non-empty ID")
	}
	if results[0].Name == "" {
		t.Error("Expected non-empty Name")
	}

	results, err = svc.QuickSearch(vaultID, "Bob")
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Name != "Bob Smith" {
		t.Errorf("Expected name 'Bob Smith', got '%s'", results[0].Name)
	}

	results, err = svc.QuickSearch(vaultID, "nonexistent")
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}

	results, err = svc.QuickSearch(vaultID, "")
	if err != nil {
		t.Fatalf("QuickSearch with empty term failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty term, got %d", len(results))
	}
}

func TestListContacts_BirthdayAgeGroups(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-bdaygroup-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	vaultID := vault.ID
	userID := resp.User.ID

	contactSvc := NewContactService(db)
	c1, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	var birthdateType models.ContactImportantDateType
	if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("Failed to find Birthdate type: %v", err)
	}

	dateSvc := NewImportantDateService(db)
	day, month, year := 15, 6, 1990
	_, err = dateSvc.Create(c1.ID, vaultID, dto.CreateImportantDateRequest{
		Label:                      "Birthdate",
		Day:                        &day,
		Month:                      &month,
		Year:                       &year,
		ContactImportantDateTypeID: &birthdateType.ID,
	})
	if err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}

	groupSvc := NewGroupService(db)
	grp, err := groupSvc.Create(vaultID, dto.CreateGroupRequest{Name: "Family"})
	if err != nil {
		t.Fatalf("Create group failed: %v", err)
	}
	if err := groupSvc.AddContactToGroup(c1.ID, dto.AddContactToGroupRequest{GroupID: grp.ID}); err != nil {
		t.Fatalf("AddContactToGroup failed: %v", err)
	}

	contacts, _, err := contactSvc.ListContacts(vaultID, userID, 1, 20, "", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("Expected 2 contacts, got %d", len(contacts))
	}

	var alice, bob *dto.ContactResponse
	for i := range contacts {
		if contacts[i].FirstName == "Alice" {
			alice = &contacts[i]
		}
		if contacts[i].FirstName == "Bob" {
			bob = &contacts[i]
		}
	}
	if alice == nil || bob == nil {
		t.Fatal("Expected to find both Alice and Bob in results")
	}

	if alice.Birthday == nil {
		t.Error("Expected Alice to have a birthday")
	} else if *alice.Birthday != "1990-06-15" {
		t.Errorf("Expected birthday '1990-06-15', got '%s'", *alice.Birthday)
	}
	if alice.Age == nil {
		t.Error("Expected Alice to have an age")
	} else if *alice.Age < 30 {
		t.Errorf("Expected age >= 30, got %d", *alice.Age)
	}
	if len(alice.Groups) != 1 {
		t.Fatalf("Expected 1 group for Alice, got %d", len(alice.Groups))
	}
	if alice.Groups[0].Name != "Family" {
		t.Errorf("Expected group name 'Family', got '%s'", alice.Groups[0].Name)
	}

	if bob.Birthday != nil {
		t.Errorf("Expected Bob to have no birthday, got '%s'", *bob.Birthday)
	}
	if bob.Age != nil {
		t.Errorf("Expected Bob to have no age, got %d", *bob.Age)
	}
	if len(bob.Groups) != 0 {
		t.Errorf("Expected 0 groups for Bob, got %d", len(bob.Groups))
	}
}
