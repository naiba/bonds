package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newLiveSQLiteBackupService opens a file database the way the server does
// (WAL journal, pooled connections) so a restore runs against a database that
// is genuinely in use.
func newLiveSQLiteBackupService(t *testing.T, dsnSuffix string) (*BackupService, *gorm.DB) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "bonds.db") + dsnSuffix
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		t.Fatalf("enable WAL: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: dsn},
		Storage:  config.StorageConfig{UploadDir: t.TempDir()},
		Backup:   config.BackupConfig{Dir: t.TempDir(), Retention: 30},
	}
	return NewBackupService(db, cfg), db
}

func assertSQLiteRestoreRoundTrip(t *testing.T, svc *BackupService, db *gorm.DB) {
	t.Helper()
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Backup", LastName: "Owner", Email: "backup-owner@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register owner: %v", err)
	}

	created, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Later", LastName: "User", Email: "registered-after-backup@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register later user: %v", err)
	}
	if err := db.Unscoped().Where("email = ?", "backup-owner@example.com").Delete(&models.User{}).Error; err != nil {
		t.Fatalf("delete owner: %v", err)
	}

	if err := svc.Restore(created.Filename); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	// Read through the same pool the application keeps using after a restore.
	var emails []string
	if err := db.Model(&models.User{}).Order("email").Pluck("email", &emails).Error; err != nil {
		t.Fatalf("read users after restore: %v", err)
	}
	if len(emails) != 1 || emails[0] != "backup-owner@example.com" {
		t.Fatalf("users after restore = %v, want only the backed-up owner", emails)
	}
	var integrity string
	if err := db.Raw("PRAGMA integrity_check").Scan(&integrity).Error; err != nil {
		t.Fatalf("integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check after restore = %q", integrity)
	}
	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "After", LastName: "Restore", Email: "registered-after-restore@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("the database is not writable after restore: %v", err)
	}
}

// Overwriting the database file underneath open connections ignores the WAL:
// committed changes made after the backup survive the "restore", or the file
// ends up corrupt.
func TestBackupRestore_restoresLiveSQLiteDatabaseInWALMode(t *testing.T) {
	svc, db := newLiveSQLiteBackupService(t, "")
	assertSQLiteRestoreRoundTrip(t, svc, db)
}

// DB_DSN may carry driver parameters. Treating the whole DSN as a file name
// wrote the restored database to a stray file and left the real one untouched.
func TestBackupRestore_restoresSQLiteDatabaseOpenedWithDSNParameters(t *testing.T) {
	svc, db := newLiveSQLiteBackupService(t, "?_busy_timeout=5000")
	assertSQLiteRestoreRoundTrip(t, svc, db)

	strayFile := svc.cfg.Database.DSN
	if _, err := os.Stat(strayFile); !os.IsNotExist(err) {
		t.Fatalf("restore created a stray file named after the full DSN (stat err = %v)", err)
	}
}

// A backup taken by an older release has that release's schema. Restoring it
// puts the old schema under the running, newer server: columns added since are
// missing and every query that touches them fails until someone restarts the
// server. The restore has to bring the schema up to date itself.
func TestBackupRestore_upgradesSchemaOfBackupFromOlderVersion(t *testing.T) {
	svc, db := newLiveSQLiteBackupService(t, "")
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	if _, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Backup", LastName: "Owner", Email: "backup-owner@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register owner: %v", err)
	}
	// Make the database look like an older release, back it up, then return
	// to the current schema.
	if err := db.Exec("ALTER TABLE users DROP COLUMN task_view").Error; err != nil {
		t.Fatalf("drop a column to simulate an older schema: %v", err)
	}
	olderVersionBackup, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate back to the current schema: %v", err)
	}
	if !db.Migrator().HasColumn(&models.User{}, "task_view") {
		t.Fatal("test setup: the current schema should have users.task_view")
	}

	if err := svc.Restore(olderVersionBackup.Filename); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	if !db.Migrator().HasColumn(&models.User{}, "task_view") {
		t.Fatal("users.task_view is missing after restoring a backup from an older version")
	}
	var owner models.User
	if err := db.Where("email = ?", "backup-owner@example.com").First(&owner).Error; err != nil {
		t.Fatalf("read the restored user with the current model: %v", err)
	}
	if err := db.Model(&owner).Update("task_view", "board").Error; err != nil {
		t.Fatalf("write a column added after the backup was taken: %v", err)
	}
}
