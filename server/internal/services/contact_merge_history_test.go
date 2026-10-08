package services

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestContactMergePreservesConcurrentViewCounts(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	for id, views := range map[string]uint{target.ID: 10, source.ID: 20} {
		if err := svc.db.Model(&models.ContactVaultUser{}).Where("contact_id = ? AND user_id = ?", id, userID).Update("number_of_views", views).Error; err != nil {
			t.Fatal(err)
		}
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	loaded, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	callback := "merge:pause_view_history"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(callback, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.ContactVaultUser); ok && row.ContactID == target.ID && paused.CompareAndSwap(false, true) {
			close(loaded)
			select {
			case <-resume:
			case <-time.After(15 * time.Second):
				tx.AddError(errors.New("view resume timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(callback)
	viewed := make(chan error, 1)
	go func() { _, err := svc.GetContact(target.ID, userID, vaultID); viewed <- err }()
	select {
	case <-loaded:
	case <-time.After(15 * time.Second):
		close(resume)
		t.Fatal("view did not load")
	}
	merged := make(chan error, 1)
	go func() { _, err := svc.MergeContacts(vaultID, userID, req); merged <- err }()
	var mergeErr error
	completed := false
	// PostgreSQL must wait for the active viewer; SQLite may reject the merge
	// and let the caller retry. Neither outcome may discard a successful view.
	select {
	case mergeErr = <-merged:
		completed = true
	case <-time.After(2 * time.Second):
	}
	close(resume)
	select {
	case err := <-viewed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("view did not finish")
	}
	if !completed {
		select {
		case mergeErr = <-merged:
		case <-time.After(15 * time.Second):
			t.Fatal("merge did not finish")
		}
	}
	if mergeErr != nil {
		if svc.db.Dialector.Name() != "sqlite" || !strings.Contains(mergeErr.Error(), "locked") {
			t.Fatal(mergeErr)
		}
		req = reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
		if _, err := svc.MergeContacts(vaultID, userID, req); err != nil {
			t.Fatal(err)
		}
	}
	var history []models.ContactVaultUser
	if err := svc.db.Where("contact_id IN ? AND user_id = ?", []string{target.ID, source.ID}, userID).Find(&history).Error; err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].ContactID != target.ID || history[0].NumberOfViews != 31 {
		t.Fatalf("successful view was lost: %+v", history)
	}
}

func TestContactMergeListsEverySharedAddressHistory(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	for _, c := range []*models.Contact{&target, &source} {
		if err := svc.db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	address := models.Address{VaultID: vaultID, City: strPtrOrNil("London")}
	if err := svc.db.Create(&address).Error; err != nil {
		t.Fatal(err)
	}
	old := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range []models.ContactAddress{{ContactID: target.ID, AddressID: address.ID, DateFrom: &old, IsPastAddress: true}, {ContactID: source.ID, AddressID: address.ID, DateFrom: &recent}} {
		if err := svc.db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	if _, err := svc.MergeContacts(vaultID, userID, req); err != nil {
		t.Fatal(err)
	}
	rows, err := NewAddressService(svc.db).List(target.ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("shared address history is hidden: %+v", rows)
	}
	if rows[0].DateFrom == nil || rows[1].DateFrom == nil || rows[0].DateFrom.Equal(*rows[1].DateFrom) {
		t.Fatalf("distinct history lost: %+v", rows)
	}
}
