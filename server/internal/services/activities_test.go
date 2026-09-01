package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupUnifiedActivityTest(t *testing.T) (*ActivityService, string, models.Contact, models.Contact, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	account := models.Account{}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	vault := models.Vault{AccountID: account.ID, Name: "Vault", Type: "personal"}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatal(err)
	}
	primary := models.Contact{VaultID: vault.ID, FirstName: strPtrOrNil("Primary")}
	if err := db.Create(&primary).Error; err != nil {
		t.Fatal(err)
	}
	mentioned := models.Contact{VaultID: vault.ID, FirstName: strPtrOrNil("Mentioned")}
	if err := db.Create(&mentioned).Error; err != nil {
		t.Fatal(err)
	}
	category := models.ActivityCategory{VaultID: vault.ID}
	if err := db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	eventType := models.ActivityType{ActivityCategoryID: category.ID}
	if err := db.Create(&eventType).Error; err != nil {
		t.Fatal(err)
	}
	return NewActivityService(db), vault.ID, primary, mentioned, eventType.ID
}

func TestActivityCRUDMentionsPeriodsAndMilestones(t *testing.T) {
	svc, vaultID, primary, mentioned, typeID := setupUnifiedActivityTest(t)
	start := time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC)
	description := fmt.Sprintf("Worked with @[Mentioned](contact:%s)", mentioned.ID)
	parent, err := svc.Create(vaultID, dto.ActivityUpsertRequest{PrimaryContactID: primary.ID, ActivityTypeID: typeID, Title: "Worked at Acme", Description: description, StartDate: &start, StartPrecision: "month", EndStatus: "ongoing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(parent.Participants) != 2 {
		t.Fatalf("participants = %+v", parent.Participants)
	}
	milestoneStart := start.AddDate(1, 0, 0)
	milestone, err := svc.Create(vaultID, dto.ActivityUpsertRequest{PrimaryContactID: primary.ID, ParentID: &parent.ID, ActivityTypeID: typeID, Title: "Promoted", StartDate: &milestoneStart, StartPrecision: "day", EndStatus: "none"})
	if err != nil || milestone.ParentID == nil || *milestone.ParentID != parent.ID {
		t.Fatalf("milestone=%+v err=%v", milestone, err)
	}
	items, meta, err := svc.List(vaultID, mentioned.ID, 1, 15)
	if err != nil || meta.Total != 1 || items[0].ID != parent.ID {
		t.Fatalf("mention list=%+v meta=%+v err=%v", items, meta, err)
	}
	end := start.Add(-time.Hour)
	_, err = svc.Update(vaultID, parent.ID, dto.ActivityUpsertRequest{PrimaryContactID: primary.ID, ActivityTypeID: typeID, Title: "Invalid", StartDate: &start, EndDate: &end, EndStatus: "known"})
	if !errors.Is(err, ErrInvalidActivityTime) {
		t.Fatalf("invalid period error=%v", err)
	}
	if err := svc.Delete(vaultID, parent.ID); err != nil {
		t.Fatal(err)
	}
	var reloaded models.Activity
	if err := svc.db.First(&reloaded, milestone.ID).Error; err != nil || reloaded.ParentID != nil {
		t.Fatalf("detached milestone=%+v err=%v", reloaded, err)
	}
}

func TestActivityMarkdownRendersAndTracksUploadedFiles(t *testing.T) {
	svc, vaultID, primary, _, typeID := setupUnifiedActivityTest(t)
	file := models.File{VaultID: vaultID, UUID: "activity-markdown-file", Name: "map.pdf", MimeType: "application/pdf", Type: "document"}
	if err := svc.db.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	description := "## Notes\n\n[map](bonds-file:" + fmt.Sprint(file.ID) + ")"
	event, err := svc.Create(vaultID, dto.ActivityUpsertRequest{
		PrimaryContactID: primary.ID, ActivityTypeID: typeID, Title: "Markdown activity",
		Description: description, DescriptionFormat: "markdown",
	})
	if err != nil {
		t.Fatalf("Create activity: %v", err)
	}
	if event.DescriptionFormat != "markdown" || !strings.Contains(event.RenderedDescription, "<h2>Notes</h2>") || !strings.Contains(event.RenderedDescription, `data-bonds-file="`) {
		t.Fatalf("Markdown activity response = %+v", event)
	}
	var referenceCount int64
	svc.db.Model(&models.ContentFileReference{}).Where("file_id = ?", file.ID).Count(&referenceCount)
	if referenceCount != 1 {
		t.Fatalf("content file reference count = %d, want 1", referenceCount)
	}
	updated, err := svc.Update(vaultID, event.ID, dto.ActivityUpsertRequest{
		PrimaryContactID: primary.ID, ActivityTypeID: typeID, Title: "Legacy client update",
		Description: description,
	})
	if err != nil {
		t.Fatalf("Update Markdown activity without format: %v", err)
	}
	if updated.DescriptionFormat != "markdown" {
		t.Fatalf("DescriptionFormat after legacy update = %q, want markdown", updated.DescriptionFormat)
	}
	svc.db.Model(&models.ContentFileReference{}).Where("file_id = ?", file.ID).Count(&referenceCount)
	if referenceCount != 1 {
		t.Fatalf("content file reference count after legacy update = %d, want 1", referenceCount)
	}
	if err := svc.Delete(vaultID, event.ID); err != nil {
		t.Fatalf("Delete activity: %v", err)
	}
	svc.db.Model(&models.ContentFileReference{}).Where("file_id = ?", file.ID).Count(&referenceCount)
	if referenceCount != 0 {
		t.Fatalf("content file reference count after delete = %d, want 0", referenceCount)
	}
}

func TestContactMentionIDsDedupe(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	ids := contactMentionIDs("hi @[A](contact:" + id + ") and @[Again](contact:" + id + ")")
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("ids=%v", ids)
	}
}

func TestActivityContactRefsIncludeHoverCardDetails(t *testing.T) {
	firstName := "Alice"
	lastName := "Example"
	nickname := "Ace"
	jobPosition := "Designer"
	lastTalkedTo := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	refs := contactRefs([]models.Contact{{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		FirstName:    &firstName,
		LastName:     &lastName,
		Nickname:     &nickname,
		JobPosition:  &jobPosition,
		LastTalkedTo: &lastTalkedTo,
	}})

	if len(refs) != 1 {
		t.Fatalf("refs=%+v", refs)
	}
	got := refs[0]
	if got.FirstName != firstName || got.LastName != lastName || got.Nickname != nickname || got.JobPosition != jobPosition {
		t.Fatalf("hover card name details=%+v", got)
	}
	if got.LastTalkedTo == nil || !got.LastTalkedTo.Equal(lastTalkedTo) {
		t.Fatalf("last_talked_to=%v", got.LastTalkedTo)
	}
}

func TestActivityExplicitParticipantsKeepDescriptionMentionsIndependent(t *testing.T) {
	svc, vaultID, primary, mentioned, typeID := setupUnifiedActivityTest(t)
	start := time.Now().UTC().Truncate(time.Second)
	empty := []string{}
	description := fmt.Sprintf("和 @[Mentioned](contact:%s) 吃饭，@[Mentioned](contact:%s) 喝吐了", mentioned.ID, mentioned.ID)
	event, err := svc.Create(vaultID, dto.ActivityUpsertRequest{PrimaryContactID: primary.ID, ParticipantIDs: &empty, ActivityTypeID: typeID, Title: "吃饭", Description: description, StartDate: &start})
	if err != nil {
		t.Fatal(err)
	}
	if len(event.Participants) != 1 || event.Participants[0].ID != primary.ID {
		t.Fatalf("participants=%+v", event.Participants)
	}
	if len(event.MentionedContacts) != 1 || event.MentionedContacts[0].ID != mentioned.ID {
		t.Fatalf("mentions=%+v", event.MentionedContacts)
	}
	if event.Description != description {
		t.Fatalf("description changed: %q", event.Description)
	}
}

func TestActivityInteractionUpdatesAllParticipantsMonotonically(t *testing.T) {
	svc, vaultID, primary, other, typeID := setupUnifiedActivityTest(t)
	if err := svc.db.Model(&models.ActivityType{}).Where("id = ?", typeID).Update("counts_as_interaction", true).Error; err != nil {
		t.Fatal(err)
	}
	participants := []string{other.ID}
	newer := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	if _, err := svc.Create(vaultID, dto.ActivityUpsertRequest{PrimaryContactID: primary.ID, ParticipantIDs: &participants, ActivityTypeID: typeID, Title: "见面", StartDate: &newer}); err != nil {
		t.Fatal(err)
	}
	older := newer.Add(-24 * time.Hour)
	if _, err := svc.Create(vaultID, dto.ActivityUpsertRequest{PrimaryContactID: primary.ID, ParticipantIDs: &participants, ActivityTypeID: typeID, Title: "旧见面", StartDate: &older}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{primary.ID, other.ID} {
		var contact models.Contact
		if err := svc.db.First(&contact, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if contact.LastTalkedTo == nil || !contact.LastTalkedTo.Equal(newer) {
			t.Fatalf("contact %s last_talked_to=%v", id, contact.LastTalkedTo)
		}
	}
}

func TestDashboardActivityUsesSystemUserSubjectWithoutCreatingContact(t *testing.T) {
	svc, vaultID, _, _, typeID := setupUnifiedActivityTest(t)
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatal(err)
	}
	firstName, lastName := "Alice", "Manager"
	creator := models.User{AccountID: vault.AccountID, FirstName: &firstName, LastName: &lastName, Email: "alice-activities@example.com"}
	if err := svc.db.Create(&creator).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&models.UserVault{VaultID: vaultID, UserID: creator.ID, Permission: models.PermissionManager}).Error; err != nil {
		t.Fatal(err)
	}

	emptyParticipants := []string{}
	event, err := svc.CreateForUser(vaultID, creator.ID, dto.ActivityUpsertRequest{
		ParticipantIDs: &emptyParticipants,
		ActivityTypeID: typeID,
		Title:          "Started a new role",
	})
	if err != nil {
		t.Fatalf("CreateForUser failed: %v", err)
	}
	if event.SubjectUserID != creator.ID || event.SubjectUserName != "Alice Manager" || !event.SubjectIsCurrentUser {
		t.Fatalf("created subject = %+v", event)
	}
	if len(event.Participants) != 0 {
		t.Fatalf("dashboard event participants = %+v, want none", event.Participants)
	}
	dashboardItems, meta, err := svc.ListForUser(vaultID, creator.ID, "", 1, 15)
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if meta.Total != 1 || len(dashboardItems) != 1 || dashboardItems[0].ID != event.ID {
		t.Fatalf("dashboard list = %+v, meta = %+v", dashboardItems, meta)
	}

	otherFirstName := "Bob"
	other := models.User{AccountID: vault.AccountID, FirstName: &otherFirstName, Email: "bob-activities@example.com"}
	if err := svc.db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&models.UserVault{VaultID: vaultID, UserID: other.ID, Permission: models.PermissionManager}).Error; err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateForUser(vaultID, other.ID, event.ID, dto.ActivityUpsertRequest{
		ParticipantIDs: &emptyParticipants,
		ActivityTypeID: typeID,
		Title:          "Started the new role",
	})
	if err != nil {
		t.Fatalf("UpdateForUser failed: %v", err)
	}
	if updated.SubjectUserID != creator.ID || updated.SubjectUserName != "Alice Manager" || updated.SubjectIsCurrentUser {
		t.Fatalf("editing manager changed event subject: %+v", updated)
	}
}
