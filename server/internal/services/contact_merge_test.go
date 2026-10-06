package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"gorm.io/gorm"
)

func TestMergeContactsPreservesInformation(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alicia", Nickname: "Ally"})
	if err != nil {
		t.Fatal(err)
	}
	note := models.Note{ContactID: source.ID, VaultID: vaultID, Body: "Keep this note"}
	if err := svc.db.Create(&note).Error; err != nil {
		t.Fatal(err)
	}
	result, err := svc.MergeContacts(vaultID, userID, dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != target.ID || result.FirstName != "Alice" || result.Nickname != "Ally" {
		t.Fatalf("unexpected merged profile: %+v", result)
	}
	if err := svc.db.First(&note, note.ID).Error; err != nil {
		t.Fatal(err)
	}
	if note.ContactID != target.ID || note.Body != "Keep this note" {
		t.Fatalf("note lost: %+v", note)
	}
	var snapshots []models.Note
	if err := svc.db.Where("contact_id = ? AND source_type = ?", target.ID, "contact_merge").Find(&snapshots).Error; err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || !strings.Contains(snapshots[0].Body, "Alicia") {
		t.Fatalf("missing original profile: %+v", snapshots)
	}
	var deleted models.Contact
	if err := svc.db.First(&deleted, "id = ?", source.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("source still active: %v", err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("replay must fail without duplicating data: %v", err)
	}
}

func TestMergeContactsAssociations(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)
	create := func(value any) {
		t.Helper()
		if err := svc.db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Primary")}
	create(&target)
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Duplicate")}
	create(&source)
	third := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Friend"), FirstMetThroughContactID: &source.ID}
	create(&third)
	task := models.ContactTask{VaultID: vaultID, Label: "Follow up"}
	create(&task)
	activity := models.Activity{VaultID: vaultID, Title: "Dinner", PaidByContactID: &source.ID}
	create(&activity)
	var role models.GroupTypeRole
	if err := svc.db.First(&role).Error; err != nil {
		t.Fatal(err)
	}
	group := models.Group{VaultID: vaultID, Name: "Friends", GroupTypeID: &role.GroupTypeID}
	create(&group)
	label := models.Label{VaultID: vaultID, Name: "Friend", Slug: "friend"}
	create(&label)
	for _, contactID := range []string{target.ID, source.ID} {
		create(&models.TaskContact{ContactID: contactID, ContactTaskID: task.ID})
		create(&models.ActivityParticipant{ContactID: contactID, ActivityID: activity.ID})
		var roleID *uint
		if contactID == source.ID {
			roleID = &role.ID
		}
		create(&models.ContactGroup{ContactID: contactID, GroupID: group.ID, GroupTypeRoleID: roleID})
		create(&models.ContactLabel{ContactID: contactID, LabelID: label.ID})
		create(&models.ContactVaultUser{ContactID: contactID, VaultID: vaultID, UserID: userID, NumberOfViews: 3, IsFavorite: contactID == source.ID})
	}
	occurredAt := time.Date(2024, 3, 2, 12, 0, 0, 0, time.UTC)
	company := models.Company{VaultID: vaultID, Name: "Garden studio"}
	create(&company)
	address := models.Address{VaultID: vaultID, Line1: strPtrOrNil("Synthetic garden road")}
	create(&address)
	metric := models.LifeMetric{VaultID: vaultID, Label: "Walk"}
	create(&metric)
	journal := models.Journal{VaultID: vaultID, Name: "Memories"}
	create(&journal)
	post := models.Post{JournalID: journal.ID, Title: strPtrOrNil("Picnic"), WrittenAt: occurredAt}
	create(&post)
	create(&models.ContactPost{ContactID: source.ID, PostID: post.ID})
	create(&models.ContactPost{ContactID: target.ID, PostID: post.ID})
	var petCategory models.PetCategory
	if err := svc.db.First(&petCategory).Error; err != nil {
		t.Fatal(err)
	}
	create(&models.Pet{ContactID: source.ID, PetCategoryID: petCategory.ID, Name: strPtrOrNil("Milo")})
	create(&models.Goal{ContactID: source.ID, Name: "Run a marathon", Active: true})
	create(&models.Gift{ContactID: source.ID, Name: "Book", Type: "received"})
	create(&models.Call{ContactID: source.ID, AuthorID: &userID, AuthorName: "Tester", CalledAt: occurredAt, Type: "phone", WhoInitiated: "me"})
	create(&models.ContactAddress{ContactID: source.ID, AddressID: address.ID, DateFrom: &occurredAt})
	create(&models.ContactCompany{ContactID: source.ID, CompanyID: company.ID, JobPosition: strPtrOrNil("Designer")})
	create(&models.ContactLifeMetric{ContactID: source.ID, LifeMetricID: metric.ID, UserID: userID, CreatedAt: occurredAt})
	loan := models.Loan{VaultID: vaultID, Name: "Shared book", Type: "loan", Category: "item"}
	create(&loan)
	loanLink := models.ContactLoan{LoanID: loan.ID, LoanerID: source.ID, LoaneeID: third.ID}
	create(&loanLink)
	giftLink := models.ContactGift{LoanID: loan.ID, LoanerID: third.ID, LoaneeID: source.ID}
	create(&giftLink)
	var dateType models.ContactImportantDateType
	if err := svc.db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&dateType).Error; err != nil {
		t.Fatal(err)
	}
	birthYear, birthMonth, birthDay := 1991, 3, 2
	targetBirthday := models.ContactImportantDate{ContactID: target.ID, Label: "Birthday", ContactImportantDateTypeID: &dateType.ID}
	create(&targetBirthday)
	sourceBirthday := models.ContactImportantDate{ContactID: source.ID, Label: "Birthday from phone", Year: &birthYear, Month: &birthMonth, Day: &birthDay, DatePrecision: "full", ContactImportantDateTypeID: &dateType.ID}
	create(&sourceBirthday)
	reminder := models.ContactReminder{ContactID: source.ID, Label: "Birthday reminder", Type: "recurring_year", ImportantDateID: &sourceBirthday.ID}
	create(&reminder)
	var channel models.UserNotificationChannel
	if err := svc.db.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		t.Fatal(err)
	}
	schedule := models.ContactReminderScheduled{ContactReminderID: reminder.ID, UserNotificationChannelID: channel.ID, ScheduledAt: occurredAt}
	create(&schedule)
	file := models.File{VaultID: vaultID, UfileableID: &source.ID, UUID: "merge-avatar", MimeType: "image/png", Type: "avatar", Name: "Avatar"}
	create(&file)
	if err := svc.db.Model(&source).Update("file_id", file.ID).Error; err != nil {
		t.Fatal(err)
	}
	var relationType models.RelationshipType
	if err := svc.db.First(&relationType).Error; err != nil {
		t.Fatal(err)
	}
	externalVault, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "External relationships"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	externalContact := models.Contact{VaultID: externalVault.ID, FirstName: strPtrOrNil("External friend")}
	create(&externalContact)
	if err := svc.db.Model(&models.UserVault{}).Where("vault_id = ? AND user_id = ?", externalVault.ID, userID).Update("permission", models.PermissionViewer).Error; err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{source.ID, third.ID}, {third.ID, source.ID}, {target.ID, third.ID}, {target.ID, source.ID}, {source.ID, target.ID}, {source.ID, externalContact.ID}, {externalContact.ID, source.ID}} {
		create(&models.Relationship{ContactID: pair[0], RelatedContactID: pair[1], RelationshipTypeID: relationType.ID})
	}
	var informationType models.ContactInformationType
	if err := svc.db.First(&informationType).Error; err != nil {
		t.Fatal(err)
	}
	information := models.ContactInformation{ContactID: source.ID, TypeID: informationType.ID, Data: "alice@example.test"}
	create(&information)
	var template models.VaultQuickFactsTemplate
	if err := svc.db.Where("vault_id = ?", vaultID).First(&template).Error; err != nil {
		t.Fatal(err)
	}
	fact := models.QuickFact{ContactID: source.ID, VaultQuickFactsTemplateID: template.ID, Content: "Tea"}
	create(&fact)
	result, err := svc.MergeContacts(vaultID, userID, dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if result.FileID == nil || *result.FileID != file.ID {
		t.Fatalf("avatar not adopted: %+v", result)
	}
	for _, table := range []string{"task_contacts", "activity_participants", "contact_group", "contact_label", "contact_vault_user", "contact_information", "quick_facts", "contact_reminders", "contact_post", "pets", "goals", "gifts", "calls", "contact_address", "contact_companies", "contact_life_metric"} {
		var count int64
		if err := svc.db.Table(table).Where("contact_id = ?", source.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%s has %d stale source references", table, count)
		}
		if err := svc.db.Table(table).Where("contact_id = ?", target.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("%s target count %d, want 1", table, count)
		}
	}
	var membership models.ContactGroup
	if err := svc.db.Where("contact_id = ? AND group_id = ?", target.ID, group.ID).First(&membership).Error; err != nil {
		t.Fatal(err)
	}
	if membership.GroupTypeRoleID == nil || *membership.GroupTypeRoleID != role.ID {
		t.Fatal("source group role was not adopted")
	}
	if !result.IsFavorite {
		t.Fatal("response lost current user's favorite")
	}
	var history models.ContactVaultUser
	if err := svc.db.Where("contact_id = ? AND user_id = ?", target.ID, userID).First(&history).Error; err != nil {
		t.Fatal(err)
	}
	if history.NumberOfViews != 6 || !history.IsFavorite {
		t.Fatalf("user history lost: %+v", history)
	}
	for _, row := range []any{&sourceBirthday, &reminder, &schedule, &file, &activity, &third, &targetBirthday} {
		if err := svc.db.First(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if sourceBirthday.ContactID != target.ID || sourceBirthday.ContactImportantDateTypeID != nil || sourceBirthday.Label != "Birthday from phone" {
		t.Fatalf("conflicting date lost: %+v", sourceBirthday)
	}
	if targetBirthday.ContactImportantDateTypeID == nil || *targetBirthday.ContactImportantDateTypeID != dateType.ID {
		t.Fatal("target birthday changed")
	}
	if reminder.ContactID != target.ID || reminder.ImportantDateID == nil || *reminder.ImportantDateID != sourceBirthday.ID || schedule.ContactReminderID != reminder.ID {
		t.Fatal("reminder chain broken")
	}
	if file.UfileableID == nil || *file.UfileableID != target.ID || activity.PaidByContactID == nil || *activity.PaidByContactID != target.ID || third.FirstMetThroughContactID == nil || *third.FirstMetThroughContactID != target.ID {
		t.Fatal("file, payer or introducer was not redirected")
	}
	if sourceBirthday.Year == nil || *sourceBirthday.Year != birthYear || sourceBirthday.Month == nil || *sourceBirthday.Month != birthMonth || sourceBirthday.Day == nil || *sourceBirthday.Day != birthDay {
		t.Fatal("birthday date components changed")
	}
	if !schedule.ScheduledAt.Equal(occurredAt) || schedule.TriggeredAt != nil {
		t.Fatal("reminder schedule changed")
	}
	if err := svc.db.First(&loanLink, loanLink.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.First(&giftLink, giftLink.ID).Error; err != nil {
		t.Fatal(err)
	}
	if loanLink.LoanerID != target.ID || loanLink.LoaneeID != third.ID || giftLink.LoaneeID != target.ID || giftLink.LoanerID != third.ID {
		t.Fatal("loan or gift parties were not redirected")
	}
	var metricEvent models.ContactLifeMetric
	if err := svc.db.Where("contact_id = ?", target.ID).First(&metricEvent).Error; err != nil {
		t.Fatal(err)
	}
	if !metricEvent.CreatedAt.Equal(occurredAt) || metricEvent.UserID != userID {
		t.Fatal("metric event history changed")
	}
	var relations []models.Relationship
	if err := svc.db.Find(&relations).Error; err != nil {
		t.Fatal(err)
	}
	if len(relations) != 4 {
		t.Fatalf("expected deduplicated same-vault and cross-vault relationships, got %+v", relations)
	}
	for _, rel := range relations {
		if rel.ContactID == source.ID || rel.RelatedContactID == source.ID || rel.ContactID == rel.RelatedContactID {
			t.Fatalf("invalid relationship: %+v", rel)
		}
	}
}

func TestMergeContactsRejectsInvalidSelectionsAndRollsBack(t *testing.T) {
	for _, scenario := range []string{"self", "duplicate", "missing", "cross-vault", "viewer", "protected", "database-error"} {
		t.Run(scenario, func(t *testing.T) {
			svc, vaultID, userID, accountID := setupContactTest(t)
			target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Keep"})
			if err != nil {
				t.Fatal(err)
			}
			source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Source", Nickname: "Preserve"})
			if err != nil {
				t.Fatal(err)
			}
			note := models.Note{ContactID: source.ID, VaultID: vaultID, Body: "Unchanged"}
			if err := svc.db.Create(&note).Error; err != nil {
				t.Fatal(err)
			}
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			switch scenario {
			case "self":
				req.SourceContactIDs = []string{target.ID}
			case "duplicate":
				req.SourceContactIDs = append(req.SourceContactIDs, source.ID)
			case "missing":
				req.SourceContactIDs = append(req.SourceContactIDs, "missing")
			case "cross-vault":
				vault, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Other"}, "en")
				if err != nil {
					t.Fatal(err)
				}
				other, err := svc.CreateContact(vault.ID, userID, dto.CreateContactRequest{FirstName: "Other"})
				if err != nil {
					t.Fatal(err)
				}
				req.SourceContactIDs = append(req.SourceContactIDs, other.ID)
			case "viewer":
				if err := svc.db.Model(&models.UserVault{}).Where("vault_id = ? AND user_id = ?", vaultID, userID).Update("permission", models.PermissionViewer).Error; err != nil {
					t.Fatal(err)
				}
			case "protected":
				protected, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Protected"})
				if err != nil {
					t.Fatal(err)
				}
				if err := svc.db.Model(&models.Contact{}).Where("id = ?", protected.ID).Update("can_be_deleted", false).Error; err != nil {
					t.Fatal(err)
				}
				req.SourceContactIDs = append(req.SourceContactIDs, protected.ID)
			case "database-error":
				if err := svc.db.Exec("CREATE TRIGGER reject_contact_merge_feed BEFORE INSERT ON contact_feed_items BEGIN SELECT RAISE(ABORT, 'merge audit unavailable'); END").Error; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := svc.MergeContacts(vaultID, userID, req); err == nil {
				t.Fatal("invalid merge succeeded")
			}
			var active int64
			if err := svc.db.Model(&models.Contact{}).Where("id IN ?", []string{target.ID, source.ID}).Count(&active).Error; err != nil {
				t.Fatal(err)
			}
			if active != 2 {
				t.Fatalf("partial deletion: %d", active)
			}
			if err := svc.db.First(&note, note.ID).Error; err != nil {
				t.Fatal(err)
			}
			if note.ContactID != source.ID || note.Body != "Unchanged" {
				t.Fatalf("partial note migration: %+v", note)
			}
			var profile models.Contact
			if err := svc.db.First(&profile, "id = ?", target.ID).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(profile.Nickname) != "" {
				t.Fatal("profile change escaped rollback")
			}
			var snapshots int64
			if err := svc.db.Model(&models.Note{}).Where("source_type = ?", "contact_merge").Count(&snapshots).Error; err != nil {
				t.Fatal(err)
			}
			if snapshots != 0 {
				t.Fatal("snapshot escaped rollback")
			}
		})
	}
}

func TestMergeContactsSearchAndDAV(t *testing.T) {
	push, client, _, svc, vaultID, userID, _ := setupDavPushTest(t)
	engine, err := search.NewBleveEngine(t.TempDir() + "/merge.bleve")
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	searchService := NewSearchServiceWithDB(svc.db, engine)
	svc.SetSearchService(searchService)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alicia"})
	if err != nil {
		t.Fatal(err)
	}
	note := models.Note{ContactID: source.ID, VaultID: vaultID, Body: "orchard memories"}
	if err := svc.db.Create(&note).Error; err != nil {
		t.Fatal(err)
	}
	if err := searchService.IndexNote(&note); err != nil {
		t.Fatal(err)
	}
	sub := createPushSubscription(t, client, vaultID, userID, SyncWayPush)
	remoteURI := "https://dav.example.com/contacts/" + source.ID + ".vcf"
	state := models.ContactSubscriptionState{ContactID: source.ID, AddressBookSubscriptionID: sub.ID, DistantURI: remoteURI, DistantEtag: "source-etag"}
	if err := svc.db.Create(&state).Error; err != nil {
		t.Fatal(err)
	}
	events := make(chan string, 2)
	push.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
		removeAllFn: func(_ context.Context, path string) error {
			var count int64
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&count).Error; err != nil || count != 0 {
				t.Errorf("DAV deletion preceded commit: count=%d err=%v", count, err)
			}
			events <- "DELETE " + path
			return nil
		},
		putAddrObjFn: func(_ context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			if card.Value(vcard.FieldFormattedName) != "Alice" {
				t.Errorf("wrong merged vCard name: %s", card.Value(vcard.FieldFormattedName))
			}
			events <- "PUT " + path
			return &carddav.AddressObject{Path: path, ETag: "merged-etag"}, nil
		},
	}})
	svc.SetDavPushService(push)
	if _, err := svc.MergeContacts(vaultID, userID, dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"DELETE " + remoteURI, "PUT https://dav.example.com/contacts/" + target.ID + ".vcf"} {
		select {
		case actual := <-events:
			if actual != expected {
				t.Fatalf("DAV event %q, want %q", actual, expected)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("DAV merge lifecycle did not finish")
		}
	}
	// Wait for the worker to persist its final remote state and release its lock.
	release := push.operationLocks.lock(target.ID)
	release()
	var states []models.ContactSubscriptionState
	if err := svc.db.Find(&states).Error; err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 || states[0].ContactID != target.ID || states[0].DistantEtag != "merged-etag" {
		t.Fatalf("unexpected remote state: %+v", states)
	}
	result, err := searchService.Search(vaultID, "orchard", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notes) != 1 || result.Notes[0].ContactID != target.ID {
		t.Fatalf("note search retained old contact: %+v", result)
	}
	result, err = searchService.Search(vaultID, "Alicia", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Contacts) != 0 || len(result.Notes) != 1 || result.Notes[0].ContactID != target.ID {
		t.Fatalf("source search or snapshot indexing incorrect: %+v", result)
	}
}

func TestMergeContactsSnapshotsBeforeRedirectingRelationships(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	contacts := make([]*dto.ContactResponse, 0, 3)
	for _, name := range []string{"Primary", "First duplicate", "Second duplicate"} {
		contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: name})
		if err != nil {
			t.Fatal(err)
		}
		contacts = append(contacts, contact)
	}
	var relationType models.RelationshipType
	if err := svc.db.First(&relationType).Error; err != nil {
		t.Fatal(err)
	}
	relation := models.Relationship{ContactID: contacts[2].ID, RelatedContactID: contacts[1].ID, RelationshipTypeID: relationType.ID}
	if err := svc.db.Create(&relation).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, dto.MergeContactsRequest{TargetContactID: contacts[0].ID, SourceContactIDs: []string{contacts[1].ID, contacts[2].ID}}); err != nil {
		t.Fatal(err)
	}
	var snapshot models.Note
	if err := svc.db.Where("contact_id = ? AND source_type = ? AND source_uuid = ?", contacts[0].ID, "contact_merge", contacts[2].ID).First(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snapshot.Body, "Relationship from "+contacts[2].ID+" to "+contacts[1].ID) {
		t.Fatal("snapshot no longer contains the original relationship between sources")
	}
}
