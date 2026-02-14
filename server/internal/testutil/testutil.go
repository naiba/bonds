package testutil

import (
	"testing"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}
	return db
}

func TestJWTConfig() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:     "test-secret-key",
		ExpiryHrs:  24,
		RefreshHrs: 168,
	}
}
