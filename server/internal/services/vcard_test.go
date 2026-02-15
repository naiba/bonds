package services

import (
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
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
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
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
