package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupLifeMetricTest(t *testing.T) (*LifeMetricService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "lm-test@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	contactSvc := NewContactService(db)
	contact, _ := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})

	return NewLifeMetricService(db), vault.ID, contact.ID, resp.User.ID
}

func TestCreateLifeMetric(t *testing.T) {
	svc, vaultID, _, _ := setupLifeMetricTest(t)

	metric, err := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Happiness"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if metric.Label != "Happiness" {
		t.Errorf("Expected label 'Happiness', got '%s'", metric.Label)
	}
}

func TestListLifeMetrics(t *testing.T) {
	svc, vaultID, _, _ := setupLifeMetricTest(t)

	_, _ = svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Metric 1"})
	_, _ = svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Metric 2"})

	metrics, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
	}
}

func TestUpdateLifeMetric(t *testing.T) {
	svc, vaultID, _, _ := setupLifeMetricTest(t)

	created, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Old"})
	updated, err := svc.Update(created.ID, vaultID, dto.UpdateLifeMetricRequest{Label: "New"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "New" {
		t.Errorf("Expected label 'New', got '%s'", updated.Label)
	}
}

func TestDeleteLifeMetric(t *testing.T) {
	svc, vaultID, _, _ := setupLifeMetricTest(t)

	created, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "ToDelete"})
	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	metrics, _ := svc.List(vaultID)
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(metrics))
	}
}

func TestLifeMetricNotFound(t *testing.T) {
	svc, vaultID, _, _ := setupLifeMetricTest(t)

	_, err := svc.Update(9999, vaultID, dto.UpdateLifeMetricRequest{Label: "nope"})
	if err != ErrLifeMetricNotFound {
		t.Errorf("Expected ErrLifeMetricNotFound, got %v", err)
	}
}

func TestAddLifeMetricContact(t *testing.T) {
	svc, vaultID, contactID, _ := setupLifeMetricTest(t)

	metric, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Wellness"})
	if err := svc.AddContact(metric.ID, vaultID, contactID); err != nil {
		t.Fatalf("AddContact failed: %v", err)
	}
}
