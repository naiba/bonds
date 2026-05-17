package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/cron"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dav"
	"github.com/naiba/bonds/internal/frontend"
	"github.com/naiba/bonds/internal/handlers"
	appMiddleware "github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

var Version = "dev"

//	@title			Bonds API
//	@version		1.0
//	@description	Personal relationship manager RESTful API.

//	@contact.name	Bonds Team
//	@contact.url	https://github.com/naiba/bonds

//	@license.name	AGPL-3.0
//	@license.url	https://www.gnu.org/licenses/agpl-3.0.html

//	@BasePath	/api

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Format: "Bearer {token}"

func main() {
	cfg := config.Load()

	db, err := database.Connect(&cfg.Database, cfg.Debug)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := models.SeedCurrencies(db); err != nil {
		log.Fatalf("Failed to seed currencies: %v", err)
	}

	systemSettingService := services.NewSystemSettingService(db)
	if err := services.SeedSettingsFromEnv(systemSettingService, cfg); err != nil {
		log.Fatalf("Failed to seed system settings: %v", err)
	}
	oauthProviderService := services.NewOAuthProviderService(db)
	oauthProviderService.SetSystemSettings(systemSettingService)
	oauthProviderService.ReloadProviders()

	migrateUploadDir(cfg.Storage.UploadDir)
	migrateModulesToContactPage(db)
	if err := migrateContactTasksToManyToMany(db); err != nil {
		log.Fatalf("Failed to migrate contact_tasks to many-to-many: %v", err)
	}
	services.BackfillImportantDateReminderSchedules(db)
	if err := models.BackfillTaskStatuses(db); err != nil {
		log.Printf("WARNING: failed to backfill task statuses: %v", err)
	}

	scheduler := cron.NewScheduler(db)
	scheduler.Start()

	mailer := services.NewDynamicMailer(systemSettingService)
	notificationSender := services.NewShoutrrrSender()
	reminderScheduler := services.NewReminderSchedulerService(db, mailer, notificationSender)
	if err := scheduler.RegisterJob("0 * * * * *", "process_reminders", func() {
		reminderScheduler.ProcessDueReminders()
	}); err != nil {
		log.Printf("WARNING: Failed to register reminder cron job: %v", err)
	}

	vcardService := services.NewVCardService(db)
	davClientService := services.NewDavClientService(db, cfg.JWT.Secret)
	davSyncService := services.NewDavSyncService(db, davClientService, vcardService)
	if err := scheduler.RegisterJob("0 */5 * * * *", "sync_address_books", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		if err := davSyncService.SyncAllDue(ctx); err != nil {
			log.Printf("[cron] sync_address_books error: %v", err)
		}
	}); err != nil {
		log.Printf("WARNING: Failed to register DAV sync cron job: %v", err)
	}

	backupService := services.NewBackupService(db, cfg)
	backupService.SetSystemSettings(systemSettingService)
	backupCron := systemSettingService.GetWithDefault("backup.cron", cfg.Backup.Cron)
	if backupCron != "" {
		if err := scheduler.RegisterJob(backupCron, "create_backup", func() {
			if _, err := backupService.Create(); err != nil {
				log.Printf("WARNING: Backup cron failed: %v", err)
			}
			if err := backupService.CleanOldBackups(); err != nil {
				log.Printf("WARNING: Backup cleanup failed: %v", err)
			}
		}); err != nil {
			log.Printf("WARNING: Failed to register backup cron job: %v", err)
		}
	}

	e := echo.New()
	e.HideBanner = true

	if cfg.Debug {
		e.Use(echoMiddleware.Logger())
	}
	e.Use(echoMiddleware.Recover())
	e.Use(appMiddleware.Locale())

	handlers.RegisterRoutes(e, db, cfg, Version)

	dav.SetupDAVRoutes(e, db)

	if frontend.HasDistFiles() {
		frontend.RegisterSPARoutes(e)
		log.Println("Serving embedded frontend")
	} else {
		log.Println("No embedded frontend found, API-only mode")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := e.Start(addr); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	cronCtx := scheduler.Stop()
	select {
	case <-cronCtx.Done():
		log.Println("Cron scheduler stopped")
	case <-time.After(30 * time.Second):
		log.Println("Cron scheduler stop timed out")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}

func migrateModulesToContactPage(db *gorm.DB) {
	var moduleIDs []uint
	db.Raw(`SELECT id FROM modules WHERE type IN ('addresses','contact_information')`).Scan(&moduleIDs)
	if len(moduleIDs) == 0 {
		return
	}
	result := db.Exec(`
		UPDATE module_template_pages SET template_page_id = (
			SELECT tp2.id FROM template_pages tp2
			WHERE tp2.slug = 'contact'
			AND tp2.template_id = (
				SELECT tp1.template_id FROM template_pages tp1 WHERE tp1.id = module_template_pages.template_page_id
			)
			LIMIT 1
		)
		WHERE module_id IN (?)
		AND template_page_id IN (SELECT id FROM template_pages WHERE slug = 'social')
	`, moduleIDs)
	if result.RowsAffected > 0 {
		log.Printf("Migrated %d module-page bindings from 'social' to 'contact' page", result.RowsAffected)
	}
}

// migrateContactTasksToManyToMany copies any pre-existing single contact_id
// values from contact_tasks into the new task_contacts pivot, then drops the
// old column. Idempotent: a no-op once the column is gone.
//
// Failures are fatal to the caller: once the application code no longer
// reads contact_id, any legacy rows that did not make it into task_contacts
// would silently disappear from the UI. Aborting at startup keeps the
// operator in control instead.
//
// The column drop is dialect-aware because the original schema declared a
// named FK constraint (fk_contacts_tasks) on contact_id. Plain DROP COLUMN
// fails on both SQLite (GORM's table-rewrite parser chokes on the stale FK)
// and Postgres (FK blocks the drop). We rewrite the table on SQLite and use
// CASCADE on Postgres.
func migrateContactTasksToManyToMany(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasColumn(&models.ContactTask{}, "contact_id") {
		return nil
	}

	copied := db.Exec(`
		INSERT INTO task_contacts (contact_task_id, contact_id, created_at, updated_at)
		SELECT id, contact_id, COALESCE(created_at, CURRENT_TIMESTAMP), COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM contact_tasks
		WHERE contact_id IS NOT NULL AND contact_id <> ''
		  AND NOT EXISTS (
		    SELECT 1 FROM task_contacts tc
		    WHERE tc.contact_task_id = contact_tasks.id AND tc.contact_id = contact_tasks.contact_id
		  )
	`)
	if copied.Error != nil {
		return fmt.Errorf("copy contact_tasks.contact_id into task_contacts: %w", copied.Error)
	}

	var err error
	switch db.Dialector.Name() {
	case "sqlite":
		err = dropContactIDColumnSQLite(db)
	case "postgres":
		err = db.Exec(`ALTER TABLE contact_tasks DROP COLUMN IF EXISTS contact_id CASCADE`).Error
	default:
		err = m.DropColumn(&models.ContactTask{}, "contact_id")
	}
	if err != nil {
		return fmt.Errorf("drop contact_tasks.contact_id: %w", err)
	}
	log.Printf("Migrated %d contact_tasks.contact_id rows into task_contacts; dropped legacy column", copied.RowsAffected)
	return nil
}

// dropContactIDColumnSQLite rewrites contact_tasks without the contact_id
// column. SQLite has no ALTER TABLE DROP CONSTRAINT, and GORM's DropColumn
// path fails to recreate the table while a stale FK references the column
// being dropped. Strategy: rename → AutoMigrate (recreates the new schema)
// → INSERT … SELECT (preserving every other column) → drop the renamed
// original.
//
// Two PRAGMAs must be set on the SAME connection BEFORE the transaction
// (both are no-ops inside a transaction, and PRAGMAs only apply to the
// connection that set them):
//
//   - foreign_keys = OFF: avoids cascading deletes during the swap.
//   - legacy_alter_table = ON: stops ALTER TABLE RENAME from rewriting
//     foreign-key references in OTHER tables. Without this, modern SQLite
//     would silently point task_contacts.contact_task_id at the renamed
//     contact_tasks_old, and the subsequent DROP TABLE contact_tasks_old
//     would leave that FK dangling.
func dropContactIDColumnSQLite(db *gorm.DB) error {
	type colInfo struct{ Name string }
	var infos []colInfo
	if err := db.Raw(`SELECT name FROM pragma_table_info('contact_tasks')`).Scan(&infos).Error; err != nil {
		return fmt.Errorf("inspect contact_tasks columns: %w", err)
	}
	cols := make([]string, 0, len(infos))
	for _, c := range infos {
		if c.Name == "contact_id" {
			continue
		}
		cols = append(cols, fmt.Sprintf("%q", c.Name))
	}
	if len(cols) == 0 {
		return fmt.Errorf("no columns to preserve")
	}
	colList := joinStrings(cols, ", ")

	return db.Connection(func(conn *gorm.DB) error {
		if err := conn.Exec(`PRAGMA foreign_keys = OFF`).Error; err != nil {
			return err
		}
		defer conn.Exec(`PRAGMA foreign_keys = ON`)
		if err := conn.Exec(`PRAGMA legacy_alter_table = ON`).Error; err != nil {
			return err
		}
		defer conn.Exec(`PRAGMA legacy_alter_table = OFF`)

		return conn.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`ALTER TABLE contact_tasks RENAME TO contact_tasks_old`).Error; err != nil {
				return err
			}
			var oldIndexNames []string
			if err := tx.Raw(`SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'contact_tasks_old' AND name NOT LIKE 'sqlite_autoindex_%'`).Scan(&oldIndexNames).Error; err != nil {
				return fmt.Errorf("inspect orphan indexes: %w", err)
			}
			for _, idx := range oldIndexNames {
				if err := tx.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %q", idx)).Error; err != nil {
					return fmt.Errorf("drop orphan index %s: %w", idx, err)
				}
			}
			if err := tx.AutoMigrate(&models.ContactTask{}); err != nil {
				return err
			}
			if err := tx.Exec(fmt.Sprintf("INSERT INTO contact_tasks (%s) SELECT %s FROM contact_tasks_old", colList, colList)).Error; err != nil {
				return err
			}
			return tx.Exec(`DROP TABLE contact_tasks_old`).Error
		})
	})
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func migrateUploadDir(currentDir string) {
	if currentDir != "data/uploads" {
		return
	}
	oldDir := "uploads"
	if info, err := os.Stat(oldDir); err != nil || !info.IsDir() {
		return
	}
	if _, err := os.Stat(currentDir); err == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(currentDir), 0o755); err != nil {
		log.Printf("WARNING: failed to create parent for %s: %v", currentDir, err)
		return
	}
	if err := os.Rename(oldDir, currentDir); err != nil {
		log.Printf("WARNING: failed to migrate %s -> %s: %v", oldDir, currentDir, err)
		return
	}
	log.Printf("Migrated upload directory: %s -> %s", oldDir, currentDir)
}
