package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupCallReasonTest(t *testing.T) (*CallReasonService, string, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "cr-test@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	var crt models.CallReasonType
	if err := db.Where("account_id = ?", resp.User.AccountID).First(&crt).Error; err != nil {
		t.Fatalf("Failed to find seeded CallReasonType: %v", err)
	}

	return NewCallReasonService(db), resp.User.AccountID, crt.ID
}

func TestCreateCallReason_Success(t *testing.T) {
	svc, accountID, callReasonTypeID := setupCallReasonTest(t)

	created, err := svc.Create(accountID, callReasonTypeID, dto.CreateCallReasonRequest{
		Label: "Catch up",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Label != "Catch up" {
		t.Errorf("Expected label 'Catch up', got '%s'", created.Label)
	}
	if created.CallReasonTypeID != callReasonTypeID {
		t.Errorf("Expected call_reason_type_id %d, got %d", callReasonTypeID, created.CallReasonTypeID)
	}
}

func TestCreateCallReason_TypeNotFound(t *testing.T) {
	svc, accountID, _ := setupCallReasonTest(t)

	_, err := svc.Create(accountID, 9999, dto.CreateCallReasonRequest{
		Label: "test",
	})
	if err != ErrCallReasonTypeNotFound {
		t.Errorf("Expected ErrCallReasonTypeNotFound, got %v", err)
	}
}

func TestUpdateCallReason_Success(t *testing.T) {
	svc, accountID, callReasonTypeID := setupCallReasonTest(t)

	created, err := svc.Create(accountID, callReasonTypeID, dto.CreateCallReasonRequest{
		Label: "Old reason",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(accountID, callReasonTypeID, created.ID, dto.UpdateCallReasonRequest{
		Label: "New reason",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "New reason" {
		t.Errorf("Expected label 'New reason', got '%s'", updated.Label)
	}
}

func TestUpdateCallReason_NotFound(t *testing.T) {
	svc, accountID, callReasonTypeID := setupCallReasonTest(t)

	_, err := svc.Update(accountID, callReasonTypeID, 9999, dto.UpdateCallReasonRequest{
		Label: "nope",
	})
	if err != ErrCallReasonNotFound {
		t.Errorf("Expected ErrCallReasonNotFound, got %v", err)
	}
}

func TestDeleteCallReason_Success(t *testing.T) {
	svc, accountID, callReasonTypeID := setupCallReasonTest(t)

	created, err := svc.Create(accountID, callReasonTypeID, dto.CreateCallReasonRequest{
		Label: "To delete",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(accountID, callReasonTypeID, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestDeleteCallReason_NotFound(t *testing.T) {
	svc, accountID, callReasonTypeID := setupCallReasonTest(t)

	err := svc.Delete(accountID, callReasonTypeID, 9999)
	if err != ErrCallReasonNotFound {
		t.Errorf("Expected ErrCallReasonNotFound, got %v", err)
	}
}
