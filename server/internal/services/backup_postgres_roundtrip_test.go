package services

import (
	"archive/zip"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests run pg_dump and psql against a real PostgreSQL server. They are
// the only coverage of what the generated SQL actually does; the tests in
// backup_postgres_test.go only pin the command-line arguments.

var postgresRoundTripDatabaseSeq atomic.Uint64

type postgresRoundTripDatabase struct {
	db        *gorm.DB
	dsn       string
	uploadDir string
	backup    *BackupService
}

// newPostgresRoundTripDatabase creates a migrated database of its own. pg_dump
// works on whole databases, so the per-test schema used elsewhere is not enough.
func newPostgresRoundTripDatabase(t *testing.T) *postgresRoundTripDatabase {
	t.Helper()
	return newPostgresRoundTripDatabaseInSchema(t, "")
}

// newPostgresRoundTripDatabaseInSchema puts the application's tables into the
// named schema through the database's search_path, the way a deployment that
// shares a database does. An empty name keeps them in public.
func newPostgresRoundTripDatabaseInSchema(t *testing.T, schema string) *postgresRoundTripDatabase {
	t.Helper()
	adminDSN := os.Getenv(testutil.PostgresDSNEnv)
	if adminDSN == "" {
		t.Skip("needs a PostgreSQL server: set " + testutil.PostgresDSNEnv)
	}
	parsed, err := url.Parse(adminDSN)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Skip("needs " + testutil.PostgresDSNEnv + " in URL form to derive a per-test database")
	}
	admin, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	skipUnlessPostgresClientToolsMatchServer(t, admin)

	name := fmt.Sprintf("bonds_backup_roundtrip_%d_%d", os.Getpid(), postgresRoundTripDatabaseSeq.Add(1))
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatalf("create database %s: %v", name, err)
	}
	if schema != "" {
		if err := admin.Exec("ALTER DATABASE " + name + " SET search_path = " + schema).Error; err != nil {
			t.Fatalf("set search_path of %s: %v", name, err)
		}
	}
	parsed.Path = "/" + name
	dsn := parsed.String()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect to %s: %v", name, err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
		if sqlDB, err := admin.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if schema != "" {
		if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
			t.Fatalf("create schema %s: %v", schema, err)
		}
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate %s: %v", name, err)
	}

	uploadDir := t.TempDir()
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "postgres", DSN: dsn},
		Storage:  config.StorageConfig{UploadDir: uploadDir},
		Backup:   config.BackupConfig{Dir: t.TempDir(), Retention: 30},
	}
	return &postgresRoundTripDatabase{db: db, dsn: dsn, uploadDir: uploadDir, backup: NewBackupService(db, cfg)}
}

// pg_dump refuses to dump a server newer than itself, which is what a CI
// runner's preinstalled client meets when the service container is newer.
func skipUnlessPostgresClientToolsMatchServer(t *testing.T, admin *gorm.DB) {
	t.Helper()
	for _, tool := range []string{"pg_dump", "psql"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("needs %s on PATH", tool)
		}
	}
	var serverVersionNum int
	if err := admin.Raw("SELECT current_setting('server_version_num')::int").Scan(&serverVersionNum).Error; err != nil {
		t.Fatalf("read server version: %v", err)
	}
	output, err := exec.Command("pg_dump", "--version").Output()
	if err != nil {
		t.Fatalf("pg_dump --version: %v", err)
	}
	match := regexp.MustCompile(`(\d+)(\.\d+)*`).FindStringSubmatch(string(output))
	if match == nil {
		t.Fatalf("cannot read pg_dump version from %q", output)
	}
	clientMajor, _ := strconv.Atoi(match[1])
	if serverMajor := serverVersionNum / 10000; clientMajor < serverMajor {
		t.Skipf("pg_dump %d cannot dump PostgreSQL %d", clientMajor, serverMajor)
	}
}

type postgresRoundTripSeed struct {
	ownerID   string
	vaultID   string
	contactID string
}

func (d *postgresRoundTripDatabase) seed(t *testing.T) postgresRoundTripSeed {
	t.Helper()
	registered, err := NewAuthService(d.db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{
		FirstName: "Backup", LastName: "Owner", Email: "backup-owner@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("register owner: %v", err)
	}
	vault, err := NewVaultService(d.db).CreateVault(registered.User.AccountID, registered.User.ID, dto.CreateVaultRequest{Name: "Backup Vault"}, "en")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	firstName := "Original"
	contact := models.Contact{VaultID: vault.ID, FirstName: &firstName}
	if err := d.db.Create(&contact).Error; err != nil {
		t.Fatalf("create contact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(d.uploadDir, "original-upload.txt"), []byte("original"), 0o644); err != nil {
		t.Fatalf("write upload: %v", err)
	}
	return postgresRoundTripSeed{ownerID: registered.User.ID, vaultID: vault.ID, contactID: contact.ID}
}

func (d *postgresRoundTripDatabase) countRows(t *testing.T, model interface{}, query string, args ...interface{}) int64 {
	t.Helper()
	var count int64
	if err := d.db.Unscoped().Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func writeBackupArchive(t *testing.T, backupDir, filename, databaseSQL string) {
	t.Helper()
	archiveFile, err := os.Create(filepath.Join(backupDir, filename))
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	zw := zip.NewWriter(archiveFile)
	entry, err := zw.Create("database.sql")
	if err != nil {
		t.Fatalf("create database.sql entry: %v", err)
	}
	if _, err := entry.Write([]byte(databaseSQL)); err != nil {
		t.Fatalf("write database.sql entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
}

// The restore target is the database the application is connected to, with
// every table present and populated. A backup has to bring it back to the
// backed-up state: deleted rows return, later rows disappear.
func TestBackupRestore_roundTripsLivePostgresDatabase(t *testing.T) {
	target := newPostgresRoundTripDatabase(t)
	seed := target.seed(t)

	created, err := target.backup.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := target.db.Unscoped().Delete(&models.Contact{}, "id = ?", seed.contactID).Error; err != nil {
		t.Fatalf("delete contact: %v", err)
	}
	if _, err := NewAuthService(target.db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{
		FirstName: "Later", LastName: "User", Email: "registered-after-backup@example.com", Password: "password123",
	}, "en"); err != nil {
		t.Fatalf("register later user: %v", err)
	}
	if err := os.Remove(filepath.Join(target.uploadDir, "original-upload.txt")); err != nil {
		t.Fatalf("remove upload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target.uploadDir, "uploaded-after-backup.txt"), []byte("later"), 0o644); err != nil {
		t.Fatalf("write later upload: %v", err)
	}

	if err := target.backup.Restore(created.Filename); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	if got := target.countRows(t, &models.Contact{}, "id = ? AND vault_id = ?", seed.contactID, seed.vaultID); got != 1 {
		t.Errorf("contact deleted after the backup: %d row(s) after restore, want 1", got)
	}
	if got := target.countRows(t, &models.User{}, "email = ?", "registered-after-backup@example.com"); got != 0 {
		t.Errorf("user registered after the backup: %d row(s) after restore, want 0", got)
	}
	if got := target.countRows(t, &models.User{}, "id = ?", seed.ownerID); got != 1 {
		t.Errorf("owner: %d row(s) after restore, want 1", got)
	}
	restoredUpload, err := os.ReadFile(filepath.Join(target.uploadDir, "original-upload.txt"))
	if err != nil || string(restoredUpload) != "original" {
		t.Errorf("original upload after restore = %q, err = %v", restoredUpload, err)
	}
	if _, err := os.Stat(filepath.Join(target.uploadDir, "uploaded-after-backup.txt")); !os.IsNotExist(err) {
		t.Errorf("upload written after the backup still exists (stat err = %v)", err)
	}

	// Sequences come back with the data: new rows must not collide with restored ids.
	journal, err := NewJournalService(target.db).Create(seed.vaultID, dto.CreateJournalRequest{Name: "Written after restore"})
	if err != nil {
		t.Fatalf("create journal after restore: %v", err)
	}
	if journal.ID == 0 {
		t.Fatal("journal created after restore has no id")
	}
}

// A restore that fails halfway must leave the database exactly as it was and
// must not go on to replace the uploads.
func TestBackupRestore_rollsBackFailedCleanDumpOnPostgres(t *testing.T) {
	target := newPostgresRoundTripDatabase(t)
	seed := target.seed(t)
	writeBackupArchive(t, target.backup.cfg.Backup.Dir, "bonds-2026-02-01-000000.zip",
		"DROP TABLE IF EXISTS public.contacts CASCADE;\nSELECT 1/0;\n")

	err := target.backup.Restore("bonds-2026-02-01-000000.zip")

	if err == nil {
		t.Fatal("Restore() reported success for a dump that fails")
	}
	if got := target.countRows(t, &models.Contact{}, "id = ?", seed.contactID); got != 1 {
		t.Errorf("contact rows after failed restore = %d, want 1 (the DROP must be rolled back)", got)
	}
	if _, statErr := os.Stat(filepath.Join(target.uploadDir, "original-upload.txt")); statErr != nil {
		t.Errorf("uploads were touched by a failed restore: %v", statErr)
	}
}

// Backups written before dumps carried DROP statements must still restore
// completely, both onto a freshly migrated instance and onto the live database
// they were taken from.
func TestBackupRestore_restoresLegacyDumpCompletelyOnPostgres(t *testing.T) {
	source := newPostgresRoundTripDatabase(t)
	seed := source.seed(t)
	legacyDumpPath := filepath.Join(t.TempDir(), "legacy.sql")
	if output, err := exec.Command("pg_dump", "--dbname="+source.dsn, "-f", legacyDumpPath).CombinedOutput(); err != nil {
		t.Fatalf("pg_dump: %s: %v", output, err)
	}
	legacyDump, err := os.ReadFile(legacyDumpPath)
	if err != nil {
		t.Fatalf("read legacy dump: %v", err)
	}
	if strings.Contains(string(legacyDump), "\nDROP ") {
		t.Fatal("test setup: the legacy dump unexpectedly contains DROP statements")
	}
	const archiveName = "bonds-2025-01-01-000000.zip"

	freshInstance := newPostgresRoundTripDatabase(t)
	liveInstance := source
	if err := liveInstance.db.Unscoped().Delete(&models.Contact{}, "id = ?", seed.contactID).Error; err != nil {
		t.Fatalf("delete contact on live instance: %v", err)
	}

	for name, target := range map[string]*postgresRoundTripDatabase{"fresh instance": freshInstance, "live instance": liveInstance} {
		t.Run(name, func(t *testing.T) {
			writeBackupArchive(t, target.backup.cfg.Backup.Dir, archiveName, string(legacyDump))

			if err := target.backup.Restore(archiveName); err != nil {
				t.Fatalf("Restore() of a legacy backup error: %v", err)
			}

			if got := target.countRows(t, &models.User{}, "id = ?", seed.ownerID); got != 1 {
				t.Errorf("owner rows = %d, want 1", got)
			}
			if got := target.countRows(t, &models.Vault{}, "id = ?", seed.vaultID); got != 1 {
				t.Errorf("vault rows = %d, want 1", got)
			}
			if got := target.countRows(t, &models.Contact{}, "id = ? AND vault_id = ?", seed.contactID, seed.vaultID); got != 1 {
				t.Errorf("contact rows = %d, want 1", got)
			}
			journal, err := NewJournalService(target.db).Create(seed.vaultID, dto.CreateJournalRequest{Name: "Written after restore"})
			if err != nil || journal.ID == 0 {
				t.Fatalf("create journal after restore: journal = %+v, err = %v", journal, err)
			}
		})
	}
}

// The restore drops and recreates tables inside one transaction, which needs
// exclusive locks. A session that keeps a transaction open on any of them made
// psql wait for ever and the restore request never returned.
func TestBackupRestore_givesUpWhenTablesStayLockedOnPostgres(t *testing.T) {
	target := newPostgresRoundTripDatabase(t)
	seed := target.seed(t)
	created, err := target.backup.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	previousTimeout := postgresRestoreLockTimeout
	t.Cleanup(func() { postgresRestoreLockTimeout = previousTimeout })
	postgresRestoreLockTimeout = "1s"

	sqlDB, err := target.db.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	lockHolder, err := sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin lock-holding transaction: %v", err)
	}
	releaseLock := sync.OnceFunc(func() { _ = lockHolder.Rollback() })
	t.Cleanup(releaseLock)
	if _, err := lockHolder.Exec("LOCK TABLE contacts IN ACCESS SHARE MODE"); err != nil {
		t.Fatalf("lock contacts: %v", err)
	}

	restored := make(chan error, 1)
	go func() { restored <- target.backup.Restore(created.Filename) }()

	select {
	case err := <-restored:
		if err == nil {
			t.Fatal("Restore() reported success although a table was locked the whole time")
		}
	case <-time.After(15 * time.Second):
		releaseLock()
		<-restored
		t.Fatal("Restore() was still waiting for the table lock after 15s")
	}
	releaseLock()
	if got := target.countRows(t, &models.Contact{}, "id = ?", seed.contactID); got != 1 {
		t.Errorf("contact rows after the abandoned restore = %d, want 1", got)
	}
}

// The locks are taken up front because every dump starts by switching the lock
// timeout off. Looking for the tables only in public found none when the
// database's search_path keeps them in another schema, and the restore waited
// on the locked table for ever again.
func TestBackupRestore_givesUpWhenTablesOutsidePublicStayLockedOnPostgres(t *testing.T) {
	target := newPostgresRoundTripDatabaseInSchema(t, "bonds_app")
	var tableSchema string
	if err := target.db.Raw("SELECT schemaname FROM pg_tables WHERE tablename = 'contacts'").Scan(&tableSchema).Error; err != nil || tableSchema != "bonds_app" {
		t.Fatalf("test setup: contacts lives in schema %q (err = %v), want bonds_app", tableSchema, err)
	}
	seed := target.seed(t)
	created, err := target.backup.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	previousTimeout := postgresRestoreLockTimeout
	t.Cleanup(func() { postgresRestoreLockTimeout = previousTimeout })
	postgresRestoreLockTimeout = "1s"

	sqlDB, err := target.db.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	lockHolder, err := sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin lock-holding transaction: %v", err)
	}
	releaseLock := sync.OnceFunc(func() { _ = lockHolder.Rollback() })
	t.Cleanup(releaseLock)
	if _, err := lockHolder.Exec("LOCK TABLE contacts IN ACCESS SHARE MODE"); err != nil {
		t.Fatalf("lock contacts: %v", err)
	}

	restored := make(chan error, 1)
	go func() { restored <- target.backup.Restore(created.Filename) }()

	select {
	case err := <-restored:
		if err == nil {
			t.Fatal("Restore() reported success although a table was locked the whole time")
		}
	case <-time.After(15 * time.Second):
		releaseLock()
		<-restored
		t.Fatal("Restore() was still waiting for the table lock after 15s")
	}
	releaseLock()
	if got := target.countRows(t, &models.Contact{}, "id = ?", seed.contactID); got != 1 {
		t.Errorf("contact rows after the abandoned restore = %d, want 1", got)
	}
}
