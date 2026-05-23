package models

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupLifeEventTaskPostDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&LifeEvent{}, &ContactTask{}, &Post{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestBackfillLifeEventTaskPostCalendarTypes covers vault databases that
// pre-date the CalendarType column: legacy LifeEvent / ContactTask / Post rows
// live with an empty calendar_type after AutoMigrate adds the column. Without
// the backfill, downstream queries that filter on calendar_type=lunar would
// quietly skip the empty rows when joining with reminders / list views,
// because empty != gregorian in raw SQL — masking the difference between
// "intentionally lunar" and "never set".
func TestBackfillLifeEventTaskPostCalendarTypes(t *testing.T) {
	db := setupLifeEventTaskPostDB(t)

	now := time.Now()
	legacyLifeEvent := LifeEvent{TimelineEventID: 1, LifeEventTypeID: 1, HappenedAt: now}
	legacyTask := ContactTask{VaultID: "v1", AuthorName: "a", Label: "l", DueAt: &now}
	legacyPost := Post{JournalID: 1, WrittenAt: now}
	if err := db.Create(&legacyLifeEvent).Error; err != nil {
		t.Fatalf("create life event: %v", err)
	}
	if err := db.Create(&legacyTask).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := db.Create(&legacyPost).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	// Simulate the pre-default state by hand-clearing the column GORM filled in
	// when inserting (default:'gregorian' fires on Create). The on-disk state
	// we care about is "row inserted before the column existed", which
	// AutoMigrate would leave as empty string on add-column.
	for _, target := range []struct {
		table string
		id    any
	}{
		{"life_events", legacyLifeEvent.ID},
		{"contact_tasks", legacyTask.ID},
		{"posts", legacyPost.ID},
	} {
		if err := db.Exec("UPDATE "+target.table+" SET calendar_type = '' WHERE id = ?", target.id).Error; err != nil {
			t.Fatalf("clear %s: %v", target.table, err)
		}
	}

	if err := BackfillLifeEventTaskPostCalendarTypes(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var reloadedEvent LifeEvent
	var reloadedTask ContactTask
	var reloadedPost Post
	if err := db.First(&reloadedEvent, legacyLifeEvent.ID).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if err := db.First(&reloadedTask, legacyTask.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if err := db.First(&reloadedPost, legacyPost.ID).Error; err != nil {
		t.Fatalf("reload post: %v", err)
	}
	if reloadedEvent.CalendarType != "gregorian" {
		t.Errorf("life event calendar_type = %q, want gregorian", reloadedEvent.CalendarType)
	}
	if reloadedTask.CalendarType != "gregorian" {
		t.Errorf("task calendar_type = %q, want gregorian", reloadedTask.CalendarType)
	}
	if reloadedPost.CalendarType != "gregorian" {
		t.Errorf("post calendar_type = %q, want gregorian", reloadedPost.CalendarType)
	}

	// Second invocation must be a no-op — runs on every server boot.
	if err := BackfillLifeEventTaskPostCalendarTypes(db); err != nil {
		t.Fatalf("backfill (second run): %v", err)
	}
}

// TestBackfillPreservesNonGregorianRows guards against the backfill clobbering
// rows that were intentionally written with calendar_type='lunar'. The WHERE
// clause filters on empty/NULL — if that ever regresses to a blanket UPDATE,
// users would silently lose their lunar anchors on the next boot.
func TestBackfillPreservesNonGregorianRows(t *testing.T) {
	db := setupLifeEventTaskPostDB(t)

	now := time.Now()
	d, m, y := 15, 8, 2026
	lunarEvent := LifeEvent{
		TimelineEventID: 1, LifeEventTypeID: 1, HappenedAt: now,
		CalendarType: "lunar", OriginalDay: &d, OriginalMonth: &m, OriginalYear: &y,
	}
	if err := db.Create(&lunarEvent).Error; err != nil {
		t.Fatalf("create lunar event: %v", err)
	}

	if err := BackfillLifeEventTaskPostCalendarTypes(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var reloaded LifeEvent
	if err := db.First(&reloaded, lunarEvent.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.CalendarType != "lunar" {
		t.Errorf("calendar_type mutated: got %q want lunar", reloaded.CalendarType)
	}
	if reloaded.OriginalDay == nil || *reloaded.OriginalDay != d {
		t.Errorf("original_day mutated: got %v want %d", reloaded.OriginalDay, d)
	}
}
