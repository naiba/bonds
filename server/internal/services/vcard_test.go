package services

import (
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVCardTest(t *testing.T) (*VCardService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vcard-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "JD",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewVCardService(db), contact.ID, vault.ID, resp.User.ID
}

func TestExportContact(t *testing.T) {
	svc, contactID, vaultID, _ := setupVCardTest(t)

	data, err := svc.ExportContact(contactID, vaultID)
	if err != nil {
		t.Fatalf("ExportContact failed: %v", err)
	}

	vcardStr := string(data)
	if !strings.Contains(vcardStr, "BEGIN:VCARD") {
		t.Error("Expected vCard to contain BEGIN:VCARD")
	}
	if !strings.Contains(vcardStr, "END:VCARD") {
		t.Error("Expected vCard to contain END:VCARD")
	}
	if !strings.Contains(vcardStr, "John") {
		t.Error("Expected vCard to contain first name 'John'")
	}
	if !strings.Contains(vcardStr, "Doe") {
		t.Error("Expected vCard to contain last name 'Doe'")
	}
	if !strings.Contains(vcardStr, "JD") {
		t.Error("Expected vCard to contain nickname 'JD'")
	}
}

func TestExportContactNotFound(t *testing.T) {
	svc, _, vaultID, _ := setupVCardTest(t)

	_, err := svc.ExportContact("nonexistent-id", vaultID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestImportSingleVCard(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Smith;Jane;;;\r\nFN:Jane Smith\r\nNICKNAME:JS\r\nEND:VCARD\r\n"
	reader := strings.NewReader(vcardData)

	result, err := svc.ImportVCard(vaultID, userID, reader)
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Errorf("Expected 1 imported, got %d", result.ImportedCount)
	}
	if len(result.Contacts) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(result.Contacts))
	}
	if result.Contacts[0].FirstName != "Jane" {
		t.Errorf("Expected first name 'Jane', got '%s'", result.Contacts[0].FirstName)
	}
	if result.Contacts[0].LastName != "Smith" {
		t.Errorf("Expected last name 'Smith', got '%s'", result.Contacts[0].LastName)
	}
	if result.Contacts[0].Nickname != "JS" {
		t.Errorf("Expected nickname 'JS', got '%s'", result.Contacts[0].Nickname)
	}
}

func TestImportMultipleVCards(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:One;Contact;;;\r\nFN:Contact One\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nN:Two;Contact;;;\r\nFN:Contact Two\r\nEND:VCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nN:Three;Contact;;;\r\nFN:Contact Three\r\nEND:VCARD\r\n"
	reader := strings.NewReader(vcardData)

	result, err := svc.ImportVCard(vaultID, userID, reader)
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 3 {
		t.Errorf("Expected 3 imported, got %d", result.ImportedCount)
	}
	if len(result.Contacts) != 3 {
		t.Errorf("Expected 3 contacts, got %d", len(result.Contacts))
	}
}

func TestExportVault(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	contactSvc := NewContactService(svc.db)
	_, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	data, err := svc.ExportVault(vaultID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	vcardStr := string(data)
	count := strings.Count(vcardStr, "BEGIN:VCARD")
	if count != 2 {
		t.Errorf("Expected 2 vCards in vault export, got %d", count)
	}
}

func TestImportVCardWithContactInfo(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Doe;Jane;;;\r\nFN:Jane Doe\r\n" +
		"TEL;TYPE=VOICE:+1-555-1234\r\nTEL;TYPE=VOICE:+1-555-5678\r\n" +
		"EMAIL;TYPE=INTERNET:jane@example.com\r\n" +
		"END:VCARD\r\n"

	result, err := svc.ImportVCard(vaultID, userID, strings.NewReader(vcardData))
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("Expected 1 imported, got %d", result.ImportedCount)
	}

	contactID := result.Contacts[0].ID

	var infos []models.ContactInformation
	svc.db.Preload("ContactInformationType").Where("contact_id = ?", contactID).Find(&infos)

	phoneCount := 0
	emailCount := 0
	for _, info := range infos {
		typeName := ptrToStr(info.ContactInformationType.Type)
		switch typeName {
		case "phone":
			phoneCount++
			if info.Data != "+1-555-1234" && info.Data != "+1-555-5678" {
				t.Errorf("Unexpected phone: %s", info.Data)
			}
		case "email":
			emailCount++
			if info.Data != "jane@example.com" {
				t.Errorf("Expected email 'jane@example.com', got '%s'", info.Data)
			}
		}
	}
	if phoneCount != 2 {
		t.Errorf("Expected 2 phone records, got %d", phoneCount)
	}
	if emailCount != 1 {
		t.Errorf("Expected 1 email record, got %d", emailCount)
	}
}

func TestImportVCardWithAddress(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Doe;John;;;\r\nFN:John Doe\r\n" +
		"ADR;TYPE=HOME:;;123 Main St;Springfield;IL;62701;US\r\n" +
		"END:VCARD\r\n"

	result, err := svc.ImportVCard(vaultID, userID, strings.NewReader(vcardData))
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("Expected 1 imported, got %d", result.ImportedCount)
	}

	contactID := result.Contacts[0].ID

	var pivots []models.ContactAddress
	svc.db.Where("contact_id = ?", contactID).Find(&pivots)
	if len(pivots) != 1 {
		t.Fatalf("Expected 1 ContactAddress pivot, got %d", len(pivots))
	}

	var addr models.Address
	if err := svc.db.First(&addr, "id = ?", pivots[0].AddressID).Error; err != nil {
		t.Fatalf("Address not found: %v", err)
	}
	if ptrToStr(addr.Line1) != "123 Main St" {
		t.Errorf("Expected Line1 '123 Main St', got '%s'", ptrToStr(addr.Line1))
	}
	if ptrToStr(addr.City) != "Springfield" {
		t.Errorf("Expected City 'Springfield', got '%s'", ptrToStr(addr.City))
	}
	if ptrToStr(addr.Province) != "IL" {
		t.Errorf("Expected Province 'IL', got '%s'", ptrToStr(addr.Province))
	}
	if ptrToStr(addr.PostalCode) != "62701" {
		t.Errorf("Expected PostalCode '62701', got '%s'", ptrToStr(addr.PostalCode))
	}
	if ptrToStr(addr.Country) != "US" {
		t.Errorf("Expected Country 'US', got '%s'", ptrToStr(addr.Country))
	}
}

func TestImportVCardWithBirthday(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Doe;Jane;;;\r\nFN:Jane Doe\r\n" +
		"BDAY:1990-01-15\r\n" +
		"END:VCARD\r\n"

	result, err := svc.ImportVCard(vaultID, userID, strings.NewReader(vcardData))
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("Expected 1 imported, got %d", result.ImportedCount)
	}

	contactID := result.Contacts[0].ID

	var dates []models.ContactImportantDate
	svc.db.Where("contact_id = ?", contactID).Find(&dates)
	if len(dates) != 1 {
		t.Fatalf("Expected 1 important date, got %d", len(dates))
	}

	d := dates[0]
	if d.Label != "Birthdate" {
		t.Errorf("Expected label 'Birthdate', got '%s'", d.Label)
	}
	if d.Year == nil || *d.Year != 1990 {
		t.Errorf("Expected year 1990, got %v", d.Year)
	}
	if d.Month == nil || *d.Month != 1 {
		t.Errorf("Expected month 1, got %v", d.Month)
	}
	if d.Day == nil || *d.Day != 15 {
		t.Errorf("Expected day 15, got %v", d.Day)
	}
}

func TestImportVCardWithJobTitle(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Doe;Jane;;;\r\nFN:Jane Doe\r\n" +
		"TITLE:Software Engineer\r\n" +
		"END:VCARD\r\n"

	result, err := svc.ImportVCard(vaultID, userID, strings.NewReader(vcardData))
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("Expected 1 imported, got %d", result.ImportedCount)
	}

	if result.Contacts[0].JobPosition != "Software Engineer" {
		t.Errorf("Expected JobPosition 'Software Engineer', got '%s'", result.Contacts[0].JobPosition)
	}
}

func TestImportVCardSkipsBadCards(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)

	vcardData := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Good;First;;;\r\nFN:First Good\r\nEND:VCARD\r\n" +
		"BEGIN:NOTAVCARD\r\nFN:Bad\r\nEND:NOTAVCARD\r\n" +
		"BEGIN:VCARD\r\nVERSION:3.0\r\nN:Good;Second;;;\r\nFN:Second Good\r\nEND:VCARD\r\n"

	result, err := svc.ImportVCard(vaultID, userID, strings.NewReader(vcardData))
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}
	if result.ImportedCount != 2 {
		t.Errorf("Expected 2 imported, got %d", result.ImportedCount)
	}
	if result.SkippedCount < 1 {
		t.Errorf("Expected at least 1 skipped, got %d", result.SkippedCount)
	}
	if len(result.Errors) < 1 {
		t.Errorf("Expected at least 1 error message, got %d", len(result.Errors))
	}
}

func TestExportContactWithContactInfo(t *testing.T) {
	svc, contactID, vaultID, _ := setupVCardTest(t)

	var vault models.Vault
	svc.db.First(&vault, "id = ?", vaultID)

	var phoneType models.ContactInformationType
	svc.db.Where("account_id = ? AND type = ?", vault.AccountID, "phone").First(&phoneType)

	var emailType models.ContactInformationType
	svc.db.Where("account_id = ? AND type = ?", vault.AccountID, "email").First(&emailType)

	svc.db.Create(&models.ContactInformation{ContactID: contactID, TypeID: phoneType.ID, Data: "+1-555-9999"})
	svc.db.Create(&models.ContactInformation{ContactID: contactID, TypeID: emailType.ID, Data: "john@example.com"})

	addr := models.Address{VaultID: vaultID, Line1: strPtrOrNil("456 Oak Ave"), City: strPtrOrNil("Chicago"), Province: strPtrOrNil("IL"), PostalCode: strPtrOrNil("60601"), Country: strPtrOrNil("US")}
	svc.db.Create(&addr)
	svc.db.Create(&models.ContactAddress{ContactID: contactID, AddressID: addr.ID})

	data, err := svc.ExportContact(contactID, vaultID)
	if err != nil {
		t.Fatalf("ExportContact failed: %v", err)
	}

	vcardStr := string(data)
	if !strings.Contains(vcardStr, "+1-555-9999") {
		t.Error("Expected vCard to contain phone number")
	}
	if !strings.Contains(vcardStr, "john@example.com") {
		t.Error("Expected vCard to contain email")
	}
	if !strings.Contains(vcardStr, "456 Oak Ave") {
		t.Error("Expected vCard to contain address street")
	}
	if !strings.Contains(vcardStr, "Chicago") {
		t.Error("Expected vCard to contain address city")
	}
}
