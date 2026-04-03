package services

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
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

	svc := NewMonicaImportService(db, "")
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

func TestMonicaImportNotes(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	// John has 1 original note + 1 activity-as-note + 1 conversation-as-note = 3
	if resp.ImportedNotes != 3 {
		t.Errorf("expected 3 imported notes (1 note + 1 activity + 1 conversation), got %d", resp.ImportedNotes)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var notes []models.Note
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&notes).Error; err != nil {
		t.Fatalf("failed to query notes: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}
	foundOriginal := false
	for _, n := range notes {
		if n.Body == "Met at the reunion last week. He looks great!" {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Error("expected to find original note body")
	}
}

func TestMonicaImportCalls(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedCalls != 1 {
		t.Errorf("expected 1 imported call, got %d", resp.ImportedCalls)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var calls []models.Call
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&calls).Error; err != nil {
		t.Fatalf("failed to query calls: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].WhoInitiated != "user" {
		t.Errorf("expected who_initiated=user (contact_called=false), got %s", calls[0].WhoInitiated)
	}
	if calls[0].Description == nil || *calls[0].Description != "Discussed weekend plans" {
		t.Errorf("unexpected call description: %v", calls[0].Description)
	}
}

func TestMonicaImportTasks(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedTasks != 1 {
		t.Errorf("expected 1 imported task, got %d", resp.ImportedTasks)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var tasks []models.ContactTask
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&tasks).Error; err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Label != "Buy birthday gift" {
		t.Errorf("expected task label=Buy birthday gift, got %s", tasks[0].Label)
	}
	if tasks[0].Completed {
		t.Error("expected task to not be completed")
	}
}

func TestMonicaImportReminders(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedReminders != 1 {
		t.Errorf("expected 1 imported reminder, got %d", resp.ImportedReminders)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var reminders []models.ContactReminder
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&reminders).Error; err != nil {
		t.Fatalf("failed to query reminders: %v", err)
	}
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].Type != "recurring_year" {
		t.Errorf("expected reminder type=recurring_year, got %s", reminders[0].Type)
	}
	if reminders[0].Label != "John's birthday" {
		t.Errorf("expected reminder label=John's birthday, got %s", reminders[0].Label)
	}
	if reminders[0].Day == nil || *reminders[0].Day != 15 {
		t.Errorf("expected reminder day=15, got %v", reminders[0].Day)
	}
	if reminders[0].Month == nil || *reminders[0].Month != 5 {
		t.Errorf("expected reminder month=5, got %v", reminders[0].Month)
	}
}

func TestMonicaImportAddresses(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedAddresses != 1 {
		t.Errorf("expected 1 imported address, got %d", resp.ImportedAddresses)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var cas []models.ContactAddress
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&cas).Error; err != nil {
		t.Fatalf("failed to query contact_address: %v", err)
	}
	if len(cas) != 1 {
		t.Fatalf("expected 1 contact_address pivot, got %d", len(cas))
	}
	var addr models.Address
	if err := svc.DB.First(&addr, cas[0].AddressID).Error; err != nil {
		t.Fatalf("address not found: %v", err)
	}
	if addr.City == nil || *addr.City != "Springfield" {
		t.Errorf("expected city=Springfield, got %v", addr.City)
	}
}

func TestMonicaImportContactInfo(t *testing.T) {
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
	var cis []models.ContactInformation
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&cis).Error; err != nil {
		t.Fatalf("failed to query contact_information: %v", err)
	}
	found := false
	for _, ci := range cis {
		if ci.Data == "john.doe@example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected contact information with data=john.doe@example.com")
	}
}

func TestMonicaImportPets(t *testing.T) {
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
	var pets []models.Pet
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&pets).Error; err != nil {
		t.Fatalf("failed to query pets: %v", err)
	}
	if len(pets) != 1 {
		t.Fatalf("expected 1 pet, got %d", len(pets))
	}
	if pets[0].Name == nil || *pets[0].Name != "Buddy" {
		t.Errorf("expected pet name=Buddy, got %v", pets[0].Name)
	}
}

func TestMonicaImportGifts(t *testing.T) {
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
	var gifts []models.Gift
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&gifts).Error; err != nil {
		t.Fatalf("failed to query gifts: %v", err)
	}
	if len(gifts) != 1 {
		t.Fatalf("expected 1 gift, got %d", len(gifts))
	}
	if gifts[0].Type != "given" {
		t.Errorf("expected gift type=given (status=offered), got %s", gifts[0].Type)
	}
	if gifts[0].Name != "Book: Clean Code" {
		t.Errorf("expected gift name=Book: Clean Code, got %s", gifts[0].Name)
	}
	if gifts[0].EstimatedPrice == nil || *gifts[0].EstimatedPrice != 3599 {
		t.Errorf("expected gift estimated_price=3599 (35.99*100), got %v", gifts[0].EstimatedPrice)
	}
}

func TestMonicaImportLoans(t *testing.T) {
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
	var loans []models.Loan
	if err := svc.DB.Where("vault_id = ?", vaultID).Find(&loans).Error; err != nil {
		t.Fatalf("failed to query loans: %v", err)
	}
	if len(loans) != 1 {
		t.Fatalf("expected 1 loan, got %d", len(loans))
	}
	if loans[0].Type != "borrowed_from" {
		t.Errorf("expected loan type=borrowed_from (in_debt=true), got %s", loans[0].Type)
	}
	if loans[0].AmountLent == nil || *loans[0].AmountLent != 5000 {
		t.Errorf("expected loan amount=5000 (50.00*100), got %v", loans[0].AmountLent)
	}
	var cls []models.ContactLoan
	if err := svc.DB.Where("loan_id = ?", loans[0].ID).Find(&cls).Error; err != nil {
		t.Fatalf("failed to query contact_loan: %v", err)
	}
	if len(cls) != 1 {
		t.Fatalf("expected 1 contact_loan pivot, got %d", len(cls))
	}
	if cls[0].LoanerID == cls[0].LoaneeID {
		t.Error("loaner and loanee should be different contacts")
	}
}

func TestMonicaImportLifeEvents(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedLifeEvents != 1 {
		t.Errorf("expected 1 imported life event, got %d", resp.ImportedLifeEvents)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var participants []models.LifeEventParticipant
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&participants).Error; err != nil {
		t.Fatalf("failed to query life_event_participants: %v", err)
	}
	if len(participants) != 1 {
		t.Fatalf("expected 1 life_event_participant, got %d", len(participants))
	}
	var le models.LifeEvent
	if err := svc.DB.First(&le, participants[0].LifeEventID).Error; err != nil {
		t.Fatalf("life event not found: %v", err)
	}
	if le.Summary == nil || *le.Summary != "Got promoted" {
		t.Errorf("expected life event summary=Got promoted, got %v", le.Summary)
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

func TestMonicaImportRelationships_Basic(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedRelationships != 2 {
		t.Errorf("expected 2 imported relationships, got %d", resp.ImportedRelationships)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	var jane models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "Jane").First(&jane).Error; err != nil {
		t.Fatalf("Jane not found: %v", err)
	}

	var rels []models.Relationship
	if err := svc.DB.Where("contact_id = ? AND related_contact_id = ?", john.ID, jane.ID).Find(&rels).Error; err != nil {
		t.Fatalf("failed to query relationships: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship John->Jane, got %d", len(rels))
	}

	var relType models.RelationshipType
	if err := svc.DB.First(&relType, rels[0].RelationshipTypeID).Error; err != nil {
		t.Fatalf("relationship type not found: %v", err)
	}
	if relType.Name == nil || strings.ToLower(*relType.Name) != "spouse" {
		t.Errorf("expected relationship type name=spouse, got %v", relType.Name)
	}
}

func TestMonicaImportRelationships_UnresolvedContact(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)

	exportJSON := `{
		"version": "1.0-preview.1",
		"account": {
			"uuid": "test-account",
			"data": [
				{"count": 1, "type": "contacts", "values": [
					{"uuid": "c1", "properties": {"first_name": "Alice"}, "data": []}
				]},
				{"count": 1, "type": "relationships", "values": [
					{"uuid": "r1", "properties": {"type": "friend", "contact_is": "c1", "of_contact": "nonexistent-uuid"}}
				]}
			],
			"properties": {},
			"instance": {}
		}
	}`

	resp, err := svc.Import(vaultID, userID, []byte(exportJSON))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedRelationships != 0 {
		t.Errorf("expected 0 imported relationships, got %d", resp.ImportedRelationships)
	}

	foundUnresolved := false
	for _, e := range resp.Errors {
		if strings.Contains(e, "unresolved contacts") {
			foundUnresolved = true
			break
		}
	}
	if !foundUnresolved {
		t.Error("expected error about unresolved contacts")
	}
}

func TestMonicaImportRelationships_UnknownType(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)

	exportJSON := `{
		"version": "1.0-preview.1",
		"account": {
			"uuid": "test-account",
			"data": [
				{"count": 2, "type": "contacts", "values": [
					{"uuid": "c1", "properties": {"first_name": "Alice"}, "data": []},
					{"uuid": "c2", "properties": {"first_name": "Bob"}, "data": []}
				]},
				{"count": 1, "type": "relationships", "values": [
					{"uuid": "r1", "properties": {"type": "soulmate", "contact_is": "c1", "of_contact": "c2"}}
				]}
			],
			"properties": {},
			"instance": {}
		}
	}`

	resp, err := svc.Import(vaultID, userID, []byte(exportJSON))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedRelationships != 0 {
		t.Errorf("expected 0 imported relationships, got %d", resp.ImportedRelationships)
	}

	foundTypeError := false
	for _, e := range resp.Errors {
		if strings.Contains(e, "relationship type not found: soulmate") {
			foundTypeError = true
			break
		}
	}
	if !foundTypeError {
		t.Errorf("expected error about unknown relationship type, errors: %v", resp.Errors)
	}
}

func TestMonicaImportActivities(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}

	var notes []models.Note
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&notes).Error; err != nil {
		t.Fatalf("failed to query notes: %v", err)
	}

	foundActivity := false
	for _, n := range notes {
		if strings.Contains(n.Body, "[Activity: Ate together]") && strings.Contains(n.Body, "Dinner at Italian restaurant") {
			if !strings.Contains(n.Body, "Had a great time catching up over pasta") {
				t.Error("expected activity note to contain description")
			}
			foundActivity = true
			break
		}
	}
	if !foundActivity {
		t.Errorf("expected to find activity degraded as note, notes: %d, resp.ImportedNotes: %d", len(notes), resp.ImportedNotes)
	}
}

func TestMonicaImportConversations(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	_ = resp

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}

	var notes []models.Note
	if err := svc.DB.Where("contact_id = ?", john.ID).Find(&notes).Error; err != nil {
		t.Fatalf("failed to query notes: %v", err)
	}

	foundConversation := false
	for _, n := range notes {
		if n.Title != nil && strings.Contains(*n.Title, "Conversation") {
			if !strings.Contains(n.Body, "Me: Hey, how are you?") {
				t.Errorf("expected conversation note body to contain message, got: %s", n.Body)
			}
			foundConversation = true
			break
		}
	}
	if !foundConversation {
		t.Error("expected to find conversation degraded as note with title containing 'Conversation'")
	}
}

func setupMonicaImportWithStorageTest(t *testing.T) (*MonicaImportService, string, string, string) {
	t.Helper()
	svc, vaultID, userID, accountID := setupMonicaImportTest(t)
	svc.UploadDir = t.TempDir()
	return svc, vaultID, userID, accountID
}

func TestMonicaImportPhotos(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportWithStorageTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedPhotos != 1 {
		t.Errorf("expected 1 imported photo, got %d", resp.ImportedPhotos)
	}

	var files []models.File
	if err := svc.DB.Where("vault_id = ? AND type = ?", vaultID, "photo").Find(&files).Error; err != nil {
		t.Fatalf("failed to query files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 photo file record, got %d", len(files))
	}
	if files[0].MimeType != "image/png" {
		t.Errorf("expected mime_type=image/png, got %s", files[0].MimeType)
	}
	if files[0].Name != "john-profile.png" {
		t.Errorf("expected name=john-profile.png, got %s", files[0].Name)
	}
	if files[0].OriginalURL == nil {
		t.Fatal("expected original_url to be set")
	}
	if _, err := os.Stat(*files[0].OriginalURL); err != nil {
		t.Errorf("expected file to exist on disk at %s: %v", *files[0].OriginalURL, err)
	}
}

func TestMonicaImportPhotos_Avatar(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportWithStorageTest(t)
	data := readMonicaFixture(t)

	_, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	if john.FileID == nil {
		t.Fatal("expected John.FileID to be set (first photo as avatar)")
	}

	var file models.File
	if err := svc.DB.First(&file, *john.FileID).Error; err != nil {
		t.Fatalf("avatar file not found: %v", err)
	}
	if file.Type != "photo" {
		t.Errorf("expected avatar file type=photo, got %s", file.Type)
	}
	if file.UfileableID == nil || *file.UfileableID != john.ID {
		t.Errorf("expected photo UfileableID=%s, got %v", john.ID, file.UfileableID)
	}
}

func TestMonicaImportDocuments(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportWithStorageTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedDocuments != 1 {
		t.Errorf("expected 1 imported document, got %d", resp.ImportedDocuments)
	}

	var files []models.File
	if err := svc.DB.Where("vault_id = ? AND type = ?", vaultID, "document").Find(&files).Error; err != nil {
		t.Fatalf("failed to query files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 document file record, got %d", len(files))
	}
	if files[0].MimeType != "application/pdf" {
		t.Errorf("expected mime_type=application/pdf, got %s", files[0].MimeType)
	}
	if files[0].Name != "resume.pdf" {
		t.Errorf("expected name=resume.pdf, got %s", files[0].Name)
	}

	var john models.Contact
	if err := svc.DB.Where("vault_id = ? AND first_name = ?", vaultID, "John").First(&john).Error; err != nil {
		t.Fatalf("John not found: %v", err)
	}
	if files[0].UfileableID == nil || *files[0].UfileableID != john.ID {
		t.Errorf("expected document UfileableID=%s, got %v", john.ID, files[0].UfileableID)
	}
}

func TestMonicaImportPhotos_InvalidBase64(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportWithStorageTest(t)

	exportJSON := `{
		"version": "1.0-preview.1",
		"account": {
			"uuid": "test-account",
			"data": [
				{"count": 1, "type": "contacts", "values": [
					{"uuid": "c1", "properties": {"first_name": "Alice"}, "data": [
						{"count": 1, "type": "photos", "values": ["photo-bad"]}
					]}
				]},
				{"count": 1, "type": "photos", "values": [
					{"uuid": "photo-bad", "properties": {
						"original_filename": "bad.png",
						"mime_type": "image/png",
						"dataUrl": "data:image/png;base64,!!!INVALID!!!"
					}}
				]}
			],
			"properties": {},
			"instance": {}
		}
	}`

	resp, err := svc.Import(vaultID, userID, []byte(exportJSON))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedPhotos != 0 {
		t.Errorf("expected 0 imported photos, got %d", resp.ImportedPhotos)
	}

	foundBase64Error := false
	for _, e := range resp.Errors {
		if strings.Contains(e, "base64 decode failed") {
			foundBase64Error = true
			break
		}
	}
	if !foundBase64Error {
		t.Errorf("expected error about base64 decode, errors: %v", resp.Errors)
	}

	var files []models.File
	svc.DB.Where("vault_id = ? AND type = ?", vaultID, "photo").Find(&files)
	if len(files) != 0 {
		t.Errorf("expected no file records for invalid base64, got %d", len(files))
	}
}

func TestMonicaImportPhotos_NoUploadDir(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	data := readMonicaFixture(t)

	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedPhotos != 0 {
		t.Errorf("expected 0 imported photos when UploadDir is empty, got %d", resp.ImportedPhotos)
	}
	if resp.ImportedDocuments != 0 {
		t.Errorf("expected 0 imported documents when UploadDir is empty, got %d", resp.ImportedDocuments)
	}

	foundSkipError := false
	for _, e := range resp.Errors {
		if strings.Contains(e, "no upload directory configured") {
			foundSkipError = true
			break
		}
	}
	if !foundSkipError {
		t.Errorf("expected error about no upload directory, errors: %v", resp.Errors)
	}
}

func TestMonicaImport_FeedRecords(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	svc.SetFeedRecorder(NewFeedRecorder(svc.DB))
	svc.SetSearchEngine(&search.NoopEngine{})

	data := readMonicaFixture(t)
	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 3 {
		t.Errorf("expected 3 contacts, got %d", resp.ImportedContacts)
	}

	var feedItems []models.ContactFeedItem
	if err := svc.DB.Where("action = ?", ActionContactCreated).Find(&feedItems).Error; err != nil {
		t.Fatalf("failed to query feed items: %v", err)
	}
	if len(feedItems) != 3 {
		t.Errorf("expected 3 feed items, got %d", len(feedItems))
	}
	for _, item := range feedItems {
		if item.ContactID == "" {
			t.Error("expected non-empty ContactID")
		}
		if item.AuthorID == nil || *item.AuthorID != userID {
			t.Errorf("expected AuthorID %s, got %v", userID, item.AuthorID)
		}
	}
}

func TestMonicaImport_SearchIndex(t *testing.T) {
	svc, vaultID, userID, _ := setupMonicaImportTest(t)
	svc.SetFeedRecorder(nil)
	svc.SetSearchEngine(&search.NoopEngine{})

	data := readMonicaFixture(t)
	resp, err := svc.Import(vaultID, userID, data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 3 {
		t.Errorf("expected 3 contacts, got %d", resp.ImportedContacts)
	}
}
