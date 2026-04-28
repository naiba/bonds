package cron

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestAcquireLockFirstRunCreatesRow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	got, err := s.acquireLock("first-run")
	if err != nil {
		t.Fatalf("acquireLock failed: %v", err)
	}
	if !got {
		t.Fatal("expected first acquireLock to succeed")
	}

	var rec models.Cron
	if err := db.Where("command = ?", "first-run").First(&rec).Error; err != nil {
		t.Fatalf("Cron row should exist: %v", err)
	}
	if rec.LastRunAt == nil {
		t.Fatal("LastRunAt must be set after acquireLock")
	}
}

func TestAcquireLockSkipsWithinWindow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	now := time.Now()
	if err := db.Create(&models.Cron{Command: "recent", LastRunAt: &now}).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	got, err := s.acquireLock("recent")
	if err != nil {
		t.Fatalf("acquireLock failed: %v", err)
	}
	if got {
		t.Fatal("expected acquireLock to be denied within window")
	}
}

func TestAcquireLockAllowsAfterWindow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	old := time.Now().Add(-2 * time.Minute)
	if err := db.Create(&models.Cron{Command: "stale", LastRunAt: &old}).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	got, err := s.acquireLock("stale")
	if err != nil {
		t.Fatalf("acquireLock failed: %v", err)
	}
	if !got {
		t.Fatal("expected acquireLock to succeed after window expired")
	}

	var rec models.Cron
	if err := db.Where("command = ?", "stale").First(&rec).Error; err != nil {
		t.Fatalf("row should exist: %v", err)
	}
	if rec.LastRunAt == nil || time.Since(*rec.LastRunAt) > 5*time.Second {
		t.Fatalf("LastRunAt should be updated to a recent time, got %v", rec.LastRunAt)
	}
}

// Concurrent callers on the same DB must yield exactly one winner per window.
// SQLite serialises writers, which is precisely what acquireLockGeneric relies
// on; this exercises that contract.
func TestAcquireLockConcurrentSingleWinner(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := NewScheduler(db)

	const goroutines = 16
	var wg sync.WaitGroup
	var winners int32
	var firstErr atomic.Value

	wg.Add(goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			got, err := s.acquireLock("contested")
			if err != nil {
				firstErr.CompareAndSwap(nil, err)
				return
			}
			if got {
				atomic.AddInt32(&winners, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if e, ok := firstErr.Load().(error); ok && e != nil {
		t.Fatalf("acquireLock errored: %v", e)
	}
	if got := atomic.LoadInt32(&winners); got != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", got)
	}
}

func TestRunJobIsAtomicAcrossReplicas(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s1 := NewScheduler(db)
	s2 := NewScheduler(db)

	var executions int32
	job := func() { atomic.AddInt32(&executions, 1) }

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); s1.runJob("shared", job) }()
	go func() { defer wg.Done(); s2.runJob("shared", job) }()
	wg.Wait()

	if got := atomic.LoadInt32(&executions); got != 1 {
		t.Fatalf("expected job to execute exactly once across replicas, got %d", got)
	}
}

func TestJobLockKeyStable(t *testing.T) {
	if jobLockKey("a") != jobLockKey("a") {
		t.Fatal("jobLockKey must be deterministic")
	}
	if jobLockKey("a") == jobLockKey("b") {
		t.Fatal("jobLockKey must differ for different names")
	}
}

func TestIsUniqueConstraintErrMatches(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"UNIQUE constraint failed: crons.command", true},
		{"pq: duplicate key value violates unique constraint \"crons_command_key\"", true},
		{"some other error", false},
		{"", false},
	}
	for _, c := range cases {
		err := errString(c.msg)
		if got := isUniqueConstraintErr(err); got != c.want {
			t.Errorf("isUniqueConstraintErr(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
	if isUniqueConstraintErr(nil) {
		t.Error("isUniqueConstraintErr(nil) should be false")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
