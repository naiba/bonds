package services

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestContactMergeRejectsConcurrentInformationEdit(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	var kind models.ContactInformationType
	if err := svc.db.Where("account_id = ? AND type = ?", accountID, "phone").First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	infos := NewContactInformationService(svc.db)
	info, err := infos.Create(source.ID, vaultID, dto.CreateContactInformationRequest{TypeID: kind.ID, Data: "old-phone"})
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	loaded, resume := make(chan struct{}), make(chan struct{})
	var intercepted atomic.Bool
	callback := "independent:pause_loaded_information"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(callback, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.ContactInformation); ok && row.ID == info.ID && intercepted.CompareAndSwap(false, true) {
			close(loaded)
			select {
			case <-resume:
			case <-time.After(15 * time.Second):
				tx.AddError(errors.New("information edit resume timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(callback)
	done := make(chan error, 1)
	go func() {
		_, err := infos.Update(info.ID, source.ID, vaultID, dto.UpdateContactInformationRequest{TypeID: kind.ID, Data: "edited-phone"})
		done <- err
	}()
	select {
	case <-loaded:
	case <-time.After(15 * time.Second):
		close(resume)
		t.Fatal("information query not intercepted")
	}
	_, mergeErr := svc.MergeContacts(vaultID, userID, req)
	close(resume)
	if mergeErr != nil {
		t.Fatal(mergeErr)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrContactInformationNotFound) {
			t.Fatalf("stale edit must be rejected: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("information update timeout")
	}
	var actual models.ContactInformation
	if err := svc.db.First(&actual, info.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.ContactID != target.ID || actual.Data != "old-phone" {
		t.Fatalf("successful edit reattached migrated information to deleted source: contact=%s source=%s target=%s data=%s", actual.ContactID, source.ID, target.ID, actual.Data)
	}
}

func TestContactMergeHistoricalPayerRequiresOwningVaultPermission(t *testing.T) {
	svc, oldVaultID, userID, accountID := setupContactTest(t)
	vault, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Editable"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(oldVaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.CreateContact(vault.ID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	// PaidByContactID is preserved from the legacy life_events migration.
	neighbor := models.Contact{VaultID: oldVaultID, FirstName: strPtrOrNil("Historical participant")}
	if err := svc.db.Create(&neighbor).Error; err != nil {
		t.Fatal(err)
	}
	activity := models.Activity{VaultID: oldVaultID, Title: "Historical dinner", PaidByContactID: &source.ID}
	if err := svc.db.Create(&activity).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&models.ActivityParticipant{ActivityID: activity.ID, ContactID: source.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&models.ActivityParticipant{ActivityID: activity.ID, ContactID: neighbor.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactMoveService(svc.db).Move(source.ID, oldVaultID, vault.ID, userID); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.UserVault{}).Where("vault_id = ? AND user_id = ?", oldVaultID, userID).Update("permission", models.PermissionViewer).Error; err != nil {
		t.Fatal(err)
	}
	req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
	preview, err := svc.PreviewContactMerge(vault.ID, userID, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Blockers) != 1 || preview.Blockers[0] != "incoming_permission" {
		t.Fatalf("missing permission blocker: %v", preview.Blockers)
	}
	req.ReviewToken = preview.ReviewToken
	_, err = svc.MergeContacts(vault.ID, userID, req)
	var actual models.Activity
	if loadErr := svc.db.First(&actual, activity.ID).Error; loadErr != nil {
		t.Fatal(loadErr)
	}
	if !errors.Is(err, ErrContactMergeBlocked) || actual.PaidByContactID == nil || *actual.PaidByContactID != source.ID {
		t.Fatalf("merge mutated read-only vault payer: err=%v vault=%s paid_by=%s source=%s target=%s", err, actual.VaultID, ptrToStr(actual.PaidByContactID), source.ID, target.ID)
	}
}

func TestContactMergePayerReviewScopeAndFreshness(t *testing.T) {
	for _, scenario := range []struct {
		name                                  string
		sourcePayer, revoke, replaceReference bool
		permission                            int
		blocked                               bool
	}{
		{name: "unchanged_target_viewer", permission: models.PermissionViewer},
		{name: "unchanged_target_no_membership"},
		{name: "source_no_membership", sourcePayer: true, blocked: true},
		{name: "source_editor", sourcePayer: true, permission: models.PermissionEditor},
		{name: "permission_revoked", sourcePayer: true, permission: models.PermissionEditor, revoke: true, blocked: true},
		{name: "same_count_changed_reference", sourcePayer: true, permission: models.PermissionEditor, replaceReference: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			svc, vaultID, userID, accountID := setupContactTest(t)
			external, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Historical activities"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			for _, c := range []*models.Contact{&target, &source} {
				if err := svc.db.Create(c).Error; err != nil {
					t.Fatal(err)
				}
			}
			payer := target.ID
			if scenario.sourcePayer {
				payer = source.ID
			}
			activity := models.Activity{VaultID: external.ID, Title: "Dinner", PaidByContactID: &payer}
			if err := svc.db.Create(&activity).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.First(&activity, activity.ID).Error; err != nil {
				t.Fatal(err)
			}
			membership := svc.db.Model(&models.UserVault{}).Where("vault_id = ? AND user_id = ?", external.ID, userID)
			if scenario.permission == 0 {
				err = membership.Delete(&models.UserVault{}).Error
			} else {
				err = membership.Update("permission", scenario.permission).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			preview, err := svc.PreviewContactMerge(vaultID, userID, req)
			if err != nil {
				t.Fatal(err)
			}
			if scenario.blocked && !scenario.revoke {
				if len(preview.Blockers) != 1 || preview.Blockers[0] != "incoming_permission" {
					t.Fatalf("blockers=%v", preview.Blockers)
				}
			} else if len(preview.Blockers) != 0 {
				t.Fatalf("unexpected blockers: %v", preview.Blockers)
			}
			expectedPayers := int64(0)
			if scenario.sourcePayer {
				expectedPayers = 1
			}
			if preview.Effects["payers"] != expectedPayers {
				t.Fatalf("payer effects=%d want=%d", preview.Effects["payers"], expectedPayers)
			}
			req.ReviewToken = preview.ReviewToken
			if scenario.revoke {
				if err := membership.Update("permission", models.PermissionViewer).Error; err != nil {
					t.Fatal(err)
				}
			}
			if scenario.replaceReference {
				if err := svc.db.Delete(&activity).Error; err != nil {
					t.Fatal(err)
				}
				activity.ID = 0
				if err := svc.db.Create(&activity).Error; err != nil {
					t.Fatal(err)
				}
			}
			_, err = svc.MergeContacts(vaultID, userID, req)
			switch {
			case scenario.blocked:
				if !errors.Is(err, ErrContactMergeBlocked) {
					t.Fatalf("expected blocked, got %v", err)
				}
			case scenario.replaceReference:
				if !errors.Is(err, ErrContactMergeReviewChanged) {
					t.Fatalf("expected changed review, got %v", err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
			var actual models.Activity
			if err := svc.db.First(&actual, activity.ID).Error; err != nil {
				t.Fatal(err)
			}
			expectedPayer := payer
			if !scenario.blocked && !scenario.replaceReference {
				expectedPayer = target.ID
			}
			if ptrToStr(actual.PaidByContactID) != expectedPayer || actual.VaultID != external.ID || actual.Title != "Dinner" {
				t.Fatalf("unexpected activity: %+v", actual)
			}
			if !scenario.sourcePayer && !actual.UpdatedAt.Equal(activity.UpdatedAt) {
				t.Fatal("unchanged target reference was written")
			}
			var sourceCount int64
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&sourceCount).Error; err != nil {
				t.Fatal(err)
			}
			expectedCount := int64(0)
			if scenario.blocked || scenario.replaceReference {
				expectedCount = 1
			}
			if sourceCount != expectedCount {
				t.Fatalf("source count=%d want=%d", sourceCount, expectedCount)
			}
		})
	}
}

func TestContactMergeRejectsConcurrentGoalEdit(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	goals := NewGoalService(svc.db)
	goal, err := goals.Create(source.ID, vaultID, dto.CreateGoalRequest{Name: "Learn"})
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	loaded, resume := make(chan struct{}), make(chan struct{})
	var intercepted atomic.Bool
	callback := "independent:pause_loaded_information"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(callback, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.Goal); ok && row.ID == goal.ID && intercepted.CompareAndSwap(false, true) {
			close(loaded)
			select {
			case <-resume:
			case <-time.After(15 * time.Second):
				tx.AddError(errors.New("information edit resume timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(callback)
	done := make(chan error, 1)
	go func() {
		_, err := goals.Update(goal.ID, source.ID, vaultID, dto.UpdateGoalRequest{Name: "Edited"})
		done <- err
	}()
	select {
	case <-loaded:
	case <-time.After(15 * time.Second):
		close(resume)
		t.Fatal("information query not intercepted")
	}
	_, mergeErr := svc.MergeContacts(vaultID, userID, req)
	close(resume)
	if mergeErr != nil {
		t.Fatal(mergeErr)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrGoalNotFound) {
			t.Fatalf("stale edit must be rejected: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("information update timeout")
	}
	var actual models.Goal
	if err := svc.db.First(&actual, goal.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.ContactID != target.ID || actual.Name != "Learn" {
		t.Fatalf("successful edit reattached migrated information to deleted source: contact=%s source=%s target=%s data=%s", actual.ContactID, source.ID, target.ID, actual.Name)
	}
	disabled := false
	if _, err := goals.Update(goal.ID, target.ID, vaultID, dto.UpdateGoalRequest{Name: "Edited", Active: &disabled}); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.First(&actual, goal.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Name != "Edited" || actual.Active {
		t.Fatalf("ordinary update lost zero value: %+v", actual)
	}
}

func TestContactMergeRejectsConcurrentContactEdit(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	loaded, resume := make(chan struct{}), make(chan struct{})
	var intercepted atomic.Bool
	callback := "independent:pause_loaded_information"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(callback, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.Contact); ok && row.ID == source.ID && intercepted.CompareAndSwap(false, true) {
			close(loaded)
			select {
			case <-resume:
			case <-time.After(15 * time.Second):
				tx.AddError(errors.New("information edit resume timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(callback)
	done := make(chan error, 1)
	go func() {
		_, err := svc.ToggleArchive(source.ID, vaultID, userID)
		done <- err
	}()
	select {
	case <-loaded:
	case <-time.After(15 * time.Second):
		close(resume)
		t.Fatal("information query not intercepted")
	}
	_, mergeErr := svc.MergeContacts(vaultID, userID, req)
	close(resume)
	if mergeErr != nil {
		t.Fatal(mergeErr)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrContactNotFound) {
			t.Fatalf("stale edit must be rejected: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("information update timeout")
	}
	var liveSources int64
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&liveSources).Error; err != nil {
		t.Fatal(err)
	}
	if liveSources != 0 {
		t.Fatal("stale edit resurrected merged source")
	}
}
