package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type giftTestContext struct {
	db          *gorm.DB
	svc         *GiftService
	contactID   string
	vaultID     string
	userID      string
	accountID   string
	occasionIDs []uint
	stateIDs    []uint
}

func setupGiftTest(t *testing.T) giftTestContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Gift",
		LastName:  "Tester",
		Email:     "gift-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Gift Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contact, err := NewContactService(db).CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return giftTestContext{
		db:          db,
		svc:         NewGiftService(db),
		contactID:   contact.ID,
		vaultID:     vault.ID,
		userID:      resp.User.ID,
		accountID:   resp.User.AccountID,
		occasionIDs: loadGiftOccasionIDs(t, db, resp.User.AccountID),
		stateIDs:    loadGiftStateIDs(t, db, resp.User.AccountID),
	}
}

func loadGiftOccasionIDs(t *testing.T, db *gorm.DB, accountID string) []uint {
	t.Helper()
	var occasions []models.GiftOccasion
	if err := db.Where("account_id = ?", accountID).Order("position ASC, id ASC").Find(&occasions).Error; err != nil {
		t.Fatalf("load gift occasions failed: %v", err)
	}
	if len(occasions) < 2 {
		t.Fatalf("expected at least 2 seeded gift occasions, got %d", len(occasions))
	}
	ids := make([]uint, len(occasions))
	for i, occasion := range occasions {
		ids[i] = occasion.ID
	}
	return ids
}

func loadGiftStateIDs(t *testing.T, db *gorm.DB, accountID string) []uint {
	t.Helper()
	var states []models.GiftState
	if err := db.Where("account_id = ?", accountID).Order("position ASC, id ASC").Find(&states).Error; err != nil {
		t.Fatalf("load gift states failed: %v", err)
	}
	if len(states) < 2 {
		t.Fatalf("expected at least 2 seeded gift states, got %d", len(states))
	}
	ids := make([]uint, len(states))
	for i, state := range states {
		ids[i] = state.ID
	}
	return ids
}

func validGiftCreateRequest(ctx giftTestContext) dto.CreateGiftRequest {
	price := 2500
	statusDate := time.Date(2026, time.January, 15, 10, 30, 0, 0, time.UTC)
	return dto.CreateGiftRequest{
		Name:           "Birthday book",
		Type:           "given",
		Description:    "Signed first edition",
		EstimatedPrice: &price,
		GiftOccasionID: ctx.occasionIDs[0],
		GiftStateID:    ctx.stateIDs[0],
		StatusDate:     &statusDate,
		GivenAt:        &statusDate,
	}
}

func validGiftUpdateRequest(ctx giftTestContext) dto.UpdateGiftRequest {
	price := 3500
	statusDate := time.Date(2026, time.February, 16, 11, 0, 0, 0, time.UTC)
	return dto.UpdateGiftRequest{
		Name:           "Anniversary record",
		Type:           "received",
		Description:    "Limited edition vinyl",
		EstimatedPrice: &price,
		GiftOccasionID: ctx.occasionIDs[1],
		GiftStateID:    ctx.stateIDs[1],
		StatusDate:     &statusDate,
		ReceivedAt:     &statusDate,
	}
}

func TestGiftServiceCreateListUpdateDelete(t *testing.T) {
	ctx := setupGiftTest(t)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, validGiftCreateRequest(ctx))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected created gift ID")
	}
	if created.ContactID != ctx.contactID {
		t.Fatalf("expected contact_id %q, got %q", ctx.contactID, created.ContactID)
	}
	if created.Name != "Birthday book" || created.Type != "given" {
		t.Fatalf("unexpected created gift response: %#v", created)
	}
	if created.GiftOccasionID == nil || *created.GiftOccasionID != ctx.occasionIDs[0] {
		t.Fatalf("expected gift_occasion_id %d, got %v", ctx.occasionIDs[0], created.GiftOccasionID)
	}
	if created.GiftStateID == nil || *created.GiftStateID != ctx.stateIDs[0] {
		t.Fatalf("expected gift_state_id %d, got %v", ctx.stateIDs[0], created.GiftStateID)
	}
	if created.GiftOccasionLabel == "" || created.GiftStateLabel == "" {
		t.Fatalf("expected preloaded occasion/state labels, got occasion=%q state=%q", created.GiftOccasionLabel, created.GiftStateLabel)
	}

	listed, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("expected one listed gift %d, got %#v", created.ID, listed)
	}

	updated, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, validGiftUpdateRequest(ctx))
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Anniversary record" || updated.Type != "received" {
		t.Fatalf("unexpected updated gift response: %#v", updated)
	}
	if updated.GiftOccasionID == nil || *updated.GiftOccasionID != ctx.occasionIDs[1] {
		t.Fatalf("expected updated gift_occasion_id %d, got %v", ctx.occasionIDs[1], updated.GiftOccasionID)
	}
	if updated.GiftStateID == nil || *updated.GiftStateID != ctx.stateIDs[1] {
		t.Fatalf("expected updated gift_state_id %d, got %v", ctx.stateIDs[1], updated.GiftStateID)
	}

	if err := ctx.svc.Delete(updated.ID, ctx.contactID, ctx.vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	listed, err = ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List after delete failed: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected no gifts after delete, got %d", len(listed))
	}
}

func TestGiftServiceRequiresNameOccasionAndState(t *testing.T) {
	ctx := setupGiftTest(t)
	req := validGiftCreateRequest(ctx)

	req.Name = "   "
	if _, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, req); !errors.Is(err, ErrGiftNameRequired) {
		t.Fatalf("expected ErrGiftNameRequired, got %v", err)
	}

	req = validGiftCreateRequest(ctx)
	req.GiftOccasionID = 0
	if _, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, req); !errors.Is(err, ErrGiftOccasionNotFound) {
		t.Fatalf("expected ErrGiftOccasionNotFound, got %v", err)
	}

	req = validGiftCreateRequest(ctx)
	req.GiftStateID = 0
	if _, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, req); !errors.Is(err, ErrGiftStateNotFound) {
		t.Fatalf("expected ErrGiftStateNotFound, got %v", err)
	}
}

func TestGiftServiceWrongVaultAndNotFound(t *testing.T) {
	ctx := setupGiftTest(t)
	otherVault, err := NewVaultService(ctx.db).CreateVault(ctx.accountID, ctx.userID, dto.CreateVaultRequest{Name: "Other Vault"}, "en")
	if err != nil {
		t.Fatalf("Create other vault failed: %v", err)
	}

	if _, err := ctx.svc.List(ctx.contactID, otherVault.ID); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for wrong-vault list, got %v", err)
	}
	if _, err := ctx.svc.Create(ctx.contactID, otherVault.ID, validGiftCreateRequest(ctx)); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for wrong-vault create, got %v", err)
	}

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, validGiftCreateRequest(ctx))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := ctx.svc.Update(created.ID, ctx.contactID, otherVault.ID, validGiftUpdateRequest(ctx)); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for wrong-vault update, got %v", err)
	}
	if err := ctx.svc.Delete(created.ID, ctx.contactID, otherVault.ID); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound for wrong-vault delete, got %v", err)
	}

	if _, err := ctx.svc.Update(9999, ctx.contactID, ctx.vaultID, validGiftUpdateRequest(ctx)); !errors.Is(err, ErrGiftNotFound) {
		t.Fatalf("expected ErrGiftNotFound for missing update, got %v", err)
	}
	if err := ctx.svc.Delete(9999, ctx.contactID, ctx.vaultID); !errors.Is(err, ErrGiftNotFound) {
		t.Fatalf("expected ErrGiftNotFound for missing delete, got %v", err)
	}
}

func TestGiftServiceRejectsCrossAccountOccasionAndState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(db)

	first, err := authSvc.Register(dto.RegisterRequest{FirstName: "First", LastName: "User", Email: "gift-first@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatalf("Register first account failed: %v", err)
	}
	second, err := authSvc.Register(dto.RegisterRequest{FirstName: "Second", LastName: "User", Email: "gift-second@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatalf("Register second account failed: %v", err)
	}
	if first.User.AccountID == second.User.AccountID {
		t.Fatal("expected distinct account IDs")
	}

	vault, err := vaultSvc.CreateVault(first.User.AccountID, first.User.ID, dto.CreateVaultRequest{Name: "First Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	contact, err := NewContactService(db).CreateContact(vault.ID, first.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	ctx := giftTestContext{
		db:          db,
		svc:         NewGiftService(db),
		contactID:   contact.ID,
		vaultID:     vault.ID,
		userID:      first.User.ID,
		accountID:   first.User.AccountID,
		occasionIDs: loadGiftOccasionIDs(t, db, first.User.AccountID),
		stateIDs:    loadGiftStateIDs(t, db, first.User.AccountID),
	}
	otherOccasionIDs := loadGiftOccasionIDs(t, db, second.User.AccountID)
	otherStateIDs := loadGiftStateIDs(t, db, second.User.AccountID)

	req := validGiftCreateRequest(ctx)
	req.GiftOccasionID = otherOccasionIDs[0]
	if _, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, req); !errors.Is(err, ErrGiftOccasionNotFound) {
		t.Fatalf("expected ErrGiftOccasionNotFound for cross-account create, got %v", err)
	}

	req = validGiftCreateRequest(ctx)
	req.GiftStateID = otherStateIDs[0]
	if _, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, req); !errors.Is(err, ErrGiftStateNotFound) {
		t.Fatalf("expected ErrGiftStateNotFound for cross-account create, got %v", err)
	}

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, validGiftCreateRequest(ctx))
	if err != nil {
		t.Fatalf("Create own-account gift failed: %v", err)
	}
	updateReq := validGiftUpdateRequest(ctx)
	updateReq.GiftOccasionID = otherOccasionIDs[0]
	if _, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, updateReq); !errors.Is(err, ErrGiftOccasionNotFound) {
		t.Fatalf("expected ErrGiftOccasionNotFound for cross-account update, got %v", err)
	}

	updateReq = validGiftUpdateRequest(ctx)
	updateReq.GiftStateID = otherStateIDs[0]
	if _, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, updateReq); !errors.Is(err, ErrGiftStateNotFound) {
		t.Fatalf("expected ErrGiftStateNotFound for cross-account update, got %v", err)
	}

	if _, err := ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, validGiftUpdateRequest(ctx)); err != nil {
		t.Fatalf("expected own-account update to still work, got %v", err)
	}
}
