package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestCompanyDeleteSerializesWithMergeForeignKeys(t *testing.T) {
	testCompanyWriterDuringMerge(t, "delete")
}

func TestJobCreateSerializesWithMergeForeignKeys(t *testing.T) {
	testCompanyWriterDuringMerge(t, "associate")
}

func testCompanyWriterDuringMerge(t *testing.T, operation string) {
	t.Helper()
	svc, vaultID, userID, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL row-lock interleaving; FK lifecycle is tested on both databases")
	}
	installCompanyForeignKeys(t, svc.db)
	var constraints int64
	if err := svc.db.Raw(`SELECT count(*) FROM pg_constraint WHERE conrelid = 'contacts'::regclass AND confrelid = 'companies'::regclass AND contype = 'f'`).Scan(&constraints).Error; err != nil || constraints != 1 {
		t.Fatalf("real company FK missing: %d %v", constraints, err)
	}
	company, err := NewCompanyService(svc.db).Create(vaultID, dto.CreateCompanyRequest{Name: "Synthetic company"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Target"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Source"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactJobService(svc.db).LegacyUpdate(source.ID, vaultID, userID, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	svc.db = svc.db.WithContext(ctx)
	entered, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	mergeDone, writerDone := make(chan error, 1), make(chan error, 1)
	var mergeFinished, writerFinished, writerStarted bool
	const callback = "pause_merge_target_company_save"
	if err := svc.db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.Contact); ok && row.ID == target.ID && row.CompanyID != nil && *row.CompanyID == company.ID {
			close(entered)
			<-resume
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		release()
		if !mergeFinished {
			<-mergeDone
		}
		if writerStarted && !writerFinished {
			<-writerDone
		}
		svc.db.Callback().Update().Remove(callback)
	}()
	go func() { _, err := svc.MergeContacts(vaultID, userID, req); mergeDone <- err }()
	select {
	case <-entered:
	case err := <-mergeDone:
		mergeFinished = true
		t.Fatalf("merge failed before boundary: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("merge boundary not reached")
	}
	writerStarted = true
	go func() {
		if operation == "delete" {
			writerDone <- NewCompanyService(svc.db).Delete(company.ID, vaultID)
		} else {
			_, err := NewContactJobService(svc.db).Create(target.ID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "New role"})
			writerDone <- err
		}
	}()
	early, err := waitForCompanyLockOrCompletion(t, svc.db, writerDone)
	writerFinished = early
	if early {
		t.Fatalf("company writer did not wait for merge: %v", err)
	}
	release()
	mergeErr := <-mergeDone
	mergeFinished = true
	writerErr := <-writerDone
	writerFinished = true
	var remainingJobs int64
	if err := svc.db.Model(&models.ContactCompany{}).Where("company_id = ?", company.ID).Count(&remainingJobs).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("merge=%v writer=%v remainingJobs=%d", mergeErr, writerErr, remainingJobs)
	if mergeErr != nil || writerErr != nil {
		t.Fatalf("valid independent operations must serialize without deadlock: merge=%v writer=%v", mergeErr, writerErr)
	}
	if operation == "associate" {
		if remainingJobs != 2 {
			t.Fatalf("merge/new association lost jobs: %d", remainingJobs)
		}
		var survivor models.Contact
		if err := svc.db.First(&survivor, "id = ?", target.ID).Error; err != nil {
			t.Fatal(err)
		}
		if survivor.CompanyID == nil || *survivor.CompanyID != company.ID || ptrToStr(survivor.FirstName) != "Target" {
			t.Fatal("merge/new association lost survivor fields")
		}
		var tombstone models.Contact
		if err := svc.db.Unscoped().First(&tombstone, "id = ?", source.ID).Error; err != nil {
			t.Fatal(err)
		}
		if !tombstone.DeletedAt.Valid {
			t.Fatal("merge source was not deleted")
		}
		card, err := NewVCardService(svc.db).ExportContactToVCard(target.ID, vaultID)
		if err != nil {
			t.Fatal(err)
		}
		assertVCardJobs(t, card, [2]string{"Synthetic company", "Engineer"}, [2]string{"Synthetic company", "New role"})
		return
	}
	if remainingJobs != 0 {
		t.Fatalf("deleted company retained %d jobs", remainingJobs)
	}
	var survivor, tombstone models.Contact
	if err := svc.db.First(&survivor, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Unscoped().First(&tombstone, "id = ?", source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if survivor.CompanyID != nil || tombstone.CompanyID != nil || !tombstone.DeletedAt.Valid || ptrToStr(survivor.FirstName) != "Target" || survivor.JobPosition != nil {
		t.Fatal("company cleanup lost merged profile or retained references")
	}
	var count int64
	if err := svc.db.Model(&models.Company{}).Where("id = ?", company.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("company not deleted: %d %v", count, err)
	}
}

func installCompanyForeignKeys(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, relation := range []string{"Contacts", "ContactCompanies"} {
		if err := db.Migrator().CreateConstraint(&models.Company{}, relation); err != nil {
			t.Fatal(err)
		}
		if !db.Migrator().HasConstraint(&models.Company{}, relation) {
			t.Fatalf("company FK missing: %s", relation)
		}
	}
	if db.Dialector.Name() == "sqlite" {
		if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			t.Fatal(err)
		}
		var enabled int
		if err := db.Raw("PRAGMA foreign_keys").Scan(&enabled).Error; err != nil || enabled != 1 {
			t.Fatalf("foreign keys disabled: %d %v", enabled, err)
		}
	}
}

func TestCompanyDeleteCleansMergedForeignKeyReferences(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	installCompanyForeignKeys(t, svc.db)
	company, err := NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "Merged company"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Merged source"})
	if err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	if _, err := jobs.LegacyUpdate(source.ID, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	// Prove actual constraint enforcement, not just a migration flag.
	invalid := models.ContactCompany{ContactID: target.ID, CompanyID: company.ID + 10000}
	if err := svc.db.Create(&invalid).Error; err == nil {
		t.Fatal("missing company FK accepted an orphan")
	}
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	if err := NewCompanyService(svc.db).Delete(company.ID, vault); err != nil {
		t.Fatal(err)
	}
	var contacts []models.Contact
	if err := svc.db.Unscoped().Where("id IN ?", []string{target.ID, source.ID}).Find(&contacts).Error; err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 2 {
		t.Fatal("company deletion lost contacts")
	}
	for _, contact := range contacts {
		if contact.CompanyID != nil || contact.PreferredJobID != nil || (contact.ID == target.ID && contact.JobPosition != nil) || (contact.ID == source.ID && ptrToStr(contact.JobPosition) != "Engineer") {
			t.Fatal("company cleanup retained selection or changed tombstone position")
		}
		if contact.ID == source.ID && !contact.DeletedAt.Valid {
			t.Fatal("merged contact revived")
		}
		if contact.ID == target.ID && (contact.DeletedAt.Valid || ptrToStr(contact.FirstName) != "Survivor") {
			t.Fatal("survivor changed")
		}
	}
	var count int64
	if err := svc.db.Model(&models.ContactCompany{}).Where("company_id = ?", company.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("company jobs retained: %d %v", count, err)
	}
}

func TestCompanyDeletionDoesNotPushTombstones(t *testing.T) {
	f := setupJobSubscription(t)
	installCompanyForeignKeys(t, f.push.db)
	active := f.contact(t, "Live employee")
	historical := f.contact(t, "Historical employee")
	company := f.company(t, "Company with history")
	for _, id := range []string{active, historical} {
		if _, err := f.jobs.LegacyUpdate(id, f.vaultID, f.userID, dto.UpdateJobInfoRequest{CompanyID: &company, JobPosition: "Preserved title"}); err != nil {
			t.Fatal(err)
		}
		f.expect(t, map[string][2]string{id: {"Company with history", "Preserved title"}})
	}
	// A stored tombstone can retain legacy CompanyID, as the real merge test
	// above demonstrates. Keep its subscription state to detect accidental push.
	if err := f.push.db.Where("id = ?", historical).Delete(&models.Contact{}).Error; err != nil {
		t.Fatal(err)
	}
	var before models.ContactSubscriptionState
	if err := f.push.db.Where("contact_id = ?", historical).First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.companies.Delete(company, f.vaultID); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{active: {"", ""}})
	f.expectNoPush(t)
	var after models.ContactSubscriptionState
	if err := f.push.db.Where("contact_id = ?", historical).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.DistantEtag != before.DistantEtag || after.DistantURI != before.DistantURI {
		t.Fatal("tombstone subscriber changed")
	}
	var tombstone models.Contact
	if err := f.push.db.Unscoped().First(&tombstone, "id = ?", historical).Error; err != nil {
		t.Fatal(err)
	}
	if !tombstone.DeletedAt.Valid || tombstone.CompanyID != nil || ptrToStr(tombstone.JobPosition) != "Preserved title" || ptrToStr(tombstone.FirstName) != "Historical employee" {
		t.Fatal("tombstone profile changed beyond detached company")
	}
}

func TestCompanyDeletionSerializesWithSuccessiveMerges(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL row-lock interleaving")
	}
	installCompanyForeignKeys(t, svc.db)
	company, err := NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "Transferred company"})
	if err != nil {
		t.Fatal(err)
	}
	create := func(name string) string {
		c, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: name})
		if err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	source, target, successor := create("Source"), create("Target"), create("Successor")
	if _, err := NewContactJobService(svc.db).LegacyUpdate(source, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	firstRequest := reviewedContactMerge(t, svc, vault, user, target, source)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	svc.db = svc.db.WithContext(ctx)
	mergeEntered, mergeResume := make(chan struct{}), make(chan struct{})
	cleanupEntered, cleanupResume := make(chan struct{}), make(chan struct{})
	var mergeOnce, cleanupOnce, mergeBoundary, cleanupBoundary sync.Once
	releaseMerge := func() { mergeOnce.Do(func() { close(mergeResume) }) }
	releaseCleanup := func() { cleanupOnce.Do(func() { close(cleanupResume) }) }
	const callback = "pause_company_cleanup_after_transferred_job_delete"
	if err := svc.db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "contacts" {
			return
		}
		if row, ok := tx.Statement.Dest.(*models.Contact); ok && row.ID == target && row.CompanyID != nil {
			mergeBoundary.Do(func() { close(mergeEntered); <-mergeResume })
		}
		if fields, ok := tx.Statement.Dest.(map[string]interface{}); ok {
			if value, exists := fields["company_id"]; exists && value == nil {
				// Live mirror cleanup and tombstone cleanup may both update company_id.
				// Pause at the first boundary without assuming one SQL statement.
				cleanupBoundary.Do(func() { close(cleanupEntered); <-cleanupResume })
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	firstDone, deleteDone, secondDone := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	var firstFinished, deleteStarted, deleteFinished, secondStarted, secondFinished bool
	defer func() {
		releaseMerge()
		releaseCleanup()
		if !firstFinished {
			<-firstDone
		}
		if deleteStarted && !deleteFinished {
			<-deleteDone
		}
		if secondStarted && !secondFinished {
			<-secondDone
		}
		svc.db.Callback().Update().Remove(callback)
	}()
	go func() { _, err := svc.MergeContacts(vault, user, firstRequest); firstDone <- err }()
	select {
	case <-mergeEntered:
	case err := <-firstDone:
		firstFinished = true
		t.Fatalf("first merge did not reach boundary: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	deleteStarted = true
	go func() { deleteDone <- NewCompanyService(svc.db).Delete(company.ID, vault) }()
	early, err := waitForCompanyLockOrCompletion(t, svc.db, deleteDone)
	deleteFinished = early
	if early {
		t.Fatalf("delete did not wait for first merge: %v", err)
	}
	releaseMerge()
	err = <-firstDone
	firstFinished = true
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-cleanupEntered:
	case err := <-deleteDone:
		deleteFinished = true
		t.Fatalf("delete did not reach cleanup: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	secondRequest := reviewedContactMerge(t, svc, vault, user, successor, target)
	secondStarted = true
	go func() { _, err := svc.MergeContacts(vault, user, secondRequest); secondDone <- err }()
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	waiting := false
	for !waiting {
		select {
		case <-tick.C:
			var count int64
			err := svc.db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation IN ('companies'::regclass,'contacts'::regclass,'contact_companies'::regclass))`).Scan(&count).Error
			if err != nil {
				t.Fatal(err)
			}
			waiting = count > 0
		case err := <-secondDone:
			secondFinished = true
			t.Fatalf("second merge completed before company boundary: %v", err)
		case <-deadline.C:
			t.Fatal("second merge did not reach competing lock")
		}
	}
	releaseCleanup()
	deleteErr := <-deleteDone
	deleteFinished = true
	secondErr := <-secondDone
	secondFinished = true
	if deleteErr != nil {
		t.Fatalf("company deletion deadlocked after a job transfer: %v; merge=%v", deleteErr, secondErr)
	}
	if !errors.Is(secondErr, ErrContactMergeReviewChanged) {
		t.Fatalf("deleted company must invalidate the second review without partial merge: %v", secondErr)
	}
	var count int64
	if err := svc.db.Model(&models.Contact{}).Where("id IN ?", []string{target, successor}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("stale merge removed a live contact: %d %v", count, err)
	}
	if err := svc.db.Model(&models.ContactCompany{}).Where("company_id = ?", company.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("deleted company retained transferred jobs: %d %v", count, err)
	}
	var survivor models.Contact
	if err := svc.db.First(&survivor, "id = ?", target).Error; err != nil {
		t.Fatal(err)
	}
	if survivor.CompanyID != nil || survivor.JobPosition != nil || ptrToStr(survivor.FirstName) != "Target" {
		t.Fatal("company cleanup or stale merge lost survivor fields")
	}
}

// A company can appear after discovery but before the merge owns its contacts.
// Reject the old review instead of taking a company lock after contact locks.
func TestContactMergeRejectsNewCompanyAfterDiscovery(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL concurrent committed write between discovery and owner locks")
	}
	installCompanyForeignKeys(t, svc.db)
	company, err := NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "New employer"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Target"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Source"})
	if err != nil {
		t.Fatal(err)
	}
	request := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	svc.db = svc.db.WithContext(ctx)
	entered, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	done := make(chan error, 1)
	finished := false
	const callback = "pause_merge_after_company_discovery"
	if err := svc.db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "companies" {
			if _, ok := tx.Statement.Dest.(*[]uint); ok && tx.Statement.Clauses["FOR"].Expression != nil {
				close(entered)
				<-resume
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		release()
		if !finished {
			<-done
		}
		svc.db.Callback().Query().Remove(callback)
	}()
	go func() { _, err := svc.MergeContacts(vault, user, request); done <- err }()
	select {
	case <-entered:
	case err := <-done:
		finished = true
		t.Fatalf("merge did not reach discovery: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	job, err := NewContactJobService(svc.db).Create(source.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "New role"})
	if err != nil {
		t.Fatal(err)
	}
	release()
	err = <-done
	finished = true
	if !errors.Is(err, ErrContactMergeReviewChanged) {
		t.Fatalf("concurrent new company must invalidate review: %v", err)
	}
	var contacts []models.Contact
	if err := svc.db.Where("id IN ?", []string{source.ID, target.ID}).Find(&contacts).Error; err != nil || len(contacts) != 2 {
		t.Fatalf("rejected merge removed contacts: %d %v", len(contacts), err)
	}
	for _, contact := range contacts {
		expected := "Source"
		if contact.ID == target.ID {
			expected = "Target"
		}
		if ptrToStr(contact.FirstName) != expected || contact.CompanyID != nil || contact.JobPosition != nil {
			t.Fatal("rejected merge changed profile")
		}
	}
	var stored models.ContactCompany
	if err := svc.db.First(&stored, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ContactID != source.ID || stored.CompanyID != company.ID || ptrToStr(stored.JobPosition) != "New role" {
		t.Fatal("rejected merge changed committed job")
	}
	card, err := NewVCardService(svc.db).ExportContactToVCard(source.ID, vault)
	if err != nil {
		t.Fatal(err)
	}
	assertVCardJob(t, card, "New employer", "New role")
}
