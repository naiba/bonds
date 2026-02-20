package services

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/testutil"
)

func setupBackupTest(t *testing.T) (*BackupService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	backupDir := t.TempDir()
	uploadDir := t.TempDir()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    ":memory:",
		},
		Storage: config.StorageConfig{
			UploadDir: uploadDir,
		},
		Backup: config.BackupConfig{
			Dir:       backupDir,
			Cron:      "",
			Retention: 30,
		},
	}

	svc := NewBackupService(db, cfg)
	return svc, backupDir
}

func TestBackupCreate(t *testing.T) {
	svc, backupDir := setupBackupTest(t)

	resp, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if resp == nil {
		t.Fatal("Create() returned nil")
	}
	if resp.Filename == "" {
		t.Error("expected non-empty filename")
	}
	if resp.Size <= 0 {
		t.Error("expected positive file size")
	}

	zipPath := filepath.Join(backupDir, resp.Filename)
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("backup file not found: %v", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	hasDB := false
	for _, f := range r.File {
		if f.Name == "database.db" {
			hasDB = true
		}
	}
	if !hasDB {
		t.Error("zip does not contain database.db")
	}
}

func TestBackupList(t *testing.T) {
	svc, _ := setupBackupTest(t)

	_, err := svc.Create()
	if err != nil {
		t.Fatalf("first Create() error: %v", err)
	}

	time.Sleep(time.Second)

	_, err = svc.Create()
	if err != nil {
		t.Fatalf("second Create() error: %v", err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(list))
	}
	if list[0].CreatedAt.Before(list[1].CreatedAt) {
		t.Error("expected newest first")
	}
}

func TestBackupDelete(t *testing.T) {
	svc, backupDir := setupBackupTest(t)

	resp, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	err = svc.Delete(resp.Filename)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	zipPath := filepath.Join(backupDir, resp.Filename)
	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		t.Error("backup file should be deleted")
	}
}

func TestBackupGetFilePath(t *testing.T) {
	svc, _ := setupBackupTest(t)

	resp, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	path, err := svc.GetFilePath(resp.Filename)
	if err != nil {
		t.Fatalf("GetFilePath() error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}

	_, err = svc.GetFilePath("../../../etc/passwd")
	if err != ErrBackupInvalidFilename {
		t.Errorf("expected ErrBackupInvalidFilename for path traversal, got: %v", err)
	}

	_, err = svc.GetFilePath("notazip.txt")
	if err != ErrBackupInvalidFilename {
		t.Errorf("expected ErrBackupInvalidFilename for non-zip, got: %v", err)
	}

	_, err = svc.GetFilePath("bonds-2026-01-01-000000.zip")
	if err != ErrBackupNotFound {
		t.Errorf("expected ErrBackupNotFound for missing file, got: %v", err)
	}
}

func TestBackupCleanOldBackups(t *testing.T) {
	svc, backupDir := setupBackupTest(t)

	oldFile := filepath.Join(backupDir, "bonds-2020-01-01-000000.zip")
	if err := os.WriteFile(oldFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("create old file: %v", err)
	}
	oldTime := time.Now().Add(-31 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	resp, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	err = svc.CleanOldBackups()
	if err != nil {
		t.Fatalf("CleanOldBackups() error: %v", err)
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old backup should be cleaned up")
	}

	newFile := filepath.Join(backupDir, resp.Filename)
	if _, err := os.Stat(newFile); err != nil {
		t.Error("new backup should still exist")
	}
}

func TestBackupGetConfig(t *testing.T) {
	svc, _ := setupBackupTest(t)

	cfg := svc.GetConfig()
	if cfg.DBDriver != "sqlite" {
		t.Errorf("expected sqlite, got %s", cfg.DBDriver)
	}
	if cfg.CronEnabled {
		t.Error("expected cron disabled")
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", cfg.RetentionDays)
	}
}
