package services

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
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

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/database"
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
	db       *gorm.DB
	cfg      *config.Config
	settings *SystemSettingService
}

func NewBackupService(db *gorm.DB, cfg *config.Config) *BackupService {
	return &BackupService{db: db, cfg: cfg}
}

func (s *BackupService) SetSystemSettings(settings *SystemSettingService) {
	s.settings = settings
}

func (s *BackupService) getCronSpec() string {
	if s.settings != nil {
		return s.settings.GetWithDefault("backup.cron", s.cfg.Backup.Cron)
	}
	return s.cfg.Backup.Cron
}

func (s *BackupService) getRetention() int {
	if s.settings != nil {
		return s.settings.GetInt("backup.retention", s.cfg.Backup.Retention)
	}
	return s.cfg.Backup.Retention
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

	// --clean --if-exists 让导出文件先 DROP 再 CREATE。恢复的目标是正在使用的库，
	// 表都已存在：不带这两个参数的导出一上来就会在 CREATE TABLE 处报“已存在”，
	// 根本恢复不进去。
	connection, passwordEnvironment := postgresClientConnection(s.cfg.Database.DSN)
	cmd := exec.Command(pgDump, "--dbname="+connection, "--clean", "--if-exists", "-f", tmpFile)
	cmd.Env = append(os.Environ(), passwordEnvironment...)
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

	// 备份里的库结构属于生成它的那个版本。恢复旧版本的备份后必须立刻把结构升到当前
	// 版本：否则之后新增的列和表都不存在，凡是用到它们的请求都会报错，直到有人重启服务。
	// 放在数据和附件都恢复完之后，这一步失败时数据本身已经完整，重启会再迁移一次。
	if err := database.AutoMigrate(s.db); err != nil {
		return fmt.Errorf("migrate restored database to the current schema: %w", err)
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

// restoreSQLite 用 SQLite 的在线备份接口，把备份文件的内容经由应用自己的连接写回
// 正在使用的数据库。
//
// 不能直接覆盖数据库文件（早期实现是 os.Create(DSN) 后整文件拷贝）：
//   - 服务运行在 WAL 模式下，其他连接和 -wal 文件里还保留着备份之后提交的内容，
//     覆盖主文件后读到的仍是“恢复之前”的数据，或者直接得到一个损坏的库；
//   - DB_DSN 可以带驱动参数（bonds.db?_busy_timeout=5000），把整个 DSN 当文件名
//     会写出一个同名的野文件，真正的数据库原封不动，却报恢复成功。
//
// 在线备份接口按页复制并遵守 SQLite 自己的锁和日志，连接池里的其他连接随后
// 读到的就是恢复后的内容。
func (s *BackupService) restoreSQLite(tmpDir string) error {
	backupDB := filepath.Join(tmpDir, "database.db")
	if _, err := os.Stat(backupDB); err != nil {
		return fmt.Errorf("database.db not found in backup: %w", err)
	}

	dbPath := s.cfg.Database.DSN
	if dbPath == "" || dbPath == ":memory:" {
		return fmt.Errorf("cannot restore to in-memory database")
	}

	ctx := context.Background()
	source, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(backupDB)+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open backup db: %w", err)
	}
	defer source.Close()
	sourceConn, err := source.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open backup db: %w", err)
	}
	defer sourceConn.Close()

	live, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("access live db: %w", err)
	}
	liveConn, err := live.Conn(ctx)
	if err != nil {
		return fmt.Errorf("access live db: %w", err)
	}
	defer liveConn.Close()

	return sourceConn.Raw(func(sourceDriverConn interface{}) error {
		return liveConn.Raw(func(liveDriverConn interface{}) error {
			sourceSQLite, ok := sourceDriverConn.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("backup db: unexpected driver connection %T", sourceDriverConn)
			}
			liveSQLite, ok := liveDriverConn.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("live db: unexpected driver connection %T", liveDriverConn)
			}
			return copySQLiteDatabase(liveSQLite, sourceSQLite)
		})
	})
}

// copySQLiteDatabase 把 source 的全部页复制到 destination。其他连接持有读事务时
// 一步会返回 busy / locked，稍等后重试即可，持续拿不到锁则放弃并报错。
func copySQLiteDatabase(destination, source *sqlite3.SQLiteConn) error {
	backup, err := destination.Backup("main", source, "main")
	if err != nil {
		return fmt.Errorf("start sqlite restore: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		done, stepErr := backup.Step(-1)
		if done {
			break
		}
		if stepErr != nil {
			var sqliteErr sqlite3.Error
			retryable := errors.As(stepErr, &sqliteErr) && (sqliteErr.Code == sqlite3.ErrBusy || sqliteErr.Code == sqlite3.ErrLocked)
			if !retryable || time.Now().After(deadline) {
				_ = backup.Finish()
				return fmt.Errorf("copy sqlite pages: %w", stepErr)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := backup.Finish(); err != nil {
		return fmt.Errorf("finish sqlite restore: %w", err)
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

	cleanDump, err := postgresDumpDropsObjects(dumpFile)
	if err != nil {
		return fmt.Errorf("inspect database.sql: %w", err)
	}
	// ON_ERROR_STOP 必不可少：psql 默认遇到 SQL 错误只打印后继续并以 0 退出，一次什么都
	// 没恢复的导入也会报成功，随后 Restore 还会照常清空并替换上传目录。
	// --single-transaction 让失败的恢复整体回滚，数据库保持原样。
	connection, passwordEnvironment := postgresClientConnection(s.cfg.Database.DSN)
	args := []string{
		"--dbname=" + connection, "-v", "ON_ERROR_STOP=1", "--single-transaction",
		"-c", "SET lock_timeout = '" + postgresRestoreLockTimeout + "'",
		"-c", lockOwnedTablesSQL,
	}
	if !cleanDump {
		// 旧版本生成的备份不带 DROP 语句，而恢复的目标库里表都已存在。
		// 实测（PostgreSQL 18）直接灌这种导出文件从来就恢复不完整：CREATE 全部报“已存在”，
		// COPY 在已有数据的库上因主键冲突整表失败，在刚迁移好的空库上也会因为外键顺序
		// 丢表（contacts 先于 vaults 导入）。所以先在同一个事务里删掉已有的表，
		// 再让导出文件自己把结构和数据建回来。
		// 只删当前角色在 public 里拥有的表和序列，不能用 DROP SCHEMA public CASCADE：
		// 那会连同库里其他应用的对象一起删掉，而且 PostgreSQL 15 起应用角色通常
		// 不是 public 的属主，根本没有权限执行。
		args = append(args, "-c", dropOwnedPublicTablesSQL)
	}
	args = append(args, "-f", dumpFile)
	cmd := exec.Command(psql, args...)
	cmd.Env = append(os.Environ(), passwordEnvironment...)
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
	retention := s.getRetention()
	if retention <= 0 {
		return nil
	}

	entries, err := os.ReadDir(s.cfg.Backup.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backup dir: %w", err)
	}

	cutoff := time.Now().Add(-time.Duration(retention) * 24 * time.Hour)

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
	cronSpec := s.getCronSpec()
	return dto.BackupConfigResponse{
		CronEnabled:   cronSpec != "",
		CronSpec:      cronSpec,
		RetentionDays: s.getRetention(),
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

// postgresDumpDropsObjects 判断导出文件是否由 pg_dump --clean 生成（建表前先 DROP）。
// DROP 语句集中在文件开头、第一张表的数据之前，所以只需要看到第一条 COPY 为止。
func postgresDumpDropsObjects(dumpFile string) (bool, error) {
	f, err := os.Open(dumpFile)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DROP ") {
			return true, nil
		}
		if strings.HasPrefix(line, "COPY ") {
			return false, nil
		}
	}
	// 超长行（bufio.ErrTooLong）只可能出现在数据区，按旧格式处理即可。
	return false, nil
}
