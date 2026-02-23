package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type mostConsultedTestContext struct {
	svc       *MostConsultedService
	db        *gorm.DB
	vaultID   string
	userID    string
	contactID string
}

func setupMostConsultedTest(t *testing.T) *mostConsultedTestContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "most-consulted-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	nickname := "Johnny"
	prefix := "Dr."
	contact := &models.Contact{
		VaultID:   vault.ID,
		FirstName: strPtrOrNil("John"),
		LastName:  strPtrOrNil("Doe"),
		Nickname:  &nickname,
		Prefix:    &prefix,
	}
	if err := db.Create(contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	// Create ContactVaultUser with view count
	cvu := models.ContactVaultUser{
		ContactID:     contact.ID,
		VaultID:       vault.ID,
		UserID:        resp.User.ID,
		NumberOfViews: 5,
	}
	if err := db.Create(&cvu).Error; err != nil {
		t.Fatalf("Create ContactVaultUser failed: %v", err)
	}

	return &mostConsultedTestContext{
		svc:       NewMostConsultedService(db),
		db:        db,
		vaultID:   vault.ID,
		userID:    resp.User.ID,
		contactID: contact.ID,
	}
}

func TestMostConsulted_ListEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewMostConsultedService(db)
	result, err := svc.List("nonexistent-vault", "nonexistent-user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty list, got %d items", len(result))
	}
}

func TestMostConsulted_ReturnsNameFields(t *testing.T) {
	ctx := setupMostConsultedTest(t)
	result, err := ctx.svc.List(ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least 1 result")
	}

	item := result[0]
	if item.ContactID != ctx.contactID {
		t.Errorf("expected contact_id=%s, got %s", ctx.contactID, item.ContactID)
	}
	if item.FirstName != "John" {
		t.Errorf("expected first_name=John, got %s", item.FirstName)
	}
	if item.LastName != "Doe" {
		t.Errorf("expected last_name=Doe, got %s", item.LastName)
	}
	if item.Nickname != "Johnny" {
		t.Errorf("expected nickname=Johnny, got %s", item.Nickname)
	}
	if item.Prefix != "Dr." {
		t.Errorf("expected prefix=Dr., got %s", item.Prefix)
	}
	if item.NumberOfViews != 5 {
		t.Errorf("expected number_of_views=5, got %d", item.NumberOfViews)
	}
}

func TestMostConsulted_OrderByViews(t *testing.T) {
	ctx := setupMostConsultedTest(t)

	// Create a second contact with more views
	contact2 := &models.Contact{
		VaultID:   ctx.vaultID,
		FirstName: strPtrOrNil("Jane"),
		LastName:  strPtrOrNil("Smith"),
	}
	if err := ctx.db.Create(contact2).Error; err != nil {
		t.Fatalf("Create contact2 failed: %v", err)
	}
	cvu2 := models.ContactVaultUser{
		ContactID:     contact2.ID,
		VaultID:       ctx.vaultID,
		UserID:        ctx.userID,
		NumberOfViews: 10,
	}
	if err := ctx.db.Create(&cvu2).Error; err != nil {
		t.Fatalf("Create ContactVaultUser for contact2 failed: %v", err)
	}

	result, err := ctx.svc.List(ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(result) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(result))
	}
	// Jane (10 views) should come before John (5 views)
	if result[0].FirstName != "Jane" {
		t.Errorf("expected first result first_name=Jane, got %s", result[0].FirstName)
	}
	if result[1].FirstName != "John" {
		t.Errorf("expected second result first_name=John, got %s", result[1].FirstName)
	}
}