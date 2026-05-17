package database

import (
	"fmt"
	"log"
	"strings"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/models"
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
	if err := migrateLegacyContactTasks(db); err != nil {
		return err
	}
	return db.AutoMigrate(AllModels()...)
}

func migrateLegacyContactTasks(db *gorm.DB) error {
	if !db.Migrator().HasTable("contact_tasks") {
		return nil
	}
	if !hasColumn(db, "contact_tasks", "contact_id") {
		return nil
	}
	if db.Dialector.Name() == "sqlite" {
		return migrateLegacyContactTasksSQLite(db)
	}
	return migrateLegacyContactTasksPortable(db)
}

func migrateLegacyContactTasksSQLite(db *gorm.DB) error {
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
			if err := backfillLegacyContactTaskVaultID(tx); err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE contact_tasks RENAME TO contact_tasks_old`).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: rename legacy table: %w", err)
			}

			var oldIndexNames []string
			if err := tx.Raw(`SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'contact_tasks_old' AND name NOT LIKE 'sqlite_autoindex_%'`).Scan(&oldIndexNames).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: inspect legacy indexes: %w", err)
			}
			for _, indexName := range oldIndexNames {
				if err := tx.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteIdentifier(indexName))).Error; err != nil {
					return fmt.Errorf("migrate legacy contact_tasks: drop legacy index %s: %w", indexName, err)
				}
			}

			if err := tx.AutoMigrate(&models.ContactTask{}, &models.TaskContact{}); err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: create current schema: %w", err)
			}
			copyColumns, err := sharedColumns(tx, "contact_tasks_old", "contact_tasks", "contact_id")
			if err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: select copy columns: %w", err)
			}
			columnList := strings.Join(quoteIdentifiers(copyColumns), ", ")
			if err := tx.Exec(fmt.Sprintf("INSERT INTO contact_tasks (%s) SELECT %s FROM contact_tasks_old", columnList, columnList)).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: copy current table rows: %w", err)
			}
			if err := backfillLegacyTaskContacts(tx, "contact_tasks_old"); err != nil {
				return err
			}
			if err := tx.Exec(`DROP TABLE contact_tasks_old`).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: drop legacy table: %w", err)
			}
			return nil
		})
	})
}

func backfillLegacyContactTaskVaultID(tx *gorm.DB) error {
	if !hasColumn(tx, "contact_tasks", "vault_id") {
		if err := tx.Exec("ALTER TABLE contact_tasks ADD COLUMN vault_id text").Error; err != nil {
			return fmt.Errorf("migrate legacy contact_tasks: add vault_id: %w", err)
		}
	}
	if err := tx.Exec(`
			UPDATE contact_tasks
			SET vault_id = (
				SELECT contacts.vault_id
				FROM contacts
				WHERE contacts.id = contact_tasks.contact_id
			)
			WHERE vault_id IS NULL OR vault_id = ''
		`).Error; err != nil {
		return fmt.Errorf("migrate legacy contact_tasks: backfill vault_id: %w", err)
	}

	var missingVaultCount int64
	if err := tx.Table("contact_tasks").Where("vault_id IS NULL OR vault_id = ?", "").Count(&missingVaultCount).Error; err != nil {
		return fmt.Errorf("migrate legacy contact_tasks: validate vault_id: %w", err)
	}
	if missingVaultCount > 0 {
		return fmt.Errorf("migrate legacy contact_tasks: %d task(s) cannot be mapped to a vault", missingVaultCount)
	}
	return nil
}

func backfillLegacyTaskContacts(tx *gorm.DB, sourceTable string) error {
	if !tx.Migrator().HasTable("task_contacts") {
		if err := tx.Migrator().CreateTable(&models.TaskContact{}); err != nil {
			return fmt.Errorf("migrate legacy contact_tasks: create task_contacts: %w", err)
		}
	}
	insertPrefix := "INSERT INTO"
	insertSuffix := ""
	if tx.Dialector.Name() == "sqlite" {
		insertPrefix = "INSERT OR IGNORE INTO"
	} else if tx.Dialector.Name() == "postgres" {
		insertSuffix = " ON CONFLICT DO NOTHING"
	}
	if err := tx.Exec(fmt.Sprintf(`
			%s task_contacts (contact_task_id, contact_id, created_at, updated_at)
			SELECT id, contact_id, COALESCE(created_at, CURRENT_TIMESTAMP), COALESCE(updated_at, CURRENT_TIMESTAMP)
			FROM %s
			WHERE contact_id IS NOT NULL AND contact_id <> ''
%s`, insertPrefix, quoteIdentifier(sourceTable), insertSuffix)).Error; err != nil {
		return fmt.Errorf("migrate legacy contact_tasks: backfill task_contacts: %w", err)
	}
	return nil
}

func migrateLegacyContactTasksPortable(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if !tx.Migrator().HasColumn(&models.ContactTask{}, "vault_id") {
			if err := tx.Exec(`ALTER TABLE contact_tasks ADD COLUMN vault_id text`).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: add vault_id: %w", err)
			}
		}
		if err := backfillLegacyContactTaskVaultID(tx); err != nil {
			return err
		}
		if err := backfillLegacyTaskContacts(tx, "contact_tasks"); err != nil {
			return err
		}
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec(`ALTER TABLE contact_tasks ALTER COLUMN vault_id SET NOT NULL`).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: enforce vault_id not null: %w", err)
			}
			if err := tx.Exec(`ALTER TABLE contact_tasks DROP COLUMN IF EXISTS contact_id CASCADE`).Error; err != nil {
				return fmt.Errorf("migrate legacy contact_tasks: drop contact_id: %w", err)
			}
			return nil
		}
		if err := tx.Migrator().DropColumn(&models.ContactTask{}, "contact_id"); err != nil {
			return fmt.Errorf("migrate legacy contact_tasks: drop contact_id: %w", err)
		}
		return nil
	})
}

func hasColumn(db *gorm.DB, tableName, columnName string) bool {
	if db.Dialector.Name() == "postgres" {
		var count int64
		err := db.Raw(`
			SELECT COUNT(*)
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			AND table_name = ?
			AND column_name = ?
		`, tableName, columnName).Scan(&count).Error
		return err == nil && count > 0
	}

	rows, err := db.Raw("PRAGMA table_info(" + quoteIdentifier(tableName) + ")").Rows()
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}

func sharedColumns(tx *gorm.DB, sourceTable, targetTable string, excludedColumns ...string) ([]string, error) {
	sourceColumns, err := tableColumns(tx, sourceTable)
	if err != nil {
		return nil, err
	}
	targetColumns, err := tableColumns(tx, targetTable)
	if err != nil {
		return nil, err
	}
	targetSet := make(map[string]bool, len(targetColumns))
	for _, column := range targetColumns {
		targetSet[column] = true
	}
	excludedSet := make(map[string]bool, len(excludedColumns))
	for _, column := range excludedColumns {
		excludedSet[column] = true
	}

	columns := make([]string, 0, len(sourceColumns))
	for _, column := range sourceColumns {
		if targetSet[column] && !excludedSet[column] {
			columns = append(columns, column)
		}
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no shared columns between %s and %s", sourceTable, targetTable)
	}
	return columns, nil
}

func tableColumns(tx *gorm.DB, tableName string) ([]string, error) {
	rows, err := tx.Raw("PRAGMA table_info(" + quoteIdentifier(tableName) + ")").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

func quoteIdentifiers(names []string) []string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, quoteIdentifier(name))
	}
	return quoted
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
