package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
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
	}, "en")
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

func TestSyncAllTranslations_AccountEntities(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "sync-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	accountID := resp.User.AccountID
	svc := NewPersonalizeService(db)

	gendersBefore, err := svc.List(accountID, "genders")
	if err != nil {
		t.Fatalf("List genders failed: %v", err)
	}
	foundMale := false
	for _, g := range gendersBefore {
		if g.Name == "Male" {
			foundMale = true
			break
		}
	}
	if !foundMale {
		t.Fatal("expected English 'Male' gender before sync")
	}

	if err := svc.SyncAllTranslations(accountID, "zh"); err != nil {
		t.Fatalf("SyncAllTranslations failed: %v", err)
	}

	gendersAfter, err := svc.List(accountID, "genders")
	if err != nil {
		t.Fatalf("List genders after sync failed: %v", err)
	}
	foundChinese := false
	for _, g := range gendersAfter {
		if g.Name == "男" {
			foundChinese = true
			break
		}
	}
	if !foundChinese {
		t.Fatal("expected Chinese '男' gender after sync to zh")
	}

	religions, err := svc.List(accountID, "religions")
	if err != nil {
		t.Fatalf("List religions failed: %v", err)
	}
	foundChristian := false
	for _, r := range religions {
		if r.Name == "基督教" {
			foundChristian = true
			break
		}
	}
	if !foundChristian {
		names := make([]string, len(religions))
		for i, r := range religions {
			names[i] = r.Name
		}
		t.Fatalf("expected Chinese '基督教' religion after sync, got: %v", names)
	}
}

func TestSyncAllTranslations_VaultEntities(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "sync-vault-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	accountID := resp.User.AccountID
	userID := resp.User.ID

	vaultSvc := NewVaultService(db)
	vault, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{
		Name: "Sync Test Vault",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	var moodsBefore []struct {
		Label string `gorm:"column:label"`
	}
	if err := db.Table("mood_tracking_parameters").
		Where("vault_id = ?", vault.ID).
		Select("label").Find(&moodsBefore).Error; err != nil {
		t.Fatalf("query moods failed: %v", err)
	}
	foundAwesome := false
	for _, m := range moodsBefore {
		if m.Label == "🥳 Awesome" {
			foundAwesome = true
			break
		}
	}
	if !foundAwesome {
		t.Fatal("expected English '🥳 Awesome' mood before sync")
	}

	svc := NewPersonalizeService(db)
	if err := svc.SyncAllTranslations(accountID, "zh"); err != nil {
		t.Fatalf("SyncAllTranslations failed: %v", err)
	}

	var moodsAfter []struct {
		Label string `gorm:"column:label"`
	}
	if err := db.Table("mood_tracking_parameters").
		Where("vault_id = ?", vault.ID).
		Select("label").Find(&moodsAfter).Error; err != nil {
		t.Fatalf("query moods after sync failed: %v", err)
	}
	foundChinese := false
	for _, m := range moodsAfter {
		if m.Label == "🥳 棒极了" {
			foundChinese = true
			break
		}
	}
	if !foundChinese {
		labels := make([]string, len(moodsAfter))
		for i, m := range moodsAfter {
			labels[i] = m.Label
		}
		t.Fatalf("expected Chinese '🥳 棒极了' mood after sync, got: %v", labels)
	}
}

func TestSyncAllTranslations_CustomLabelsUnchanged(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "sync-custom-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	accountID := resp.User.AccountID
	svc := NewPersonalizeService(db)

	customLabel := "My Custom Gender"
	customGender := models.Gender{
		AccountID: accountID,
		Name:      &customLabel,
	}
	if err := db.Create(&customGender).Error; err != nil {
		t.Fatalf("Create custom gender failed: %v", err)
	}

	if err := svc.SyncAllTranslations(accountID, "zh"); err != nil {
		t.Fatalf("SyncAllTranslations failed: %v", err)
	}

	genders, err := svc.List(accountID, "genders")
	if err != nil {
		t.Fatalf("List genders failed: %v", err)
	}
	foundCustom := false
	for _, g := range genders {
		if g.Name == customLabel {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		names := make([]string, len(genders))
		for i, g := range genders {
			names[i] = g.Name
		}
		t.Fatalf("custom label %q should be unchanged after sync, got: %v", customLabel, names)
	}
}
