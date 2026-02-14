package cron

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestNewScheduler(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.cron == nil {
		t.Fatal("Scheduler.cron is nil")
	}
	if s.db == nil {
		t.Fatal("Scheduler.db is nil")
	}
}

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

func TestStartAndStop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	s.Start()
	ctx := s.Stop()
	<-ctx.Done()
}
