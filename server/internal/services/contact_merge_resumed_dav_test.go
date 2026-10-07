package services

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/models"
)

func TestMergeContactsResumedDAVPreservesMergedRecords(t *testing.T) {
	push, client, cards, svc, vaultID, userID, _ := setupDavPushTest(t)
	sub := createPushSubscription(t, client, vaultID, userID, SyncWayBoth)
	if err := svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Updates(map[string]any{"active": false, "address_book_path": "/contacts/"}).Error; err != nil {
		t.Fatal(err)
	}
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	for _, row := range []any{&target, &source} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	year, month, day := 1990, 3, 2
	date := models.ContactImportantDate{ContactID: source.ID, Label: "Graduation", Year: &year, Month: &month, Day: &day, DatePrecision: "full"}
	if err := svc.db.Create(&date).Error; err != nil {
		t.Fatal(err)
	}
	reminder := models.ContactReminder{ContactID: source.ID, Label: "Remember graduation", Type: "recurring_year", ImportantDateID: &date.ID}
	if err := svc.db.Create(&reminder).Error; err != nil {
		t.Fatal(err)
	}
	var phoneType models.ContactInformationType
	if err := svc.db.Where("type = ?", "phone").First(&phoneType).Error; err != nil {
		t.Fatal(err)
	}
	phone := models.ContactInformation{ContactID: source.ID, TypeID: phoneType.ID, Data: "+15550101999", Kind: strPtrOrNil("mobile"), Pref: true}
	if err := svc.db.Create(&phone).Error; err != nil {
		t.Fatal(err)
	}

	scheduledAt := time.Date(2030, 3, 2, 9, 0, 0, 0, time.UTC)
	var channel models.UserNotificationChannel
	if err := svc.db.Where("user_id = ?", userID).First(&channel).Error; err != nil {
		t.Fatal(err)
	}
	schedule := models.ContactReminderScheduled{ContactReminderID: reminder.ID, UserNotificationChannelID: channel.ID, ScheduledAt: scheduledAt}
	address := models.Address{VaultID: vaultID, Line1: strPtrOrNil("Synthetic garden road"), Line2: strPtrOrNil("Apartment A")}
	for _, row := range []any{&schedule, &address} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	addressLink := models.ContactAddress{ContactID: source.ID, AddressID: address.ID, DateFrom: &scheduledAt}
	if err := svc.db.Create(&addressLink).Error; err != nil {
		t.Fatal(err)
	}
	svc.SetDavPushService(push)
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	// The advertised pause/merge/resume flow has no mappings at merge time.
	if err := svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("active", true).Error; err != nil {
		t.Fatal(err)
	}
	var remote carddav.AddressObject
	remoteToken := "remote-token"
	queryFallback := false
	readRemote := func() ([]carddav.AddressObject, error) {
		// Exercise the wire representation, not just a shared in-memory Card.
		var encoded bytes.Buffer
		if err := vcard.NewEncoder(&encoded).Encode(remote.Card); err != nil {
			return nil, err
		}
		decoded, err := vcard.NewDecoder(&encoded).Decode()
		if err != nil {
			return nil, err
		}
		return []carddav.AddressObject{{Path: remote.Path, ETag: remote.ETag, Card: decoded}}, nil
	}
	remoteClient := &mockCardDAVClient{
		putAddrObjFn: func(_ context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			remote = carddav.AddressObject{Path: path, ETag: "after-push", Card: card}
			return &remote, nil
		},
		syncCollFn: func(_ context.Context, _ string, _ *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			if queryFallback {
				return nil, errors.New("sync collection unsupported")
			}
			return &carddav.SyncResponse{SyncToken: remoteToken, Updated: []carddav.AddressObject{remote}}, nil
		},
		multiGetFn: func(_ context.Context, _ string, _ *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
			return readRemote()
		},
		queryFn: func(_ context.Context, _ string, _ *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
			return readRemote()
		},
	}
	push.SetClientFactory(&mockCardDAVClientFactory{client: remoteClient})
	push.PushContactChange(target.ID, vaultID)
	if remote.Card == nil {
		t.Fatal("resumed push did not execute")
	}
	var state models.ContactSubscriptionState
	if err := svc.db.Where("contact_id = ? AND address_book_subscription_id = ?", target.ID, sub.ID).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.DistantEtag != "after-push" {
		t.Fatalf("push did not save mapping: %+v", state)
	}
	remote.Card.SetValue(vcard.FieldNickname, "Remote nickname")
	// Ordinary additions and parameter normalization must not block a resumed pull.
	remote.Card.SetValue(vcard.FieldBirthday, "2000-01-01")
	remote.Card.SetValue(vcard.FieldEmail, "alice@example.test")
	remote.Card[vcard.FieldTelephone][0].Params[vcard.ParamType] = []string{"VOICE", "CELL"}
	remote.ETag = "after-nickname-edit"
	syncer := NewDavSyncService(svc.db, client, cards)
	syncer.SetClientFactory(&mockCardDAVClientFactory{client: remoteClient})
	result, err := syncer.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors != 0 || result.Updated != 1 {
		t.Fatalf("pull did not successfully update: %+v", result)
	}
	var retainedDate models.ContactImportantDate
	dateErr := svc.db.First(&retainedDate, date.ID).Error
	var storedDate models.ContactImportantDate
	if err := svc.db.Unscoped().First(&storedDate, date.ID).Error; err != nil {
		t.Fatal(err)
	}
	if dateErr != nil || retainedDate.ContactID != target.ID {
		t.Errorf("resumed DAV pull deleted merged date: id=%d read_error=%v deleted_at=%v", date.ID, dateErr, storedDate.DeletedAt)
	}
	if err := svc.db.First(&reminder, reminder.ID).Error; err != nil {
		t.Fatal(err)
	}
	var activeReferencedDates int64
	if err := svc.db.Model(&models.ContactImportantDate{}).Where("id = ?", reminder.ImportantDateID).Count(&activeReferencedDates).Error; err != nil {
		t.Fatal(err)
	}
	if activeReferencedDates != 1 {
		t.Errorf("reminder %d points to inactive merged date %d", reminder.ID, *reminder.ImportantDateID)
	}
	var phones []models.ContactInformation
	if err := svc.db.Where("contact_id = ? AND type_id = ?", target.ID, phoneType.ID).Find(&phones).Error; err != nil {
		t.Fatal(err)
	}
	if len(phones) != 1 || phones[0].ID != phone.ID || ptrToStr(phones[0].Kind) != "mobile" {
		t.Errorf("resumed DAV pull replaced typed phone: original_id=%d stored=%+v", phone.ID, phones)
	}
	if err := svc.db.First(&target, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(target.Nickname) != "Remote nickname" {
		t.Fatal("safe remote nickname edit not applied")
	}
	assertRetained := func() {
		t.Helper()
		if err := svc.db.First(&date, date.ID).Error; err != nil {
			t.Fatal(err)
		}
		if date.ContactID != target.ID || date.Year == nil || *date.Year != year || date.Label != "Graduation" {
			t.Fatal("merged date contents changed")
		}
		if err := svc.db.First(&phone, phone.ID).Error; err != nil {
			t.Fatal(err)
		}
		if phone.ContactID != target.ID || phone.Data != "+15550101999" || ptrToStr(phone.Kind) != "mobile" {
			t.Fatal("typed phone contents changed")
		}
		if err := svc.db.First(&addressLink, addressLink.ID).Error; err != nil {
			t.Fatal(err)
		}
		if addressLink.ContactID != target.ID || addressLink.DateFrom == nil || !addressLink.DateFrom.Equal(scheduledAt) {
			t.Fatal("address history changed")
		}
		if err := svc.db.First(&address, address.ID).Error; err != nil {
			t.Fatal(err)
		}
		if ptrToStr(address.Line1) != "Synthetic garden road" || ptrToStr(address.Line2) != "Apartment A" {
			t.Fatal("address details changed")
		}
		if err := svc.db.First(&schedule, schedule.ID).Error; err != nil {
			t.Fatal(err)
		}
		if schedule.ContactReminderID != reminder.ID || !schedule.ScheduledAt.Equal(scheduledAt) || schedule.TriggeredAt != nil {
			t.Fatal("reminder schedule changed")
		}
	}
	assertRetained()
	var birthdays int64
	if err := svc.db.Model(&models.ContactImportantDate{}).Where("contact_id = ? AND label = ?", target.ID, "Birthdate").Count(&birthdays).Error; err != nil || birthdays != 1 {
		t.Fatalf("new birthday missing: %d, %v", birthdays, err)
	}
	var emails []models.ContactInformation
	if err := svc.db.Where("contact_id = ? AND data = ?", target.ID, "alice@example.test").Find(&emails).Error; err != nil || len(emails) != 1 {
		t.Fatalf("new email missing: %+v, %v", emails, err)
	}
	var acceptedSub models.AddressBookSubscription
	if err := svc.db.First(&acceptedSub, "id = ?", sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	acceptedCard := remote.Card
	acceptedETag := "after-nickname-edit"
	for _, mode := range []string{"incremental", "full", "query"} {
		queryFallback = mode == "query"
		checkpoint := "remote-token"
		if mode != "incremental" {
			checkpoint = ""
		}
		if err := svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("distant_sync_token", strPtrOrNil(checkpoint)).Error; err != nil {
			t.Fatal(err)
		}
		acceptedSub.DistantSyncToken = strPtrOrNil(checkpoint)
		for _, field := range []string{vcard.FieldAddress} {
			t.Run(mode+"_reject_"+field, func(t *testing.T) {
				remote.Card = make(vcard.Card)
				for key, values := range acceptedCard {
					remote.Card[key] = values
				}
				remote.Card.SetValue(vcard.FieldNickname, "Uncommitted nickname")
				switch field {
				case vcard.FieldAddress:
					remote.Card.SetValue(field, ";;Different street;City;Region;12345;Country")
				}
				remote.ETag = "rejected-" + mode + "-" + field
				remoteToken = "unaccepted-token"
				result, err := syncer.SyncSubscription(context.Background(), sub.ID, vaultID)
				if err != nil {
					t.Fatal(err)
				}
				if result.Errors != 1 || result.Updated != 0 {
					t.Fatalf("unsafe pull was not rejected: %+v", result)
				}
				if err := svc.db.First(&target, "id = ?", target.ID).Error; err != nil {
					t.Fatal(err)
				}
				if ptrToStr(target.Nickname) != "Remote nickname" || ptrToStr(target.DistantEtag) != acceptedETag {
					t.Fatal("rejected pull partially committed profile or etag")
				}
				if err := svc.db.First(&state, state.ID).Error; err != nil {
					t.Fatal(err)
				}
				if state.DistantEtag != acceptedETag {
					t.Fatal("rejected pull acknowledged remote mapping")
				}
				var storedSub models.AddressBookSubscription
				if err := svc.db.First(&storedSub, "id = ?", sub.ID).Error; err != nil {
					t.Fatal(err)
				}
				if ptrToStr(storedSub.DistantSyncToken) != ptrToStr(acceptedSub.DistantSyncToken) || storedSub.LastSynchronizedAt == nil || acceptedSub.LastSynchronizedAt == nil || !storedSub.LastSynchronizedAt.Equal(*acceptedSub.LastSynchronizedAt) {
					t.Error("rejected pull advanced sync checkpoint and will not be retried")
				}
				var logs []models.DavSyncLog
				if err := svc.db.Where("address_book_subscription_id = ? AND distant_etag = ? AND action = ?", sub.ID, remote.ETag, "error").Find(&logs).Error; err != nil {
					t.Fatal(err)
				}
				if len(logs) != 1 || !strings.Contains(ptrToStr(logs[0].ErrorMessage), "edit") || !strings.Contains(ptrToStr(logs[0].ErrorMessage), "Bonds") {
					t.Fatalf("missing actionable sync error: %+v", logs)
				}
				assertRetained()
			})
		}

		// A failure is recoverable without dropping mappings or restarting the
		// subscription. A safe subsequent representation must advance the checkpoint.
		remote.Card = acceptedCard
		remote.ETag = "recovered-" + mode
		remoteToken = "recovered-token-" + mode
		result, err := syncer.SyncSubscription(context.Background(), sub.ID, vaultID)
		if err != nil || result.Errors != 0 || result.Updated != 1 {
			t.Fatalf("safe retry failed: %+v, %v", result, err)
		}
		acceptedETag = remote.ETag
		if err := svc.db.First(&acceptedSub, "id = ?", sub.ID).Error; err != nil {
			t.Fatal(err)
		}
		if !queryFallback && ptrToStr(acceptedSub.DistantSyncToken) != remoteToken {
			t.Fatal("successful retry did not advance checkpoint")
		}
		if err := svc.db.First(&state, state.ID).Error; err != nil || state.DistantEtag != acceptedETag {
			t.Fatalf("successful retry did not acknowledge mapping: %v", err)
		}
		assertRetained()
	}
}
