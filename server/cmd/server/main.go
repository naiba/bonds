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
	services.MigrateOAuthSettingsToProviders(db, systemSettingService)
	oauthProviderService := services.NewOAuthProviderService(db)
	oauthProviderService.SetSystemSettings(systemSettingService)
	oauthProviderService.ReloadProviders()

	migrateUploadDir(cfg.Storage.UploadDir)
	migrateModulesToContactPage(db)

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
