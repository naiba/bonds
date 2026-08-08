package cron

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestRegisterJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	err := s.RegisterJob("@every 1h", "test-job", func() {})
	if err != nil {
		t.Fatalf("RegisterJob failed: %v", err)
	}
}

func TestRegisterJobInvalidSpec(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	err := s.RegisterJob("invalid-spec", "bad-job", func() {})
	if err == nil {
		t.Fatal("Expected error for invalid cron spec")
	}
}

func TestJobExecution(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	var executed atomic.Int32

	// Schedule job to run every second
	err := s.RegisterJob("@every 1s", "exec-test", func() {
		executed.Add(1)
	})
	if err != nil {
		t.Fatalf("RegisterJob failed: %v", err)
	}

	s.Start()
	// Wait enough time for at least one execution
	time.Sleep(2500 * time.Millisecond)
	ctx := s.Stop()
	<-ctx.Done()

	count := executed.Load()
	if count < 1 {
		t.Fatalf("Expected job to execute at least once, got %d", count)
	}

	// Verify cron record was created in DB
	var cronRecord models.Cron
	result := db.Where("command = ?", "exec-test").First(&cronRecord)
	if result.Error != nil {
		t.Fatalf("Cron record not found: %v", result.Error)
	}
	if cronRecord.LastRunAt == nil {
		t.Fatal("LastRunAt should be set after execution")
	}
}

func TestJobSkipsWhenRecentlyRun(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	// Pre-create a cron record with a recent LastRunAt
	now := time.Now()
	db.Create(&models.Cron{
		Command:   "recent-job",
		LastRunAt: &now,
	})

	var executed atomic.Int32
	s.runJob("recent-job", func() {
		executed.Add(1)
	})

	if executed.Load() != 0 {
		t.Fatal("Job should have been skipped because it ran recently")
	}
}

func TestJobPanicRecovery(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	// Should not panic
	s.runJob("panic-job", func() {
		panic("test panic")
	})
}

func TestUpsertJobReplacesSpec(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	if err := s.UpsertJob("@every 2s", "upsert-test", func() {}); err != nil {
		t.Fatalf("UpsertJob failed: %v", err)
	}
	// Reschedule with a new spec: the old entry must be replaced, not stacked.
	if err := s.UpsertJob("@every 1s", "upsert-test", func() {}); err != nil {
		t.Fatalf("UpsertJob failed: %v", err)
	}

	entries := s.cron.Entries()
	if len(entries) != 1 {
		t.Fatalf("Expected a single cron entry after reschedule, got %d", len(entries))
	}
	if s.jobs["upsert-test"] != entries[0].ID {
		t.Fatalf("jobs map does not point at the live entry")
	}
}

func TestUpsertJobRemovesOnEmptySpec(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	var executed atomic.Int32
	fn := func() {
		executed.Add(1)
	}

	if err := s.UpsertJob("@every 1s", "remove-test", fn); err != nil {
		t.Fatalf("UpsertJob failed: %v", err)
	}
	// Empty spec must remove the job entirely.
	if err := s.UpsertJob("", "remove-test", fn); err != nil {
		t.Fatalf("UpsertJob remove failed: %v", err)
	}

	s.Start()
	time.Sleep(1200 * time.Millisecond)
	ctx := s.Stop()
	<-ctx.Done()

	if count := executed.Load(); count != 0 {
		t.Fatalf("Expected removed job to never run, got %d", count)
	}
}

func TestStartAndStop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	s.Start()
	ctx := s.Stop()
	<-ctx.Done()
}
