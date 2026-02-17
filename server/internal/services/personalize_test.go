package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPersonalizeTest(t *testing.T) (*PersonalizeService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "personalize-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewPersonalizeService(db), resp.User.AccountID
}

func TestListEmotions(t *testing.T) {
	svc, accountID := setupPersonalizeTest(t)

	emotions, err := svc.List(accountID, "emotions")
	if err != nil {
		t.Fatalf("List emotions failed: %v", err)
	}

	if len(emotions) != 3 {
		t.Fatalf("Expected 3 seeded emotions, got %d", len(emotions))
	}

	typesSeen := map[string]bool{}
	for _, e := range emotions {
		if e.Label == "" {
			t.Errorf("Expected emotion label to be non-empty, got empty for id %d", e.ID)
		}
		typesSeen[e.Name] = true
	}

	for _, expected := range []string{"positive", "neutral", "negative"} {
		if !typesSeen[expected] {
			t.Errorf("Expected emotion type '%s' to be present", expected)
		}
	}
}

func TestListGenders(t *testing.T) {
	svc, accountID := setupPersonalizeTest(t)

	genders, err := svc.List(accountID, "genders")
	if err != nil {
		t.Fatalf("List genders failed: %v", err)
	}

	if len(genders) != 3 {
		t.Fatalf("Expected 3 seeded genders, got %d", len(genders))
	}
}

func TestListUnknownEntity(t *testing.T) {
	svc, accountID := setupPersonalizeTest(t)

	_, err := svc.List(accountID, "unknown-entity")
	if err != ErrUnknownEntityType {
		t.Errorf("Expected ErrUnknownEntityType, got %v", err)
	}
}
