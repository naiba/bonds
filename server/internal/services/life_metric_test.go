package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
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
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	_, _ = svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Metric 1"})
	_, _ = svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Metric 2"})

	metrics, err := svc.List(vaultID, userID)
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
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	created, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "ToDelete"})
	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	metrics, _ := svc.List(vaultID, userID)
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

func TestIncrementLifeMetric(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	metric, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Push-ups"})

	result, err := svc.Increment(metric.ID, vaultID, userID)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if result.Label != "Push-ups" {
		t.Errorf("Expected label 'Push-ups', got '%s'", result.Label)
	}
	if result.Stats.WeeklyEvents < 1 {
		t.Errorf("Expected WeeklyEvents >= 1, got %d", result.Stats.WeeklyEvents)
	}
	if result.Stats.MonthlyEvents < 1 {
		t.Errorf("Expected MonthlyEvents >= 1, got %d", result.Stats.MonthlyEvents)
	}
	if result.Stats.YearlyEvents < 1 {
		t.Errorf("Expected YearlyEvents >= 1, got %d", result.Stats.YearlyEvents)
	}
}

func TestIncrementLifeMetricStats(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	metric, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Meditation"})

	for i := 0; i < 5; i++ {
		_, err := svc.Increment(metric.ID, vaultID, userID)
		if err != nil {
			t.Fatalf("Increment %d failed: %v", i, err)
		}
	}

	metrics, err := svc.List(vaultID, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var found *dto.LifeMetricResponse
	for i := range metrics {
		if metrics[i].ID == metric.ID {
			found = &metrics[i]
			break
		}
	}
	if found == nil {
		t.Fatal("Metric not found in list")
	}
	if found.Stats.WeeklyEvents != 5 {
		t.Errorf("Expected WeeklyEvents == 5, got %d", found.Stats.WeeklyEvents)
	}
	if found.Stats.MonthlyEvents != 5 {
		t.Errorf("Expected MonthlyEvents == 5, got %d", found.Stats.MonthlyEvents)
	}
	if found.Stats.YearlyEvents != 5 {
		t.Errorf("Expected YearlyEvents == 5, got %d", found.Stats.YearlyEvents)
	}
}

func TestIncrementLifeMetric_NotFound(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	_, err := svc.Increment(9999, vaultID, userID)
	if err != ErrLifeMetricNotFound {
		t.Errorf("Expected ErrLifeMetricNotFound, got %v", err)
	}
}

func TestGetLifeMetricDetail(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	metric, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Running"})

	for i := 0; i < 3; i++ {
		_, err := svc.Increment(metric.ID, vaultID, userID)
		if err != nil {
			t.Fatalf("Increment %d failed: %v", i, err)
		}
	}

	year := time.Now().Year()
	detail, err := svc.GetDetail(metric.ID, vaultID, userID, year)
	if err != nil {
		t.Fatalf("GetDetail failed: %v", err)
	}
	if detail.Label != "Running" {
		t.Errorf("Expected label 'Running', got '%s'", detail.Label)
	}
	if len(detail.Months) != 12 {
		t.Fatalf("Expected 12 months, got %d", len(detail.Months))
	}

	currentMonth := int(time.Now().Month())
	totalEvents := 0
	for _, m := range detail.Months {
		totalEvents += m.Events
	}
	if totalEvents != 3 {
		t.Errorf("Expected 3 total events across months, got %d", totalEvents)
	}
	if detail.Months[currentMonth-1].Events != 3 {
		t.Errorf("Expected 3 events in current month, got %d", detail.Months[currentMonth-1].Events)
	}
	if detail.MaxEvents != 3 {
		t.Errorf("Expected MaxEvents == 3, got %d", detail.MaxEvents)
	}
	if detail.Months[0].FriendlyName != "January" {
		t.Errorf("Expected first month 'January', got '%s'", detail.Months[0].FriendlyName)
	}
}

func TestGetLifeMetricDetail_NotFound(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	_, err := svc.GetDetail(9999, vaultID, userID, 2026)
	if err != ErrLifeMetricNotFound {
		t.Errorf("Expected ErrLifeMetricNotFound, got %v", err)
	}
}

func TestGetLifeMetricDetail_EmptyYear(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	metric, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Swimming"})

	detail, err := svc.GetDetail(metric.ID, vaultID, userID, 2020)
	if err != nil {
		t.Fatalf("GetDetail failed: %v", err)
	}
	for _, m := range detail.Months {
		if m.Events != 0 {
			t.Errorf("Expected 0 events for month %d in empty year, got %d", m.Month, m.Events)
		}
	}
	if detail.MaxEvents != 0 {
		t.Errorf("Expected MaxEvents == 0, got %d", detail.MaxEvents)
	}
}

func TestListLifeMetrics_StatsScoping(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp1, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "User", LastName: "One",
		Email: "user1-lm@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp1.User.AccountID, resp1.User.ID, dto.CreateVaultRequest{Name: "Shared Vault"}, "en")

	resp2, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "User", LastName: "Two",
		Email: "user2-lm@example.com", Password: "password123",
	}, "en")

	svc := NewLifeMetricService(db)
	metric, _ := svc.Create(vault.ID, dto.CreateLifeMetricRequest{Label: "Exercise"})

	svc.Increment(metric.ID, vault.ID, resp1.User.ID)
	svc.Increment(metric.ID, vault.ID, resp1.User.ID)
	svc.Increment(metric.ID, vault.ID, resp2.User.ID)

	metricsUser1, _ := svc.List(vault.ID, resp1.User.ID)
	if metricsUser1[0].Stats.WeeklyEvents != 2 {
		t.Errorf("User1 should have 2 weekly events, got %d", metricsUser1[0].Stats.WeeklyEvents)
	}

	metricsUser2, _ := svc.List(vault.ID, resp2.User.ID)
	if metricsUser2[0].Stats.WeeklyEvents != 1 {
		t.Errorf("User2 should have 1 weekly event, got %d", metricsUser2[0].Stats.WeeklyEvents)
	}
}

// Ensure ContactLifeMetric UserID is stored correctly on increment
func TestIncrementLifeMetric_UserIDStored(t *testing.T) {
	svc, vaultID, _, userID := setupLifeMetricTest(t)

	metric, _ := svc.Create(vaultID, dto.CreateLifeMetricRequest{Label: "Steps"})
	svc.Increment(metric.ID, vaultID, userID)

	var events []models.ContactLifeMetric
	svc.db.Where("life_metric_id = ?", metric.ID).Find(&events)
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].UserID != userID {
		t.Errorf("Expected UserID '%s', got '%s'", userID, events[0].UserID)
	}
}
