package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupJournalMetricTest(t *testing.T) (*JournalMetricService, *JournalService, uint, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "jm-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	journalSvc := NewJournalService(db)
	journal, err := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})
	if err != nil {
		t.Fatalf("CreateJournal failed: %v", err)
	}

	return NewJournalMetricService(db), journalSvc, journal.ID, vault.ID
}

func TestCreateJournalMetric(t *testing.T) {
	svc, _, journalID, vaultID := setupJournalMetricTest(t)

	metric, err := svc.Create(journalID, vaultID, dto.CreateJournalMetricRequest{Label: "Mood"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if metric.Label != "Mood" {
		t.Errorf("Expected label 'Mood', got '%s'", metric.Label)
	}
	if metric.JournalID != journalID {
		t.Errorf("Expected journal_id %d, got %d", journalID, metric.JournalID)
	}
}

func TestListJournalMetrics(t *testing.T) {
	svc, _, journalID, vaultID := setupJournalMetricTest(t)

	_, _ = svc.Create(journalID, vaultID, dto.CreateJournalMetricRequest{Label: "Mood"})
	_, _ = svc.Create(journalID, vaultID, dto.CreateJournalMetricRequest{Label: "Energy"})

	metrics, err := svc.List(journalID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
	}
}

func TestDeleteJournalMetric(t *testing.T) {
	svc, _, journalID, vaultID := setupJournalMetricTest(t)

	metric, _ := svc.Create(journalID, vaultID, dto.CreateJournalMetricRequest{Label: "Mood"})
	if err := svc.Delete(metric.ID, journalID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	metrics, _ := svc.List(journalID, vaultID)
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics after delete, got %d", len(metrics))
	}
}

func TestJournalMetricNotFound(t *testing.T) {
	svc, _, journalID, vaultID := setupJournalMetricTest(t)

	err := svc.Delete(9999, journalID, vaultID)
	if err != ErrJournalMetricNotFound {
		t.Errorf("Expected ErrJournalMetricNotFound, got %v", err)
	}
}

func TestPostMetricCreateDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "pm-test@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	journalSvc := NewJournalService(db)
	journal, _ := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})

	postSvc := NewPostService(db)
	post, _ := postSvc.Create(journal.ID, dto.CreatePostRequest{
		Title: "Test Post", WrittenAt: time.Now(),
	})

	jmSvc := NewJournalMetricService(db)
	jm, _ := jmSvc.Create(journal.ID, vault.ID, dto.CreateJournalMetricRequest{Label: "Mood"})

	pmSvc := NewPostMetricService(db)
	pm, err := pmSvc.Create(post.ID, journal.ID, dto.CreatePostMetricRequest{
		JournalMetricID: jm.ID, Value: 8,
	})
	if err != nil {
		t.Fatalf("Create post metric failed: %v", err)
	}
	if pm.Value != 8 {
		t.Errorf("Expected value 8, got %d", pm.Value)
	}

	err = pmSvc.Delete(pm.ID, post.ID, journal.ID)
	if err != nil {
		t.Fatalf("Delete post metric failed: %v", err)
	}
}

func TestPostMetricList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "pm-list@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	journalSvc := NewJournalService(db)
	journal, _ := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})

	postSvc := NewPostService(db)
	post, _ := postSvc.Create(journal.ID, dto.CreatePostRequest{
		Title: "Test Post", WrittenAt: time.Now(),
	})

	jmSvc := NewJournalMetricService(db)
	jm1, _ := jmSvc.Create(journal.ID, vault.ID, dto.CreateJournalMetricRequest{Label: "Mood"})
	jm2, _ := jmSvc.Create(journal.ID, vault.ID, dto.CreateJournalMetricRequest{Label: "Energy"})

	pmSvc := NewPostMetricService(db)
	pmSvc.Create(post.ID, journal.ID, dto.CreatePostMetricRequest{JournalMetricID: jm1.ID, Value: 8})
	pmSvc.Create(post.ID, journal.ID, dto.CreatePostMetricRequest{JournalMetricID: jm2.ID, Value: 5})

	metrics, err := pmSvc.List(post.ID, journal.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
	}
}

func TestPostMetricListNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	pmSvc := NewPostMetricService(db)

	_, err := pmSvc.List(99999, 1)
	if err != ErrPostNotFound {
		t.Errorf("Expected ErrPostNotFound, got %v", err)
	}
}
