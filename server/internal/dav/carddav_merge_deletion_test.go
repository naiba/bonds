package dav

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

func TestCardDAVDeletionUsesMergeContactLockOrder(t *testing.T) {
	backend, db, ctx, vault, user := setupCardDAVTest(t)
	if db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL DAV deletion parent row locks")
	}
	contacts := []*models.Contact{}
	for _, name := range []string{"Source", "Target", "Referrer"} {
		contacts = append(contacts, createTestContact(t, db, vault, user, name, ""))
	}
	sort.Slice(contacts, func(i, j int) bool { return contacts[i].ID < contacts[j].ID })
	source, target, owner := contacts[0], contacts[1], contacts[2]
	svc := services.NewContactService(db)
	if _, err := svc.UpdateContact(owner.ID, vault, user, dto.UpdateContactRequest{FirstName: *owner.FirstName, FirstMetThroughContactID: &source.ID}); err != nil {
		t.Fatal(err)
	}
	req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}, FieldChoices: map[string]string{"first_name": target.ID}}
	preview, err := svc.PreviewContactMerge(vault, user, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Blockers) != 0 {
		t.Fatalf("invalid fixture: %+v", preview)
	}
	req.ReviewToken = preview.ReviewToken
	cleared, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	hook := "dav_delete:pause_introducer_cleanup"
	if err := db.Callback().Update().After("gorm:update").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "contacts" && strings.Contains(tx.Statement.SQL.String(), "first_met_through_contact_id") && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(cleared)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("DAV deletion scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Update().Remove(hook)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	backend = NewCardDAVBackend(db.WithContext(ctx))
	deleted := make(chan error, 1)
	go func() {
		deleted <- backend.DeleteAddressObject(ctx, fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, source.ID))
	}()
	select {
	case <-cleared:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("DAV deletion did not clear referrer")
	}
	merged := make(chan error, 1)
	go func() {
		_, err := services.NewContactService(db.WithContext(ctx)).MergeContacts(vault, user, req)
		merged <- err
	}()
	waited := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		if err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE '%contacts%FOR UPDATE%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='contacts'::regclass)`).Scan(&count).Error; err != nil {
			close(resume)
			t.Fatal(err)
		}
		if count > 0 {
			waited = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	close(resume)
	var deleteErr, mergeErr error
	select {
	case deleteErr = <-deleted:
	case <-time.After(15 * time.Second):
		t.Fatal("DAV deletion did not finish")
	}
	select {
	case mergeErr = <-merged:
	case <-time.After(15 * time.Second):
		t.Fatal("merge did not finish")
	}
	if deleteErr != nil || !errors.Is(mergeErr, services.ErrContactNotFound) {
		t.Fatalf("DAV delete=%v merge=%v", deleteErr, mergeErr)
	}
	if !waited {
		t.Fatal("merge did not wait on DAV deletion")
	}
	var stored models.Contact
	if err := db.First(&stored, "id = ?", owner.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.FirstMetThroughContactID != nil {
		t.Fatal("DAV deletion left introducer reference")
	}
	var sourceCount, targetCount int64
	if err := db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&sourceCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", target.ID).Count(&targetCount).Error; err != nil {
		t.Fatal(err)
	}
	if sourceCount != 0 || targetCount != 1 {
		t.Fatalf("unexpected contact state: %d/%d", sourceCount, targetCount)
	}
}

func TestCardDAVDeletionReturnsConflictForNewIntroducerReference(t *testing.T) {
	backend, db, ctx, vault, user := setupCardDAVTest(t)
	if db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL reference creation during DAV deletion discovery")
	}
	source := createTestContact(t, db, vault, user, "Introducer", "")
	discovered, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	hook := "dav_delete:pause_reference_discovery"
	if err := db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*[]string); !ok {
			return
		}
		if tx.Statement.Table == "contacts" && strings.Contains(tx.Statement.SQL.String(), "first_met_through_contact_id") && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(discovered)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("DAV discovery timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Query().Remove(hook)
	handler := carddav.Handler{Backend: backend, Prefix: "/dav"}
	path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, source.ID)
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil).WithContext(ctx))
		result <- recorder
	}()
	select {
	case <-discovered:
	case early := <-result:
		close(resume)
		t.Fatalf("DAV request returned before discovery: %d %s", early.Code, early.Body.String())
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("DAV deletion did not discover referrers")
	}
	late, err := services.NewContactService(db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "New referrer", FirstMetThroughContactID: &source.ID})
	close(resume)
	if err != nil {
		t.Fatal(err)
	}
	var response *httptest.ResponseRecorder
	select {
	case response = <-result:
	case <-time.After(10 * time.Second):
		t.Fatal("DAV deletion did not finish")
	}
	if response.Code != http.StatusConflict {
		t.Fatalf("expected DAV 409 got %d: %s", response.Code, response.Body.String())
	}
	var referrer models.Contact
	if err := db.First(&referrer, "id = ?", late.ID).Error; err != nil {
		t.Fatal(err)
	}
	if referrer.FirstMetThroughContactID == nil || *referrer.FirstMetThroughContactID != source.ID {
		t.Fatal("conflict changed new referrer")
	}
	var count int64
	if err := db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("conflict deleted source")
	}
	retry := httptest.NewRecorder()
	handler.ServeHTTP(retry, httptest.NewRequest(http.MethodDelete, path, nil).WithContext(ctx))
	if retry.Code != http.StatusNoContent {
		t.Fatalf("DAV retry failed: %d %s", retry.Code, retry.Body.String())
	}
	if err := db.First(&referrer, "id = ?", late.ID).Error; err != nil {
		t.Fatal(err)
	}
	if referrer.FirstMetThroughContactID != nil {
		t.Fatal("retry left introducer reference")
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("retry left source active")
	}
}
