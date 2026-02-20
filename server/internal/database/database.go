package database

import (
	"fmt"
	"log"

	"github.com/naiba/bonds/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig, debug bool) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}
	usePrepareStmt := cfg.Driver != "sqlite"
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: false,
		PrepareStmt:                              usePrepareStmt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode for SQLite
	if cfg.Driver == "sqlite" {
		if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
			log.Printf("Warning: failed to enable WAL mode: %v", err)
		}
		if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
			log.Printf("Warning: failed to enable foreign keys: %v", err)
		}
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}
