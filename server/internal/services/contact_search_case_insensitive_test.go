package services

import (
	"os"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

// SQLite LIKE is case-insensitive by default; PostgreSQL LIKE is
// case-sensitive. Contact name search must behave identically on both, so the
// source must use LOWER(col) LIKE LOWER(?) (or ILIKE) rather than a bare LIKE.
// A functional SQLite test would pass without the fix, so we inspect the source
// directly.

func TestContactSearchUsesCaseInsensitiveLike(t *testing.T) {
	src, err := os.ReadFile("contact.go")
	if err != nil {
		t.Fatalf("read contact.go: %v", err)
	}
	content := string(src)

	for _, bare := range []string{
		`first_name LIKE ?`,
		`last_name LIKE ?`,
		`nickname LIKE ?`,
		`maiden_name LIKE ?`,
		`middle_name LIKE ?`,
	} {
		if strings.Contains(content, bare) {
			t.Errorf("contact.go uses bare %q - breaks case-insensitive search on PostgreSQL; use LOWER(col) LIKE LOWER(?) instead", bare)
		}
	}
}

func TestContactSearchIsCaseInsensitive(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	contactSvc := NewContactService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "T", LastName: "U",
		Email: "ci-fn@example.com", Password: "password123",
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

	for _, term := range []string{"alice", "ALICE", "AlIcE"} {
		got, _, err := contactSvc.ListContacts(vault.ID, resp.User.ID, 1, 15, term, "", "")
		if err != nil {
			t.Fatalf("ListContacts(%q) failed: %v", term, err)
		}
		if len(got) != 1 || got[0].FirstName != "Alice" {
			t.Errorf("ListContacts(%q) = %v; want 1 match 'Alice'", term, got)
		}

		qs, err := contactSvc.QuickSearch(vault.ID, term)
		if err != nil {
			t.Fatalf("QuickSearch(%q) failed: %v", term, err)
		}
		if len(qs) != 1 {
			t.Errorf("QuickSearch(%q) = %v; want 1 match", term, qs)
		}
	}
}
