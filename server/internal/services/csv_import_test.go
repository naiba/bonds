package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// setupCSVImportTest creates a DB, account, vault, and a service instance with
// feed, search, and DAV push wired in (feed and search via real in-memory
// implementations; DAV push via nil, which is safe for tests).
func setupCSVImportTest(t *testing.T) (*CSVImportService, *gorm.DB, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "csv-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewCSVImportService(db)
	svc.SetFeedRecorder(NewFeedRecorder(db))
	svc.SetSearchService(NewSearchService(&search.NoopEngine{}))
	// DAV push left nil — safe for tests

	return svc, db, vault.ID, resp.User.ID
}

func csvQuote(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func csvData(headers []string, rows ...[]string) []byte {
	var b strings.Builder
	quoted := make([]string, len(headers))
	for i, h := range headers {
		quoted[i] = csvQuote(h)
	}
	b.WriteString(strings.Join(quoted, ",") + "\n")
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, v := range row {
			cells[i] = csvQuote(v)
		}
		b.WriteString(strings.Join(cells, ",") + "\n")
	}
	return []byte(b.String())
}

func defaultMapping() dto.CSVColumnMapping {
	return dto.CSVColumnMapping{
		FirstName: "first_name",
		LastName:  "last_name",
	}
}

// ---------------------------------------------------------------------------
// Basic import
// ---------------------------------------------------------------------------

func TestCSVImport_SuccessBasic(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	data := csvData(
		[]string{"first_name", "last_name"},
		[]string{"Alice", "Smith"},
		[]string{"Bob", "Jones"},
	)

	resp, err := svc.Import(vaultID, userID, data, defaultMapping())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 2 {
		t.Errorf("expected 2 imported, got %d", resp.ImportedContacts)
	}
	if resp.SkippedCount != 0 {
		t.Errorf("expected 0 skipped, got %d", resp.SkippedCount)
	}

	var count int64
	db.Model(&models.Contact{}).Where("vault_id = ? AND listed = ?", vaultID, true).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 contacts in DB, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// UTF-8 BOM stripping
// ---------------------------------------------------------------------------

func TestCSVImport_UTFBOM(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	raw := "\xEF\xBB\xBFfirst_name,last_name\nCarol,White\n"

	resp, err := svc.Import(vaultID, userID, []byte(raw), defaultMapping())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}

	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Carol").First(&contact).Error; err != nil {
		t.Errorf("Carol not found in DB: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Quoted commas inside cells
// ---------------------------------------------------------------------------

func TestCSVImport_QuotedCommas(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	// Tags field contains comma-separated values inside a quoted cell.
	m := dto.CSVColumnMapping{
		FirstName: "first_name",
		Tags:      "tags",
	}
	raw := `first_name,tags` + "\n" + `"Dana","friends,family"` + "\n"

	resp, err := svc.Import(vaultID, userID, []byte(raw), m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}

	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Dana").First(&contact).Error; err != nil {
		t.Fatalf("Dana not found: %v", err)
	}
	var labelCount int64
	db.Model(&models.ContactLabel{}).Where("contact_id = ?", contact.ID).Count(&labelCount)
	if labelCount != 2 {
		t.Errorf("expected 2 labels, got %d", labelCount)
	}
}

// ---------------------------------------------------------------------------
// Missing first name → skipped
// ---------------------------------------------------------------------------

func TestCSVImport_MissingFirstName(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	data := csvData(
		[]string{"first_name", "last_name"},
		[]string{"Eve", "Green"},
		[]string{"", "Nameless"},
	)

	resp, err := svc.Import(vaultID, userID, data, defaultMapping())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}
	if resp.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", resp.SkippedCount)
	}

	var count int64
	db.Model(&models.Contact{}).Where("vault_id = ? AND listed = ?", vaultID, true).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 contact in DB, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Malformed CSV → 400-equivalent error from Import
// ---------------------------------------------------------------------------

func TestCSVImport_MalformedCSV(t *testing.T) {
	svc, _, vaultID, userID := setupCSVImportTest(t)

	// A row with more fields than the header triggers csv.ErrFieldCount.
	raw := "first_name,last_name\nFrank,Jones,ExtraUnexpectedField\n"

	_, err := svc.Import(vaultID, userID, []byte(raw), defaultMapping())
	if err == nil {
		t.Error("expected parse error for malformed CSV (field count mismatch), got nil")
	}
}

// ---------------------------------------------------------------------------
// Email mapping
// ---------------------------------------------------------------------------

func TestCSVImport_EmailMapping(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	// SeedAccountDefaults already creates the "seed.contact_info_types.email_address"
	// type when the user registers, so no extra setup is needed here.

	m := dto.CSVColumnMapping{FirstName: "first_name", Email: "email"}
	data := csvData(
		[]string{"first_name", "email"},
		[]string{"Grace", "grace@example.com"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}

	var contact models.Contact
	db.Where("vault_id = ? AND first_name = ?", vaultID, "Grace").First(&contact)
	var ciCount int64
	db.Model(&models.ContactInformation{}).Where("contact_id = ? AND data = ?", contact.ID, "grace@example.com").Count(&ciCount)
	if ciCount != 1 {
		t.Errorf("expected 1 ContactInformation row for email, got %d", ciCount)
	}
}

// ---------------------------------------------------------------------------
// Phone mapping
// ---------------------------------------------------------------------------

func TestCSVImport_PhoneMapping(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	// SeedAccountDefaults already creates the "seed.contact_info_types.phone"
	// type when the user registers, so no extra setup is needed here.

	m := dto.CSVColumnMapping{FirstName: "first_name", Phone: "phone"}
	data := csvData(
		[]string{"first_name", "phone"},
		[]string{"Henry", "+1-555-0100"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}

	var contact models.Contact
	db.Where("vault_id = ? AND first_name = ?", vaultID, "Henry").First(&contact)
	var ciCount int64
	db.Model(&models.ContactInformation{}).Where("contact_id = ? AND data = ?", contact.ID, "+1-555-0100").Count(&ciCount)
	if ciCount != 1 {
		t.Errorf("expected 1 ContactInformation row for phone, got %d", ciCount)
	}
}

// ---------------------------------------------------------------------------
// Birthday mapping
// ---------------------------------------------------------------------------

func TestCSVImport_BirthdayMapping(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	// SeedVaultDefaults already creates the "birthdate" important-date type
	// when the vault is created, so no extra setup is needed here.

	m := dto.CSVColumnMapping{FirstName: "first_name", Birthday: "birthday"}
	data := csvData(
		[]string{"first_name", "birthday"},
		[]string{"Iris", "1990-06-15"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}

	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Iris").First(&contact).Error; err != nil {
		t.Fatalf("Iris not found: %v", err)
	}
	var dateCount int64
	db.Model(&models.ContactImportantDate{}).Where("contact_id = ?", contact.ID).Count(&dateCount)
	if dateCount != 1 {
		t.Errorf("expected 1 important date, got %d", dateCount)
	}
}

// ---------------------------------------------------------------------------
// Address mapping
// ---------------------------------------------------------------------------

func TestCSVImport_AddressMapping(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	// SeedAccountDefaults already creates the "seed.address_types.home" type
	// when the user registers, so no extra setup is needed here.

	m := dto.CSVColumnMapping{
		FirstName:     "first_name",
		AddressStreet: "street",
		AddressCity:   "city",
		AddressCountry: "country",
	}
	data := csvData(
		[]string{"first_name", "street", "city", "country"},
		[]string{"Jack", "123 Main St", "Springfield", "US"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}

	var contact models.Contact
	db.Where("vault_id = ? AND first_name = ?", vaultID, "Jack").First(&contact)
	var addrCount int64
	db.Model(&models.ContactAddress{}).Where("contact_id = ?", contact.ID).Count(&addrCount)
	if addrCount != 1 {
		t.Errorf("expected 1 address, got %d", addrCount)
	}
}

// ---------------------------------------------------------------------------
// Label creation
// ---------------------------------------------------------------------------

func TestCSVImport_LabelCreation(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	m := dto.CSVColumnMapping{FirstName: "first_name", Tags: "tags"}
	data := csvData(
		[]string{"first_name", "tags"},
		[]string{"Karen", "colleague"},
		[]string{"Leo", "colleague"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 2 {
		t.Errorf("expected 2 imported, got %d", resp.ImportedContacts)
	}

	// The "colleague" label should be created once and reused.
	var labelCount int64
	db.Model(&models.Label{}).Where("vault_id = ? AND name = ?", vaultID, "colleague").Count(&labelCount)
	if labelCount != 1 {
		t.Errorf("expected 1 'colleague' label, got %d", labelCount)
	}
	var clCount int64
	db.Model(&models.ContactLabel{}).Count(&clCount)
	if clCount != 2 {
		t.Errorf("expected 2 contact-label links, got %d", clCount)
	}
}

// ---------------------------------------------------------------------------
// Group not found → reported as soft error, contact still imported
// ---------------------------------------------------------------------------

func TestCSVImport_GroupNotFound(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	m := dto.CSVColumnMapping{FirstName: "first_name", Groups: "group"}
	data := csvData(
		[]string{"first_name", "group"},
		[]string{"Mia", "NonExistentGroup"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	// Contact is imported despite the group not being found.
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected a soft error for missing group")
	}

	var count int64
	db.Model(&models.Contact{}).Where("vault_id = ? AND first_name = ?", vaultID, "Mia").Count(&count)
	if count != 1 {
		t.Errorf("expected Mia in DB, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Transaction atomicity: Contact and CVU are created together
// ---------------------------------------------------------------------------

func TestCSVImport_ContactAndCVUCreatedTogether(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	data := csvData(
		[]string{"first_name"},
		[]string{"Nick"},
	)

	resp, err := svc.Import(vaultID, userID, data, dto.CSVColumnMapping{FirstName: "first_name"})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}

	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Nick").First(&contact).Error; err != nil {
		t.Fatalf("Contact not found: %v", err)
	}

	// CVU must exist alongside the contact.
	var cvu models.ContactVaultUser
	if err := db.Where("contact_id = ? AND vault_id = ? AND user_id = ?", contact.ID, vaultID, userID).First(&cvu).Error; err != nil {
		t.Errorf("ContactVaultUser not found for imported contact: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Import into nonexistent vault returns error and creates no contacts
// ---------------------------------------------------------------------------

func TestCSVImport_NonexistentVault(t *testing.T) {
	svc, db, _, userID := setupCSVImportTest(t)

	data := csvData(
		[]string{"first_name"},
		[]string{"Ghost"},
	)

	_, err := svc.Import("nonexistent-vault-id", userID, data, dto.CSVColumnMapping{FirstName: "first_name"})
	if err == nil {
		t.Error("expected error for nonexistent vault, got nil")
	}

	// The nonexistent vault ID should yield zero contacts; the real vault's
	// self-contact (Listed=false) is excluded from this assertion.
	var count int64
	db.Model(&models.Contact{}).Where("vault_id = ? AND listed = ?", "nonexistent-vault-id", true).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 contacts created, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Feed entries created after successful import
// ---------------------------------------------------------------------------

func TestCSVImport_FeedRecords(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	data := csvData(
		[]string{"first_name"},
		[]string{"Olivia"},
		[]string{"Paul"},
	)

	resp, err := svc.Import(vaultID, userID, data, dto.CSVColumnMapping{FirstName: "first_name"})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 2 {
		t.Errorf("expected 2 imported, got %d", resp.ImportedContacts)
	}

	var feedItems []models.ContactFeedItem
	db.Where("action = ?", ActionContactCreated).Find(&feedItems)
	if len(feedItems) != 2 {
		t.Errorf("expected 2 feed items, got %d", len(feedItems))
	}
	for _, item := range feedItems {
		if item.ContactID == "" {
			t.Error("expected non-empty ContactID in feed item")
		}
		if item.AuthorID == nil || *item.AuthorID != userID {
			t.Errorf("expected AuthorID %s, got %v", userID, item.AuthorID)
		}
	}
}

// ---------------------------------------------------------------------------
// Note with feed record
// ---------------------------------------------------------------------------

func TestCSVImport_NoteWithFeedRecord(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	m := dto.CSVColumnMapping{FirstName: "first_name", Notes: "notes"}
	data := csvData(
		[]string{"first_name", "notes"},
		[]string{"Quinn", "This is a note"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}

	var contact models.Contact
	db.Where("vault_id = ? AND first_name = ?", vaultID, "Quinn").First(&contact)
	var note models.Note
	if err := db.Where("contact_id = ?", contact.ID).First(&note).Error; err != nil {
		t.Fatalf("note not found: %v", err)
	}
	if note.Body != "This is a note" {
		t.Errorf("expected note body 'This is a note', got %q", note.Body)
	}

	var noteFeed []models.ContactFeedItem
	db.Where("action = ? AND contact_id = ?", ActionNoteCreated, contact.ID).Find(&noteFeed)
	if len(noteFeed) != 1 {
		t.Errorf("expected 1 note feed item, got %d", len(noteFeed))
	}
}

// ---------------------------------------------------------------------------
// Company stored as note prefix
// ---------------------------------------------------------------------------

func TestCSVImport_CompanyNote(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	m := dto.CSVColumnMapping{FirstName: "first_name", Company: "company"}
	data := csvData(
		[]string{"first_name", "company"},
		[]string{"Rachel", "Acme Corp"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}

	var contact models.Contact
	db.Where("vault_id = ? AND first_name = ?", vaultID, "Rachel").First(&contact)
	var note models.Note
	db.Where("contact_id = ?", contact.ID).First(&note)
	if !strings.HasPrefix(note.Body, "Company: Acme Corp") {
		t.Errorf("expected note to start with 'Company: Acme Corp', got %q", note.Body)
	}
}

// ---------------------------------------------------------------------------
// LastUpdatedAt is set
// ---------------------------------------------------------------------------

func TestCSVImport_LastUpdatedAt(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	data := csvData(
		[]string{"first_name"},
		[]string{"Tina"},
	)

	resp, err := svc.Import(vaultID, userID, data, dto.CSVColumnMapping{FirstName: "first_name"})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected 1 imported, got %d", resp.ImportedContacts)
	}

	var contact models.Contact
	db.Where("vault_id = ? AND first_name = ?", vaultID, "Tina").First(&contact)
	if contact.LastUpdatedAt == nil {
		t.Error("expected LastUpdatedAt to be set, got nil")
	}
}

// ---------------------------------------------------------------------------
// Empty CSV (header only) → zero imports, no error
// ---------------------------------------------------------------------------

func TestCSVImport_EmptyCSV(t *testing.T) {
	svc, _, vaultID, userID := setupCSVImportTest(t)

	data := []byte("first_name,last_name\n")

	resp, err := svc.Import(vaultID, userID, data, defaultMapping())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 0 {
		t.Errorf("expected 0 imported, got %d", resp.ImportedContacts)
	}
}

// ---------------------------------------------------------------------------
// Multiple birthday formats
// ---------------------------------------------------------------------------

func TestCSVImport_BirthdayFormats(t *testing.T) {
	svc, db, vaultID, userID := setupCSVImportTest(t)

	// SeedVaultDefaults already creates the "birthdate" important-date type.

	cases := []struct {
		name   string
		bdVal  string
		expDay int
	}{
		{"ISO", "1985-03-20", 20},
		{"EU", "20/03/1985", 20},
		{"US", "03/20/1985", 20},
		{"LongForm", "20 March 1985", 20},
		{"LongFormUS", "March 20, 1985", 20},
	}

	m := dto.CSVColumnMapping{FirstName: "first_name", Birthday: "birthday"}
	for i, tc := range cases {
		name := fmt.Sprintf("%s_%d", tc.name, i)
		data := csvData(
			[]string{"first_name", "birthday"},
			[]string{name, tc.bdVal},
		)
		resp, err := svc.Import(vaultID, userID, data, m)
		if err != nil {
			t.Errorf("%s: Import failed: %v", tc.name, err)
			continue
		}
		if len(resp.Errors) > 0 {
			t.Errorf("%s: unexpected errors: %v", tc.name, resp.Errors)
		}
		var contact models.Contact
		db.Where("vault_id = ? AND first_name = ?", vaultID, name).First(&contact)
		var date models.ContactImportantDate
		if err := db.Where("contact_id = ?", contact.ID).First(&date).Error; err != nil {
			t.Errorf("%s: birthday not found: %v", tc.name, err)
			continue
		}
		if date.Day == nil || *date.Day != tc.expDay {
			t.Errorf("%s: expected day %d, got %v", tc.name, tc.expDay, date.Day)
		}
	}
}

// ---------------------------------------------------------------------------
// Unrecognised birthday format → soft error, contact still imported
// ---------------------------------------------------------------------------

func TestCSVImport_BadBirthdayFormat(t *testing.T) {
	svc, _, vaultID, userID := setupCSVImportTest(t)

	// SeedVaultDefaults already creates the "birthdate" important-date type.

	m := dto.CSVColumnMapping{FirstName: "first_name", Birthday: "birthday"}
	data := csvData(
		[]string{"first_name", "birthday"},
		[]string{"Uma", "not-a-date"},
	)

	resp, err := svc.Import(vaultID, userID, data, m)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if resp.ImportedContacts != 1 {
		t.Errorf("expected contact imported despite bad birthday, got %d", resp.ImportedContacts)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected a soft error for bad birthday format")
	}
}

// ---------------------------------------------------------------------------
// File size constant is exported and correct
// ---------------------------------------------------------------------------

func TestCSVImport_MaxFileSizeConstant(t *testing.T) {
	if MaxCSVFileSize != 10*1024*1024 {
		t.Errorf("expected MaxCSVFileSize = 10MB, got %d", MaxCSVFileSize)
	}
}
