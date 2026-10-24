package services

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardConcurrentCompanyResolution(t *testing.T) {
	svc, vault, user, account := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("requires PostgreSQL concurrent transactions and lock observation")
	}
	other, err := NewVaultService(svc.db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Independent vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for i := 0; i < 3; i++ {
		c, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: fmt.Sprintf("Synthetic employee %d", i)})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.ID)
	}
	otherContact, err := svc.CreateContact(other.ID, user, dto.CreateContactRequest{FirstName: "Other vault employee"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	lookedUp, resume, secondLookup := make(chan struct{}), make(chan struct{}), make(chan struct{}, 1)
	var first atomic.Bool
	const hook = "employment:pause_automatic_company_lookup"
	// Pause after the real first SELECT saw no company. The second transaction
	// must wait before its own SELECT, or both writers would create an identity.
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table != "companies" {
			return
		}
		if _, ok := tx.Statement.Dest.(*[]models.Company); !ok {
			return
		}
		if tx.Statement.Context.Value(companyResolutionTestContact{}) == ids[0] && first.CompareAndSwap(false, true) {
			close(lookedUp)
			select {
			case <-resume:
			case <-ctx.Done():
				tx.AddError(ctx.Err())
			}
		} else if tx.Statement.Context.Value(companyResolutionTestContact{}) == ids[1] {
			select {
			case secondLookup <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(hook)
	apply := func(id, vaultID string, pid chan<- int) error {
		return svc.db.WithContext(context.WithValue(ctx, companyResolutionTestContact{}, id)).Transaction(func(tx *gorm.DB) error {
			if pid != nil {
				var n int
				if err := tx.Raw("SELECT pg_backend_pid()").Scan(&n).Error; err != nil {
					return err
				}
				pid <- n
			}
			card := vcard.Card{vcard.FieldOrganization: []*vcard.Field{vcard.NewOrganization([]string{"Shared new employer"})}, vcard.FieldTitle: []*vcard.Field{{Value: "Engineer"}}}
			return ImportContactVCardFields(tx, card, id, vaultID, account)
		})
	}
	doneFirst, doneSecond := make(chan error, 1), make(chan error, 1)
	go func() { doneFirst <- apply(ids[0], vault, nil) }()
	select {
	case <-lookedUp:
	case <-ctx.Done():
		close(resume)
		<-doneFirst
		t.Fatal("first lookup not reached")
	}
	// Different vault must commit while the first transaction is paused.
	otherErr := apply(otherContact.ID, other.ID, nil)
	if otherErr != nil {
		close(resume)
		<-doneFirst
		t.Fatalf("other vault blocked or failed: %v", otherErr)
	}
	pid := make(chan int, 1)
	go func() { doneSecond <- apply(ids[1], vault, pid) }()
	var secondPID int
	select {
	case secondPID = <-pid:
	case err := <-doneSecond:
		close(resume)
		<-doneFirst
		t.Fatalf("second transaction failed before start: %v", err)
	}
	waiting := false
	observationErr := error(nil)
	ticker := time.NewTicker(10 * time.Millisecond)
observe:
	for {
		select {
		case <-secondLookup:
			break observe // Deterministic pre-fix race, keep going to assert actual data.
		case <-ctx.Done():
			observationErr = ctx.Err()
			break observe
		case <-ticker.C:
			var count int64
			if err := svc.db.WithContext(ctx).Raw("SELECT count(*) FROM pg_locks WHERE pid = ? AND locktype = 'advisory' AND NOT granted", secondPID).Scan(&count).Error; err != nil {
				observationErr = err
				break observe
			}
			if count > 0 {
				waiting = true
				break observe
			}
		}
	}
	ticker.Stop()
	// Before the fix let the second writer finish while the first still has its
	// empty lookup result; after the fix release the holder so the waiter proceeds.
	var secondErr error
	if !waiting {
		secondErr = <-doneSecond
	}
	close(resume)
	firstErr := <-doneFirst
	if waiting {
		secondErr = <-doneSecond
	}
	if observationErr != nil || firstErr != nil || secondErr != nil {
		t.Fatalf("concurrent operations: observation=%v first=%v second=%v", observationErr, firstErr, secondErr)
	}
	if !waiting {
		t.Error("concurrent automatic resolution did not wait for the in-flight identity")
	}
	if err := apply(ids[2], vault, nil); err != nil {
		t.Errorf("subsequent unambiguous ORG rejected: %v", err)
	}
	var companies []models.Company
	if err := svc.db.Where("name = ?", "Shared new employer").Find(&companies).Error; err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	companyIDs := map[string]uint{}
	for _, c := range companies {
		counts[c.VaultID]++
		companyIDs[c.VaultID] = c.ID
	}
	if counts[vault] != 1 || counts[other.ID] != 1 {
		t.Errorf("automatic creation duplicated identities: counts=%v", counts)
	}
	for _, id := range ids {
		var jobs []models.ContactCompany
		if err := svc.db.Where("contact_id = ?", id).Find(&jobs).Error; err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 || jobs[0].CompanyID != companyIDs[vault] {
			t.Errorf("contact %s did not share resolved company (jobs=%d)", id, len(jobs))
		}
	}
	// Explicit same-name companies remain legal; a new unassociated input then
	// receives the established recoverable error without partial Jobs writes.
	duplicate := models.Company{VaultID: vault, Name: "Shared new employer"}
	if err := svc.db.Create(&duplicate).Error; err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Ambiguous employee"})
	if err != nil {
		t.Fatal(err)
	}
	if err := apply(fresh.ID, vault, nil); !errors.Is(err, ErrVCardUnsafeReplacement) {
		t.Fatalf("ambiguous explicit identities: %v", err)
	}
	var jobs int64
	if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ?", fresh.ID).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if jobs != 0 {
		t.Fatal("rejected resolution retained a job")
	}
}

type companyResolutionTestContact struct{}
