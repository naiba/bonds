package services

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"gorm.io/gorm"
)

var (
	ErrBackupNotFound        = errors.New("backup not found")
	ErrBackupInvalidFilename = errors.New("invalid backup filename")
	ErrPgDumpNotFound        = errors.New("pg_dump not found, cannot backup PostgreSQL")
)

// validBackupFilename matches bonds-YYYY-MM-DD-HHmmss.zip
var validBackupFilename = regexp.MustCompile(`^bonds-\d{4}-\d{2}-\d{2}-\d{6}\.zip$`)

type BackupService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewBackupService(db *gorm.DB, cfg *config.Config) *BackupService {
	return &BackupService{db: db, cfg: cfg}
}

// Create creates a new backup zip containing the database and uploads directory.
func (s *BackupService) Create() (*dto.BackupResponse, error) {
	if err := os.MkdirAll(s.cfg.Backup.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	now := time.Now()
	filename := fmt.Sprintf("bonds-%s.zip", now.Format("2006-01-02-150405"))
	zipPath := filepath.Join(s.cfg.Backup.Dir, filename)

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return nil, fmt.Errorf("create zip file: %w", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// Backup database
	if s.cfg.Database.Driver == "postgres" {
		if err := s.backupPostgres(zw); err != nil {
			os.Remove(zipPath)
			return nil, err
		}
	} else {
		if err := s.backupSQLite(zw); err != nil {
			os.Remove(zipPath)
			return nil, err
		}
	}

	// Backup uploads directory
	if err := s.backupUploads(zw); err != nil {
		os.Remove(zipPath)
		return nil, err
	}

	// Close zip writer to flush
	if err := zw.Close(); err != nil {
		os.Remove(zipPath)
		return nil, fmt.Errorf("close zip: %w", err)
	}

	info, err := os.Stat(zipPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	return &dto.BackupResponse{
		Filename:  filename,
		Size:      info.Size(),
		CreatedAt: now,
	}, nil
}

func (s *BackupService) backupSQLite(zw *zip.Writer) error {
	// Create a temp file for VACUUM INTO
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("bonds-backup-%d.db", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	// VACUUM INTO creates an atomic copy of the database
	if err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpFile)).Error; err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}

	// Add the DB copy to zip
	w, err := zw.Create("database.db")
	if err != nil {
		return fmt.Errorf("create zip entry: %w", err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		return fmt.Errorf("open temp db: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy db to zip: %w", err)
	}

	return nil
}

func (s *BackupService) backupPostgres(zw *zip.Writer) error {
	pgDump, err := exec.LookPath("pg_dump")
	if err != nil {
		return ErrPgDumpNotFound
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("bonds-backup-%d.sql", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	cmd := exec.Command(pgDump, "--dbname="+s.cfg.Database.DSN, "-f", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump failed: %s: %w", string(output), err)
	}

	w, err := zw.Create("database.sql")
	if err != nil {
		return fmt.Errorf("create zip entry: %w", err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		return fmt.Errorf("open dump file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy dump to zip: %w", err)
	}

	return nil
}

func (s *BackupService) backupUploads(zw *zip.Writer) error {
	uploadDir := s.cfg.Storage.UploadDir
	if uploadDir == "" {
		return nil
	}

	info, err := os.Stat(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no uploads to backup
		}
		return fmt.Errorf("stat upload dir: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.Walk(uploadDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(uploadDir, path)
		if err != nil {
			return err
		}
		zipPath := filepath.Join("uploads", relPath)

		if fi.IsDir() {
			// Add trailing slash for directories
			if !strings.HasSuffix(zipPath, "/") {
				zipPath += "/"
			}
			_, err := zw.Create(zipPath)
			return err
		}

		w, err := zw.Create(zipPath)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
}

// List returns all backup files sorted by creation time descending.
func (s *BackupService) List() ([]dto.BackupResponse, error) {
	if err := os.MkdirAll(s.cfg.Backup.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	entries, err := os.ReadDir(s.cfg.Backup.Dir)
	if err != nil {
		return nil, fmt.Errorf("read backup dir: %w", err)
	}

	var backups []dto.BackupResponse
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, dto.BackupResponse{
			Filename:  entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Sort by time descending (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// GetFilePath validates the filename and returns its full path.
func (s *BackupService) GetFilePath(filename string) (string, error) {
	if err := s.validateFilename(filename); err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.cfg.Backup.Dir, filename)
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", ErrBackupNotFound
		}
		return "", fmt.Errorf("stat backup: %w", err)
	}

	return fullPath, nil
}

// Delete removes a backup file.
func (s *BackupService) Delete(filename string) error {
	fullPath, err := s.GetFilePath(filename)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

// Restore restores from a backup zip file.
func (s *BackupService) Restore(filename string) error {
	fullPath, err := s.GetFilePath(filename)
	if err != nil {
		return err
	}

	// Open zip
	r, err := zip.OpenReader(fullPath)
	if err != nil {
		return fmt.Errorf("open backup zip: %w", err)
	}
	defer r.Close()

	// Create temp dir for extraction
	tmpDir, err := os.MkdirTemp("", "bonds-restore-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract all files
	for _, f := range r.File {
		if err := s.extractZipFile(f, tmpDir); err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}

	// Restore database
	if s.cfg.Database.Driver == "postgres" {
		if err := s.restorePostgres(tmpDir); err != nil {
			return err
		}
	} else {
		if err := s.restoreSQLite(tmpDir); err != nil {
			return err
		}
	}

	// Restore uploads
	if err := s.restoreUploads(tmpDir); err != nil {
		return err
	}

	return nil
}

func (s *BackupService) extractZipFile(f *zip.File, destDir string) error {
	// Prevent path traversal
	name := filepath.Clean(f.Name)
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid path in zip: %s", f.Name)
	}

	destPath := filepath.Join(destDir, name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func (s *BackupService) restoreSQLite(tmpDir string) error {
	backupDB := filepath.Join(tmpDir, "database.db")
	if _, err := os.Stat(backupDB); err != nil {
		return fmt.Errorf("database.db not found in backup: %w", err)
	}

	dbPath := s.cfg.Database.DSN
	if dbPath == "" || dbPath == ":memory:" {
		return fmt.Errorf("cannot restore to in-memory database")
	}

	// Copy backup DB over current DB
	src, err := os.Open(backupDB)
	if err != nil {
		return fmt.Errorf("open backup db: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("create db file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy db: %w", err)
	}

	return nil
}

func (s *BackupService) restorePostgres(tmpDir string) error {
	dumpFile := filepath.Join(tmpDir, "database.sql")
	if _, err := os.Stat(dumpFile); err != nil {
		return fmt.Errorf("database.sql not found in backup: %w", err)
	}

	psql, err := exec.LookPath("psql")
	if err != nil {
		return fmt.Errorf("psql not found: %w", err)
	}

	cmd := exec.Command(psql, "--dbname="+s.cfg.Database.DSN, "-f", dumpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("psql restore failed: %s: %w", string(output), err)
	}

	return nil
}

func (s *BackupService) restoreUploads(tmpDir string) error {
	backupUploads := filepath.Join(tmpDir, "uploads")
	if _, err := os.Stat(backupUploads); err != nil {
		if os.IsNotExist(err) {
			return nil // no uploads in backup
		}
		return err
	}

	uploadDir := s.cfg.Storage.UploadDir
	if uploadDir == "" {
		return nil
	}

	// Remove existing uploads and replace with backup
	if err := os.RemoveAll(uploadDir); err != nil {
		return fmt.Errorf("remove existing uploads: %w", err)
	}

	return filepath.Walk(backupUploads, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(backupUploads, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(uploadDir, relPath)

		if fi.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		return err
	})
}

// CleanOldBackups removes backups older than the configured retention days.
func (s *BackupService) CleanOldBackups() error {
	if s.cfg.Backup.Retention <= 0 {
		return nil
	}

	entries, err := os.ReadDir(s.cfg.Backup.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backup dir: %w", err)
	}

	cutoff := time.Now().Add(-time.Duration(s.cfg.Backup.Retention) * 24 * time.Hour)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(s.cfg.Backup.Dir, entry.Name()))
		}
	}

	return nil
}

// GetConfig returns the current backup configuration.
func (s *BackupService) GetConfig() dto.BackupConfigResponse {
	return dto.BackupConfigResponse{
		CronEnabled:   s.cfg.Backup.Cron != "",
		CronSpec:      s.cfg.Backup.Cron,
		RetentionDays: s.cfg.Backup.Retention,
		BackupDir:     s.cfg.Backup.Dir,
		DBDriver:      s.cfg.Database.Driver,
	}
}

func (s *BackupService) validateFilename(filename string) error {
	if filename == "" {
		return ErrBackupInvalidFilename
	}
	// Prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		return ErrBackupInvalidFilename
	}
	if !strings.HasSuffix(filename, ".zip") {
		return ErrBackupInvalidFilename
	}
	if !validBackupFilename.MatchString(filename) {
		return ErrBackupInvalidFilename
	}
	return nil
}
