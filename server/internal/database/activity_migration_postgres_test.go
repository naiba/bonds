package database

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateStandaloneLifeEventSchemaToActivitiesPostgres(t *testing.T) {
	db, schemaName := openPostgresActivityMigrationTestDB(t)
	statements := []string{
		`CREATE TABLE life_events (id bigserial PRIMARY KEY, vault_id text NOT NULL, life_event_type_id bigint, title text NOT NULL)`,
		`CREATE INDEX idx_life_events_vault_id ON life_events (vault_id)`,
		`INSERT INTO life_events (vault_id, life_event_type_id, title) VALUES ('vault-1', 7, 'Dinner')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare standalone PostgreSQL schema: %v", err)
		}
	}

	if err := migrateActivitySchema(db); err != nil {
		t.Fatalf("migrate standalone PostgreSQL schema: %v", err)
	}
	if db.Migrator().HasTable("life_events") || !db.Migrator().HasTable("activities") {
		t.Fatal("standalone PostgreSQL activity table was not renamed")
	}
	var legacyIndexCount int64
	if err := db.Raw(
		"SELECT count(*) FROM pg_indexes WHERE schemaname = ? AND indexname = ?",
		schemaName,
		"idx_life_events_vault_id",
	).Scan(&legacyIndexCount).Error; err != nil {
		t.Fatalf("inspect migrated PostgreSQL indexes: %v", err)
	}
	if legacyIndexCount != 0 {
		t.Fatalf("legacy PostgreSQL index still exists: count=%d", legacyIndexCount)
	}
}

func TestMigrateLegacyTimelineActivitiesPostgres(t *testing.T) {
	db, _ := openPostgresActivityMigrationTestDB(t)
	statements := []string{
		`CREATE TABLE timeline_events (id bigserial PRIMARY KEY, vault_id text NOT NULL, started_at date NOT NULL, label text, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE life_events (id bigserial PRIMARY KEY, timeline_event_id bigint NOT NULL, life_event_type_id bigint NOT NULL, emotion_id bigint, happened_at date NOT NULL, calendar_type text, original_day bigint, original_month bigint, original_year bigint, collapsed boolean, summary text, description text, costs bigint, currency_id bigint, paid_by_contact_id text, duration_in_minutes bigint, distance bigint, distance_unit text, from_place text, to_place text, place text, created_at timestamptz, updated_at timestamptz)`,
		`CREATE INDEX idx_life_events_timeline_event_id ON life_events (timeline_event_id)`,
		`CREATE INDEX idx_life_events_life_event_type_id ON life_events (life_event_type_id)`,
		`CREATE INDEX idx_life_events_emotion_id ON life_events (emotion_id)`,
		`CREATE INDEX idx_life_events_currency_id ON life_events (currency_id)`,
		`CREATE INDEX idx_life_events_paid_by_contact_id ON life_events (paid_by_contact_id)`,
		`INSERT INTO timeline_events (id, vault_id, started_at, label, created_at, updated_at) VALUES (1, 'vault-1', '2026-01-01', 'Dinner', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		`INSERT INTO life_events (id, timeline_event_id, life_event_type_id, happened_at, summary, created_at, updated_at) VALUES (3, 1, 7, '2026-01-02', 'Dinner', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare timeline PostgreSQL schema: %v", err)
		}
	}

	if err := migrateActivitySchema(db); err != nil {
		t.Fatalf("migrate timeline PostgreSQL schema: %v", err)
	}
	for _, tableName := range []string{"life_events", "legacy_activities_v1", "timeline_events"} {
		if db.Migrator().HasTable(tableName) {
			t.Fatalf("legacy PostgreSQL table %s still exists", tableName)
		}
	}
	var activity models.Activity
	if err := db.First(&activity, 3).Error; err != nil {
		t.Fatalf("load migrated PostgreSQL activity: %v", err)
	}
	if activity.Title != "Dinner" || activity.ActivityTypeID == nil || *activity.ActivityTypeID != 7 {
		t.Fatalf("migrated PostgreSQL activity = %+v", activity)
	}
}

func openPostgresActivityMigrationTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	dsn := os.Getenv("BONDS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("BONDS_TEST_POSTGRES_DSN is not configured")
	}

	schemaName := fmt.Sprintf("activity_migration_%d", time.Now().UnixNano())
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	if err := admin.Exec("CREATE SCHEMA " + quoteIdentifier(schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL test schema: %v", err)
	}

	testDSN := dsn
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, parseErr := url.Parse(dsn)
		if parseErr != nil {
			t.Fatalf("parse PostgreSQL test DSN: %v", parseErr)
		}
		query := parsed.Query()
		query.Set("search_path", schemaName)
		parsed.RawQuery = query.Encode()
		testDSN = parsed.String()
	} else {
		testDSN += " search_path=" + schemaName
	}
	db, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("connect to PostgreSQL test schema: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec("DROP SCHEMA IF EXISTS " + quoteIdentifier(schemaName) + " CASCADE").Error
		if sqlDB, sqlErr := admin.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db, schemaName
}
