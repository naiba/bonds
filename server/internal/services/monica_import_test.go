package services

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestParseMonicaExport_ValidFixture(t *testing.T) {
	data, err := os.ReadFile("../testdata/monica_export.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	export, err := ParseMonicaExport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export.Version != "1.0-preview.1" {
		t.Errorf("expected version 1.0-preview.1, got %s", export.Version)
	}
	// u9a8cu8bc1 3 u4e2au8054u7cfbu4eba
	contacts := getCollectionByType(export.Account.Data, "contacts")
	if len(contacts) != 3 {
		t.Errorf("expected 3 contacts, got %d", len(contacts))
	}
	// u9a8cu8bc1 2 u6761 relationship
	relationships := getCollectionByType(export.Account.Data, "relationships")
	if len(relationships) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(relationships))
	}
	// u9a8cu8bc1 instance.genders u975eu7a7a
	if len(export.Account.Instance.Genders) == 0 {
		t.Error("expected at least 1 gender in instance")
	}
	// u89e3u6790u7b2cu4e00u4e2au8054u7cfbu4ebauff08Johnuff09u5e76u9a8cu8bc1u5b50u8d44u6e90
	var john MonicaContact
	if err := json.Unmarshal(contacts[0], &john); err != nil {
		t.Fatalf("failed to unmarshal contact: %v", err)
	}
	if john.Properties.Birthdate == nil {
		t.Error("expected John to have birthdate")
	}
	notes := getCollectionByType(john.Data, "notes")
	if len(notes) == 0 {
		t.Error("expected John to have at least 1 note")
	}
}

func TestParseMonicaExport_InvalidJSON(t *testing.T) {
	_, err := ParseMonicaExport([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseMonicaExport_WrongVersion(t *testing.T) {
	jsonData := `{"version": "2.0", "account": {"uuid": "test"}}`
	_, err := ParseMonicaExport([]byte(jsonData))
	if err == nil {
		t.Error("expected error for wrong version")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error message, got: %v", err)
	}
}

func TestParseMonicaExport_EmptyContacts(t *testing.T) {
	jsonData := `{"version": "1.0-preview.1", "account": {"uuid": "test", "data": [], "instance": {}}}`
	export, err := ParseMonicaExport([]byte(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	contacts := getCollectionByType(export.Account.Data, "contacts")
	if len(contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(contacts))
	}
}

func setupMonicaImportTest(t *testing.T) (*MonicaImportService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "monica-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewMonicaImportService(db)
	return svc, vault.ID, resp.User.ID, resp.User.AccountID
}

func readMonicaFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../testdata/monica_export.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	return data
}

func TestMonicaImportContacts_Basic(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 3 {
		t.Errorf("expected 3 imported contacts, got %d", resp.ImportedContacts)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found in database: %v", err)
	}
	if john.DistantUUID == nil || *john.DistantUUID != "550e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("expected John DistantUUID=550e8400-e29b-41d4-a716-446655440001, got %v", john.DistantUUID)
	}
	if john.LastName == nil || *john.LastName != "Doe" {
		t.Errorf("expected John LastName=Doe, got %v", john.LastName)
	}
	if john.MiddleName == nil || *john.MiddleName != "Michael" {
		t.Errorf("expected John MiddleName=Michael, got %v", john.MiddleName)
	}
	if john.Nickname == nil || *john.Nickname != "Johnny" {
		t.Errorf("expected John Nickname=Johnny, got %v", john.Nickname)
	}
	if john.JobPosition == nil || *john.JobPosition != "Software Engineer" {
		t.Errorf("expected John JobPosition=Software Engineer, got %v", john.JobPosition)
	}
}

func TestMonicaImportContacts_Gender(t *testing.T) {
	svc, vaultID, userID, accountID := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	if john.GenderID == nil {
		t.Fatal("expected John.GenderID to be set")
	}

	var gender models.Gender
	if err := svc.DB.First(&gender, *john.GenderID).Error; err != nil {
		t.Fatalf("Gender not found: %v", err)
	}
	if gender.AccountID != accountID {
		t.Errorf("expected gender account_id=%s, got %s", accountID, gender.AccountID)
	}
	if gender.Name == nil || *gender.Name != "Male" {
		t.Errorf("expected gender name=Male, got %v", gender.Name)
	}
}

func TestMonicaImportContacts_Tags(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var labels []models.Label
	if err := svc.DB.Where("vault_id = ?", vaultID).Find(&labels).Error; err != nil {
		t.Fatalf("failed to query labels: %v", err)
	}
	if len(labels) != 2 {
		t.Errorf("expected 2 labels (friend, college), got %d", len(labels))
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var contactLabels []models.ContactLabel
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&contactLabels).Error; err != nil {
		t.Fatalf("failed to query contact_labels: %v", err)
	}
	if len(contactLabels) != 2 {
		t.Errorf("expected 2 contact-label pivots for John, got %d", len(contactLabels))
	}

	labelBySlug := make(map[string]bool)
	for _, l := range labels {
		if labelBySlug[l.Slug] {
			t.Errorf("duplicate label slug: %s", l.Slug)
		}
		labelBySlug[l.Slug] = true
	}
}

func TestMonicaImportContacts_Birthdate(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}

	var dates []models.ContactImportantDate
	if err := svc.DB.Preload("ContactImportantDateType").Where("contact_id = ?", john.ID).Find(&dates).Error; err != nil {
		t.Fatalf("failed to query important dates: %v", err)
	}

	var birthdate *models.ContactImportantDate
	for i := range dates {
		if dates[i].ContactImportantDateType != nil &&
			dates[i].ContactImportantDateType.InternalType != nil &&
			*dates[i].ContactImportantDateType.InternalType == "birthdate" {
			birthdate = &dates[i]
			break
		}
	}
	if birthdate == nil {
		t.Fatal("expected John to have a birthdate ImportantDate")
	}
	if birthdate.Year == nil || *birthdate.Year != 1990 {
		t.Errorf("expected birthdate year=1990, got %v", birthdate.Year)
	}
	if birthdate.Month == nil || *birthdate.Month != 5 {
		t.Errorf("expected birthdate month=5, got %v", birthdate.Month)
	}
	if birthdate.Day == nil || *birthdate.Day != 15 {
		t.Errorf("expected birthdate day=15, got %v", birthdate.Day)
	}
}

func TestMonicaImportContacts_DeceasedDate(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var jane models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "Jane").First(&jane).Error; err != nil {
		t.Fatalf("Jane not found: %v", err)
	}

	var dates []models.ContactImportantDate
	if err := svc.DB.Preload("ContactImportantDateType").Where("contact_id = ?", jane.ID).Find(&dates).Error; err != nil {
		t.Fatalf("failed to query important dates: %v", err)
	}

	var deceased *models.ContactImportantDate
	for i := range dates {
		if dates[i].ContactImportantDateType != nil &&
			dates[i].ContactImportantDateType.InternalType != nil &&
			*dates[i].ContactImportantDateType.InternalType == "deceased_date" {
			deceased = &dates[i]
			break
		}
	}
	if deceased == nil {
		t.Fatal("expected Jane to have a deceased_date ImportantDate")
	}
	if deceased.Year == nil || *deceased.Year != 2024 {
		t.Errorf("expected deceased year=2024, got %v", deceased.Year)
	}
	if deceased.Month == nil || *deceased.Month != 1 {
		t.Errorf("expected deceased month=1, got %v", deceased.Month)
	}
	if deceased.Day == nil || *deceased.Day != 10 {
		t.Errorf("expected deceased day=10, got %v", deceased.Day)
	}
}

func TestMonicaImportContacts_IsPartial(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var bob models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "Bob").First(&bob).Error; err != nil {
		t.Fatalf("Bob not found: %v", err)
	}
	if bob.Listed {
		t.Error("expected Bob.Listed=false (is_partial=true in Monica)")
	}
}

func TestMonicaImportContacts_IsFavorite(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}

	var cvu models.ContactVaultUser
	if err := svc.DB.Where("contact_id = ? AND vault_id = ?", john.ID, vaultID).First(&cvu).Error; err != nil {
		t.Fatalf("ContactVaultUser not found: %v", err)
	}
	if !cvu.IsFavorite {
		t.Error("expected John's ContactVaultUser.IsFavorite=true")
	}
}

func TestMonicaImportContacts_Duplicate(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp1, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("first Import failed: %v", err)
	}
	if resp1.ImportedContacts != 3 {
		t.Errorf("first import: expected 3 imported, got %d", resp1.ImportedContacts)
	}

	resp2, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("second Import failed: %v", err)
	}
	if resp2.ImportedContacts != 0 {
		t.Errorf("second import: expected 0 imported (all duplicates), got %d", resp2.ImportedContacts)
	}
	if resp2.SkippedCount < 3 {
		t.Errorf("second import: expected >=3 skipped, got %d", resp2.SkippedCount)
	}

	var contacts []models.Contact
	if err := svc.DB.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		t.Fatalf("failed to query contacts: %v", err)
	}
	// Vault seed creates 1 shadow contact (UserVault.ContactID) + 3 imported = 4 total
	expectedCount := 4
	if len(contacts) != expectedCount {
		t.Errorf("expected %d contacts in vault (1 shadow + 3 imported), got %d", expectedCount, len(contacts))
	}
}
