package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/naiba/bonds/internal/config"
)

// PRAGMA foreign_keys is a per-connection setting. Running it once after
// connecting only configures the single pooled connection that happened to
// execute it; every connection the pool opens later enforces nothing.
func TestConnectSQLiteEnablesForeignKeysOnEveryPooledConnection(t *testing.T) {
	for name, dsnSuffix := range map[string]string{"plain path": "", "dsn with parameters": "?_busy_timeout=5000"} {
		t.Run(name, func(t *testing.T) {
			dsn := filepath.Join(t.TempDir(), "bonds.db") + dsnSuffix
			db, err := Connect(&config.DatabaseConfig{Driver: "sqlite", DSN: dsn}, false)
			if err != nil {
				t.Fatalf("Connect failed: %v", err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatalf("access pool: %v", err)
			}
			t.Cleanup(func() { sqlDB.Close() })

			// Hold several connections at once so the pool has to open new ones.
			ctx := context.Background()
			var held []*sql.Conn
			t.Cleanup(func() {
				for _, conn := range held {
					conn.Close()
				}
			})
			for i := 0; i < 4; i++ {
				conn, err := sqlDB.Conn(ctx)
				if err != nil {
					t.Fatalf("open pooled connection %d: %v", i, err)
				}
				held = append(held, conn)
				var enabled int
				if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
					t.Fatalf("read foreign_keys on connection %d: %v", i, err)
				}
				if enabled != 1 {
					t.Fatalf("pooled connection %d has foreign_keys = %d, want 1", i, enabled)
				}
			}
		})
	}
}
