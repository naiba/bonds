package main

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"

	"gorm.io/gorm"
)

func TestMigrateContactTasksFreshDB(t *testing.T) {
	db := testutil.SetupTestDB(t)
	if err := migrateContactTasksToManyToMany(db); err != nil {
		t.Fatalf("fresh DB migration should be a no-op, got %v", err)
	}
	if db.Migrator().HasColumn(&models.ContactTask{}, "contact_id") {
		t.Errorf("contact_id should not exist on a fresh DB")
	}
}

func TestMigrateContactTasksLegacyDataMoves(t *testing.T) {
	db := testutil.SetupTestDB(t)
	seedLegacyContactTask(t, db, "task-vault", "task-contact", "Old task")

	if err := migrateContactTasksToManyToMany(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if db.Migrator().HasColumn(&models.ContactTask{}, "contact_id") {
		t.Errorf("legacy contact_id column should be gone")
	}

	var pivots []models.TaskContact
	if err := db.Find(&pivots).Error; err != nil {
		t.Fatalf("load pivots: %v", err)
	}
	if len(pivots) != 1 {
		t.Fatalf("expected 1 pivot row, got %d", len(pivots))
	}
	if pivots[0].ContactID != "task-contact" {
		t.Errorf("pivot contact_id = %q, want 'task-contact'", pivots[0].ContactID)
	}
}

func TestMigrateContactTasksIdempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	seedLegacyContactTask(t, db, "v1", "c1", "Task")

	if err := migrateContactTasksToManyToMany(db); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := migrateContactTasksToManyToMany(db); err != nil {
		t.Fatalf("second run should be a no-op, got %v", err)
	}

	var pivots []models.TaskContact
	db.Find(&pivots)
	if len(pivots) != 1 {
		t.Errorf("second run duplicated pivots, got %d rows", len(pivots))
	}
}

func TestMigrateContactTasksPreservesPivotFKReferentialIntegrity(t *testing.T) {
	db := testutil.SetupTestDB(t)
	seedLegacyContactTask(t, db, "v1", "c1", "Task A")
	seedLegacyContactTask(t, db, "v1", "c1", "Task B")

	if err := migrateContactTasksToManyToMany(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	var taskIDs []uint
	db.Model(&models.ContactTask{}).Pluck("id", &taskIDs)
	if len(taskIDs) != 2 {
		t.Fatalf("expected 2 tasks post-migration, got %d", len(taskIDs))
	}

	var pivots []models.TaskContact
	db.Where("contact_task_id IN ?", taskIDs).Find(&pivots)
	if len(pivots) != 2 {
		t.Errorf("expected 2 pivot rows, got %d", len(pivots))
	}
	for _, p := range pivots {
		var task models.ContactTask
		if err := db.First(&task, p.ContactTaskID).Error; err != nil {
			t.Errorf("pivot %v references missing task: %v", p, err)
		}
	}
}

func seedLegacyContactTask(t *testing.T, db *gorm.DB, vaultID, contactID, label string) {
	t.Helper()

	if !db.Migrator().HasColumn(&models.ContactTask{}, "contact_id") {
		if err := db.Exec(`ALTER TABLE contact_tasks ADD COLUMN contact_id TEXT`).Error; err != nil {
			t.Fatalf("add legacy contact_id column: %v", err)
		}
	}
	if err := db.Exec(`
		INSERT INTO contact_tasks (vault_id, contact_id, label, author_name, status, position, completed, created_at, updated_at)
		VALUES (?, ?, ?, 'seed', 'todo', 0, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, vaultID, contactID, label).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
}
