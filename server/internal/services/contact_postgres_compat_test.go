package services

import (
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// Regression tests for https://github.com/naiba/bonds/issues/85:
// PostgreSQL rejects `boolean_column = 1` with
// "operator does not exist: boolean = integer". SQLite accepts it, so these
// tests inspect the generated SQL instead of executing against Postgres.

func TestFavoriteOrderClauseIsPostgresCompatible(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	contactSvc := NewContactService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "pg-compat@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "V"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	if _, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice"}); err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	dryDB := db.Session(&gorm.Session{DryRun: true})
	stmt := dryDB.Model(&models.Contact{}).
		Where("vault_id = ? AND NOT (can_be_deleted = ? AND listed = ?)", vault.ID, false, false).
		Where("listed = ?", true).
		Order(favoriteOrderClause(resp.User.ID) + ", first_name ASC, last_name ASC").
		Find(&[]models.Contact{}).Statement

	sql := strings.ToLower(stmt.SQL.String())
	if strings.Contains(sql, "is_favorite = 1") || strings.Contains(sql, "is_favorite = 0") {
		t.Errorf("ORDER BY compares boolean is_favorite against integer literal; SQL: %s", sql)
	}

	if _, _, err := NewContactService(dryDB).ListContacts(vault.ID, resp.User.ID, 1, 15, "", "first_name", ""); err != nil {
		t.Fatalf("ListContacts (dry-run) failed: %v", err)
	}
}

func TestFavoriteOrderClauseStringIsPostgresCompatible(t *testing.T) {
	clause := favoriteOrderClause("some-user-id")
	if strings.Contains(clause, "is_favorite = 1") || strings.Contains(clause, "is_favorite = 0") {
		t.Errorf("favoriteOrderClause uses integer literal, breaks PostgreSQL: %s", clause)
	}
}

func TestAdminContactCountRawSQLIsPostgresCompatible(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	adminSvc := NewAdminService(db, t.TempDir())

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "pg-admin@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	dryDB := db.Session(&gorm.Session{DryRun: true})
	var contactCount int64
	stmt := dryDB.Raw(adminContactCountSQL(), resp.User.AccountID, false, false).Scan(&contactCount).Statement

	sql := strings.ToLower(stmt.SQL.String())
	if strings.Contains(sql, "can_be_deleted = 0") || strings.Contains(sql, "listed = 0") {
		t.Errorf("admin contact-count SQL compares boolean against integer literal; SQL: %s", sql)
	}

	if _, err := adminSvc.ListUsers(); err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
}
