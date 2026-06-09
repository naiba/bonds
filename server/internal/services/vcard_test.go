package services

import (
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
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

	card := mustDecodeVCard(t, data)
	if got := card.Value(vcard.FieldVersion); got != "4.0" {
		t.Fatalf("expected VERSION 4.0, got %q", got)
	}
	if got := card.Kind(); got != vcard.KindIndividual {
		t.Fatalf("expected KIND individual, got %q", got)
	}
	if got := card.Value(vcard.FieldUID); got != contactID {
		t.Fatalf("expected UID %q, got %q", contactID, got)
	}
	if got := card.Value(vcard.FieldFormattedName); got != "John Doe" {
		t.Fatalf("expected FN John Doe, got %q", got)
	}
	if got := card.Value(vcard.FieldNickname); got != "JD" {
		t.Fatalf("expected nickname JD, got %q", got)
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

	var socialType models.ContactInformationType
	svc.db.Where("account_id = ? AND type = ?", vault.AccountID, "social").First(&socialType)

	svc.db.Create(&models.ContactInformation{ContactID: contactID, TypeID: phoneType.ID, Data: "+1-555-9999"})
	svc.db.Create(&models.ContactInformation{ContactID: contactID, TypeID: emailType.ID, Data: "john@example.com"})
	svc.db.Create(&models.ContactInformation{ContactID: contactID, TypeID: socialType.ID, Data: "https://twitter.com/john_doe"})

	var birthdateType models.ContactImportantDateType
	svc.db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&birthdateType)
	svc.db.Create(&models.ContactImportantDate{
		ContactID:                  contactID,
		ContactImportantDateTypeID: &birthdateType.ID,
		Label:                      "Birthdate",
		Day:                        intPtr(15),
		Month:                      intPtr(1),
		Year:                       intPtr(1990),
	})

	addr := models.Address{VaultID: vaultID, Line1: strPtrOrNil("456 Oak Ave"), City: strPtrOrNil("Chicago"), Province: strPtrOrNil("IL"), PostalCode: strPtrOrNil("60601"), Country: strPtrOrNil("US")}
	svc.db.Create(&addr)
	svc.db.Create(&models.ContactAddress{ContactID: contactID, AddressID: addr.ID})

	photoURL := "https://example.com/avatar.jpg"
	svc.db.Create(&models.File{VaultID: vaultID, UUID: "photo-uuid", OriginalURL: &photoURL, MimeType: "image/jpeg", Name: "avatar.jpg", Type: "image", Size: 1234})
	var photo models.File
	svc.db.Where("uuid = ?", "photo-uuid").First(&photo)
	svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("file_id", photo.ID)
	svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("description", "Trusted friend")
	svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("job_position", "Software Engineer")

	data, err := svc.ExportContact(contactID, vaultID)
	if err != nil {
		t.Fatalf("ExportContact failed: %v", err)
	}

	card := mustDecodeVCard(t, data)
	if got := card.Value(vcard.FieldVersion); got != "4.0" {
		t.Fatalf("expected VERSION 4.0, got %q", got)
	}
	if got := card.Kind(); got != vcard.KindIndividual {
		t.Fatalf("expected KIND individual, got %q", got)
	}
	if got := card.Value(vcard.FieldUID); got != contactID {
		t.Fatalf("expected UID %q, got %q", contactID, got)
	}
	if got := card.Value(vcard.FieldTitle); got != "Software Engineer" {
		t.Fatalf("expected TITLE Software Engineer, got %q", got)
	}
	if got := card.Value(vcard.FieldNote); got != "Trusted friend" {
		t.Fatalf("expected NOTE Trusted friend, got %q", got)
	}
	if got := card.Value(vcard.FieldBirthday); got != "1990-01-15" {
		t.Fatalf("expected BDAY 1990-01-15, got %q", got)
	}
	if got := card.Value(vcard.FieldFormattedName); got != "John Doe" {
		t.Fatalf("expected FN John Doe, got %q", got)
	}
	if got := card.Value(vcard.FieldNickname); got != "JD" {
		t.Fatalf("expected nickname JD, got %q", got)
	}
	if got := card.Value(vcard.FieldTelephone); got != "+1-555-9999" {
		t.Fatalf("expected phone +1-555-9999, got %q", got)
	}
	if got := card.Value(vcard.FieldEmail); got != "john@example.com" {
		t.Fatalf("expected email john@example.com, got %q", got)
	}
	if got := card.Value(vcard.FieldURL); got != "https://twitter.com/john_doe" {
		t.Fatalf("expected URL https://twitter.com/john_doe, got %q", got)
	}
	if got := card.Value(vcard.FieldIMPP); got != "https://twitter.com/john_doe" {
		t.Fatalf("expected IMPP https://twitter.com/john_doe, got %q", got)
	}
	socialFields := card["X-SOCIALPROFILE"]
	if len(socialFields) != 1 {
		t.Fatalf("expected one X-SOCIALPROFILE field, got %d", len(socialFields))
	}
	if got := socialFields[0].Value; got != "https://twitter.com/john_doe" {
		t.Fatalf("expected X-SOCIALPROFILE value https://twitter.com/john_doe, got %q", got)
	}
	if len(card[vcard.FieldPhoto]) != 1 {
		t.Fatalf("expected one PHOTO field, got %d", len(card[vcard.FieldPhoto]))
	}
	if got := card[vcard.FieldPhoto][0].Value; got != photoURL {
		t.Fatalf("expected PHOTO %q, got %q", photoURL, got)
	}
	if got := card[vcard.FieldPhoto][0].Params.Get(vcard.ParamMediaType); got != "image/jpeg" {
		t.Fatalf("expected PHOTO media type image/jpeg, got %q", got)
	}
	if len(card[vcard.FieldTelephone]) != 1 {
		t.Fatalf("expected one TEL field, got %d", len(card[vcard.FieldTelephone]))
	}
	if len(card[vcard.FieldEmail]) != 1 {
		t.Fatalf("expected one EMAIL field, got %d", len(card[vcard.FieldEmail]))
	}
	if len(card[vcard.FieldAddress]) != 1 {
		t.Fatalf("expected one ADR field, got %d", len(card[vcard.FieldAddress]))
	}
	address := card.Address()
	if address == nil {
		t.Fatal("expected preferred ADR field to be present")
	}
	if got := address.StreetAddress; got != "456 Oak Ave" {
		t.Fatalf("expected ADR street 456 Oak Ave, got %q", got)
	}
	if got := address.Locality; got != "Chicago" {
		t.Fatalf("expected ADR city Chicago, got %q", got)
	}
	if got := address.Region; got != "IL" {
		t.Fatalf("expected ADR region IL, got %q", got)
	}
	if got := address.PostalCode; got != "60601" {
		t.Fatalf("expected ADR postal code 60601, got %q", got)
	}
	if got := address.Country; got != "US" {
		t.Fatalf("expected ADR country US, got %q", got)
	}
}

func mustDecodeVCard(t *testing.T, data []byte) vcard.Card {
	t.Helper()
	card, err := vcard.NewDecoder(strings.NewReader(string(data))).Decode()
	if err != nil {
		t.Fatalf("decode vCard: %v", err)
	}
	return card
}
