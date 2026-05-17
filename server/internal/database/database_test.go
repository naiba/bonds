package database

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAutoMigrateBackfillsLegacyContactTaskVaultID(t *testing.T) {
	db := openMigrationTestDB(t)
	createLegacyContactTaskSchema(t, db)

	seedContactTaskVaultAndContact(t, db, "account-1", "vault-1", "contact-1")
	if err := db.Exec(`
		INSERT INTO contact_tasks (id, contact_id, author_name, label, completed)
		VALUES (?, ?, ?, ?, ?)
	`, 42, "contact-1", "User", "Legacy task", false).Error; err != nil {
		t.Fatalf("insert legacy task: %v", err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	var vaultID string
	if err := db.Raw(`SELECT vault_id FROM contact_tasks WHERE id = ?`, 42).Scan(&vaultID).Error; err != nil {
		t.Fatalf("read migrated vault_id: %v", err)
	}
	if vaultID != "vault-1" {
		t.Fatalf("vault_id = %q, want %q", vaultID, "vault-1")
	}

	var assignmentCount int64
	if err := db.Table("task_contacts").
		Where("contact_task_id = ? AND contact_id = ?", 42, "contact-1").
		Count(&assignmentCount).Error; err != nil {
		t.Fatalf("count migrated task_contacts row: %v", err)
	}
	if assignmentCount != 1 {
		t.Fatalf("task_contacts rows = %d, want 1", assignmentCount)
	}

	if !columnIsNotNull(t, db, "contact_tasks", "vault_id") {
		t.Fatal("contact_tasks.vault_id should be NOT NULL after migration")
	}
	if hasColumn(db, "contact_tasks", "contact_id") {
		t.Fatal("legacy contact_tasks.contact_id column should be removed after migration")
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second AutoMigrate failed: %v", err)
	}
}

func TestAutoMigratePreservesPartiallyMigratedContactTaskColumns(t *testing.T) {
	db := openMigrationTestDB(t)
	createPartiallyMigratedContactTaskSchema(t, db)
	seedContactTaskVaultAndContact(t, db, "account-1", "vault-1", "contact-1")

	if err := db.Exec(`
		INSERT INTO contact_tasks (id, vault_id, contact_id, parent_task_id, author_name, label, status, position, completed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 42, "vault-1", "contact-1", nil, "User", "Parent task", models.TaskStatusInProgress, 10, false).Error; err != nil {
		t.Fatalf("insert partially migrated parent task: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO contact_tasks (id, vault_id, contact_id, parent_task_id, author_name, label, status, position, completed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 43, "vault-1", "contact-1", 42, "User", "Child task", models.TaskStatusBlocked, 20, true).Error; err != nil {
		t.Fatalf("insert partially migrated child task: %v", err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	var child struct {
		VaultID      string        `gorm:"column:vault_id"`
		ParentTaskID sql.NullInt64 `gorm:"column:parent_task_id"`
		Status       string        `gorm:"column:status"`
		Position     int           `gorm:"column:position"`
		Completed    bool          `gorm:"column:completed"`
	}
	if err := db.Raw(`SELECT vault_id, parent_task_id, status, position, completed FROM contact_tasks WHERE id = ?`, 43).Scan(&child).Error; err != nil {
		t.Fatalf("read migrated child task: %v", err)
	}
	if child.VaultID != "vault-1" {
		t.Fatalf("vault_id = %q, want vault-1", child.VaultID)
	}
	if !child.ParentTaskID.Valid || child.ParentTaskID.Int64 != 42 {
		t.Fatalf("parent_task_id = %+v, want 42", child.ParentTaskID)
	}
	if child.Status != models.TaskStatusBlocked {
		t.Fatalf("status = %q, want %q", child.Status, models.TaskStatusBlocked)
	}
	if child.Position != 20 {
		t.Fatalf("position = %d, want 20", child.Position)
	}
	if !child.Completed {
		t.Fatal("completed = false, want true")
	}

	var assignmentCount int64
	if err := db.Table("task_contacts").Where("contact_task_id IN ? AND contact_id = ?", []uint{42, 43}, "contact-1").Count(&assignmentCount).Error; err != nil {
		t.Fatalf("count migrated task contacts: %v", err)
	}
	if assignmentCount != 2 {
		t.Fatalf("task_contacts rows = %d, want 2", assignmentCount)
	}
}

func TestAutoMigrateMigratesMultipleLegacyContactTasks(t *testing.T) {
	db := openMigrationTestDB(t)
	createLegacyContactTaskSchema(t, db)
	seedContactTaskVaultAndContact(t, db, "account-1", "vault-1", "contact-1")

	for id, label := range map[int]string{42: "Task A", 43: "Task B"} {
		if err := db.Exec(`
			INSERT INTO contact_tasks (id, contact_id, author_name, label, completed)
			VALUES (?, ?, ?, ?, ?)
		`, id, "contact-1", "User", label, false).Error; err != nil {
			t.Fatalf("insert legacy task %d: %v", id, err)
		}
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	var taskCount int64
	if err := db.Model(&models.ContactTask{}).Count(&taskCount).Error; err != nil {
		t.Fatalf("count migrated tasks: %v", err)
	}
	if taskCount != 2 {
		t.Fatalf("contact_tasks rows = %d, want 2", taskCount)
	}
	var assignmentCount int64
	if err := db.Model(&models.TaskContact{}).Count(&assignmentCount).Error; err != nil {
		t.Fatalf("count migrated task contacts: %v", err)
	}
	if assignmentCount != 2 {
		t.Fatalf("task_contacts rows = %d, want 2", assignmentCount)
	}
}

func TestAutoMigrateFailsWhenLegacyContactTaskVaultCannotBeBackfilled(t *testing.T) {
	db := openMigrationTestDB(t)
	createLegacyContactTaskSchema(t, db)
	if err := db.Exec(`
		INSERT INTO contact_tasks (id, contact_id, author_name, label, completed)
		VALUES (?, ?, ?, ?, ?)
	`, 7, "missing-contact", "User", "Orphan task", false).Error; err != nil {
		t.Fatalf("insert legacy orphan task: %v", err)
	}

	err := AutoMigrate(db)
	if err == nil {
		t.Fatal("AutoMigrate succeeded, want legacy contact task backfill error")
	}
	if !strings.Contains(err.Error(), "legacy contact_tasks") {
		t.Fatalf("AutoMigrate error = %v, want legacy contact_tasks context", err)
	}
}

func openMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	return db
}

func createLegacyContactTaskSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	createContactTaskMigrationDependencies(t, db)
	statements := []string{
		`CREATE TABLE contact_tasks (
			id integer PRIMARY KEY AUTOINCREMENT,
			contact_id text NOT NULL,
			author_id text,
			uuid text,
			vcalendar text,
			distant_uuid text,
			distant_etag text,
			distant_uri text,
			author_name text NOT NULL,
			label text NOT NULL,
			description text,
			completed numeric DEFAULT false,
			completed_at datetime,
			due_at datetime,
			deleted_at datetime,
			created_at datetime,
			updated_at datetime
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create legacy schema: %v", err)
		}
	}
}

func createPartiallyMigratedContactTaskSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	createContactTaskMigrationDependencies(t, db)
	if err := db.Exec(`CREATE TABLE contact_tasks (
		id integer PRIMARY KEY AUTOINCREMENT,
		vault_id text,
		contact_id text NOT NULL,
		parent_task_id integer,
		author_id text,
		uuid text,
		vcalendar text,
		distant_uuid text,
		distant_etag text,
		distant_uri text,
		author_name text NOT NULL,
		label text NOT NULL,
		description text,
		status text DEFAULT 'todo',
		position integer DEFAULT 0,
		completed numeric DEFAULT false,
		completed_at datetime,
		due_at datetime,
		deleted_at datetime,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create partially migrated task schema: %v", err)
	}
}

func createContactTaskMigrationDependencies(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Migrator().CreateTable(&models.Account{}, &models.Vault{}, &models.Contact{}); err != nil {
		t.Fatalf("create current account/vault/contact schema: %v", err)
	}
}

func seedContactTaskVaultAndContact(t *testing.T, db *gorm.DB, accountID, vaultID, contactID string) {
	t.Helper()
	if err := db.Create(&models.Account{ID: accountID}).Error; err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if err := db.Create(&models.Vault{ID: vaultID, AccountID: accountID, Type: "private", Name: "Legacy Vault"}).Error; err != nil {
		t.Fatalf("insert vault: %v", err)
	}
	if err := db.Create(&models.Contact{ID: contactID, VaultID: vaultID}).Error; err != nil {
		t.Fatalf("insert contact: %v", err)
	}
}

func columnIsNotNull(t *testing.T, db *gorm.DB, tableName, columnName string) bool {
	t.Helper()
	var rows []struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}
	if err := db.Raw(`PRAGMA table_info(` + tableName + `)`).Scan(&rows).Error; err != nil {
		t.Fatalf("read table info: %v", err)
	}
	for _, row := range rows {
		if row.Name == columnName {
			return row.NotNull == 1
		}
	}
	t.Fatalf("column %s.%s not found", tableName, columnName)
	return false
}
