package services

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupCreate_writesDatabaseSQL_whenDatabaseDriverIsPostgres(t *testing.T) {
	svc, backupDir := setupBackupTest(t)
	svc.cfg.Database.Driver = "postgres"
	svc.cfg.Database.DSN = "postgres://bonds:test@db.example.com:5432/bonds"

	binDir := t.TempDir()
	prependExecutableToPath(t, binDir, "pg_dump", "#!/bin/sh\nset -eu\noutput_file=\"\"\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    -f)\n      output_file=\"$2\"\n      shift 2\n      ;;\n    --dbname=*)\n      shift 1\n      ;;\n    *)\n      shift 1\n      ;;\n  esac\ndone\nprintf 'CREATE TABLE contacts (id int);\n' > \"$output_file\"\n")

	resp, err := svc.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	zipPath := filepath.Join(backupDir, resp.Filename)
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	var databaseSQL string
	for _, f := range r.File {
		if f.Name != "database.sql" {
			continue
		}

		rc, openErr := f.Open()
		if openErr != nil {
			t.Fatalf("failed to open database.sql entry: %v", openErr)
		}
		contents, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			t.Fatalf("failed to read database.sql entry: %v", readErr)
		}
		if closeErr != nil {
			t.Fatalf("failed to close database.sql entry: %v", closeErr)
		}
		databaseSQL = string(contents)
	}

	if databaseSQL == "" {
		t.Fatal("zip does not contain database.sql")
	}
	if !strings.Contains(databaseSQL, "CREATE TABLE contacts") {
		t.Fatalf("expected database.sql dump content, got %q", databaseSQL)
	}
}

func TestBackupRestore_invokesPSQLWithDumpPath_whenDatabaseDriverIsPostgres(t *testing.T) {
	svc, backupDir := setupBackupTest(t)
	svc.cfg.Database.Driver = "postgres"
	svc.cfg.Database.DSN = "postgres://bonds:test@db.example.com:5432/bonds"

	backupArchivePath := filepath.Join(backupDir, "bonds-2026-01-01-000000.zip")
	archiveFile, err := os.Create(backupArchivePath)
	if err != nil {
		t.Fatalf("create backup archive: %v", err)
	}

	zw := zip.NewWriter(archiveFile)
	entry, err := zw.Create("database.sql")
	if err != nil {
		t.Fatalf("create database.sql entry: %v", err)
	}
	if _, err := entry.Write([]byte("SELECT 1;\n")); err != nil {
		t.Fatalf("write database.sql entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatalf("close backup archive: %v", err)
	}

	binDir := t.TempDir()
	invocationLogPath := filepath.Join(t.TempDir(), "psql-invocation.log")
	psqlScript := fmt.Sprintf("#!/bin/sh\nset -eu\nprintf '%%s\\n' \"$@\" > %q\n", invocationLogPath)
	prependExecutableToPath(t, binDir, "psql", psqlScript)

	if err := svc.Restore(filepath.Base(backupArchivePath)); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	invocationLog, err := os.ReadFile(invocationLogPath)
	if err != nil {
		t.Fatalf("read psql invocation log: %v", err)
	}
	invocationText := string(invocationLog)
	if !strings.Contains(invocationText, "--dbname="+svc.cfg.Database.DSN) {
		t.Fatalf("expected psql to receive DSN, got %q", invocationText)
	}
	if !strings.Contains(invocationText, string(filepath.Separator)+"database.sql") {
		t.Fatalf("expected psql to receive extracted database.sql path, got %q", invocationText)
	}
}

func prependExecutableToPath(t *testing.T, directory string, executableName string, scriptContents string) {
	t.Helper()

	executablePath := filepath.Join(directory, executableName)
	if err := os.WriteFile(executablePath, []byte(scriptContents), 0o755); err != nil {
		t.Fatalf("write %s helper: %v", executableName, err)
	}

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+originalPath)
}
