package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactTest(t *testing.T) (*ContactService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewContactService(db), vault.ID, resp.User.ID, resp.User.AccountID
}

func TestCreateContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "JD",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.FirstName != "John" {
		t.Errorf("Expected first_name 'John', got '%s'", contact.FirstName)
	}
	if contact.LastName != "Doe" {
		t.Errorf("Expected last_name 'Doe', got '%s'", contact.LastName)
	}
	if contact.Nickname != "JD" {
		t.Errorf("Expected nickname 'JD', got '%s'", contact.Nickname)
	}
	if contact.ID == "" {
		t.Error("Expected contact ID to be non-empty")
	}
	if contact.VaultID != vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", vaultID, contact.VaultID)
	}
}

func TestListContacts(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	contacts, meta, err := svc.ListContacts(vaultID, userID, 1, 15, "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}
	if meta.Total != 2 {
		t.Errorf("Expected total 2, got %d", meta.Total)
	}
}

func TestUpdateContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Original"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName: "Updated",
		LastName:  "Name",
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	if updated.FirstName != "Updated" {
		t.Errorf("Expected first_name 'Updated', got '%s'", updated.FirstName)
	}
	if updated.LastName != "Name" {
		t.Errorf("Expected last_name 'Name', got '%s'", updated.LastName)
	}
}

func TestDeleteContact(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "ToDelete"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	if err := svc.DeleteContact(created.ID, vaultID); err != nil {
		t.Fatalf("DeleteContact failed: %v", err)
	}

	contacts, _, err := svc.ListContacts(vaultID, userID, 1, 15, "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts after delete, got %d", len(contacts))
	}
}

func TestToggleArchive(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Archive"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	toggled, err := svc.ToggleArchive(created.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive failed: %v", err)
	}
	if !toggled.IsArchived {
		t.Error("Expected contact to be archived after toggle")
	}

	toggledBack, err := svc.ToggleArchive(created.ID, vaultID)
	if err != nil {
		t.Fatalf("ToggleArchive back failed: %v", err)
	}
	if toggledBack.IsArchived {
		t.Error("Expected contact to not be archived after second toggle")
	}
}

func TestToggleFavorite(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "toggle-fav@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	resp2, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Other",
		LastName:  "User",
		Email:     "toggle-fav2@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewContactService(db)
	created, err := svc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Favorite"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Use a different user who has no CVU yet — this exercises the Create path
	toggled, err := svc.ToggleFavorite(created.ID, resp2.User.ID, vault.ID)
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	if !toggled.IsFavorite {
		t.Error("Expected contact to be favorite after toggle")
	}

	// Toggle back — exercises the Save/update path
	toggledBack, err := svc.ToggleFavorite(created.ID, resp2.User.ID, vault.ID)
	if err != nil {
		t.Fatalf("ToggleFavorite back failed: %v", err)
	}
	if toggledBack.IsFavorite {
		t.Error("Expected contact to not be favorite after second toggle")
	}
}

func TestContactNotFound(t *testing.T) {
	svc, _, _, _ := setupContactTest(t)

	_, err := svc.GetContact("nonexistent-id", "some-user", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.UpdateContact("nonexistent-id", "some-vault", dto.UpdateContactRequest{FirstName: "nope"})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	err = svc.DeleteContact("nonexistent-id", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.ToggleArchive("nonexistent-id", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}

	_, err = svc.ToggleFavorite("nonexistent-id", "some-user", "some-vault")
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func ptrBool(b bool) *bool { return &b }
func ptrUint(u uint) *uint { return &u }

func TestCreateContact_WithAllFields(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)

	db := svc.db
	gender := models.Gender{AccountID: accountID, Name: strPtrOrNil("Custom")}
	db.Create(&gender)
	pronoun := models.Pronoun{AccountID: accountID, Name: strPtrOrNil("they/them/custom")}
	db.Create(&pronoun)

	var tmpl models.Template
	db.Where("account_id = ?", accountID).First(&tmpl)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName:  "John",
		LastName:   "Doe",
		MiddleName: "Michael",
		Nickname:   "Johnny",
		MaidenName: "Smith",
		Prefix:     "Mr.",
		Suffix:     "Jr.",
		GenderID:   ptrUint(gender.ID),
		PronounID:  ptrUint(pronoun.ID),
		TemplateID: ptrUint(tmpl.ID),
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.FirstName != "John" {
		t.Errorf("Expected first_name 'John', got '%s'", contact.FirstName)
	}
	if contact.LastName != "Doe" {
		t.Errorf("Expected last_name 'Doe', got '%s'", contact.LastName)
	}
	if contact.MiddleName != "Michael" {
		t.Errorf("Expected middle_name 'Michael', got '%s'", contact.MiddleName)
	}
	if contact.Nickname != "Johnny" {
		t.Errorf("Expected nickname 'Johnny', got '%s'", contact.Nickname)
	}
	if contact.MaidenName != "Smith" {
		t.Errorf("Expected maiden_name 'Smith', got '%s'", contact.MaidenName)
	}
	if contact.Prefix != "Mr." {
		t.Errorf("Expected prefix 'Mr.', got '%s'", contact.Prefix)
	}
	if contact.Suffix != "Jr." {
		t.Errorf("Expected suffix 'Jr.', got '%s'", contact.Suffix)
	}
	if contact.GenderID == nil || *contact.GenderID != gender.ID {
		t.Errorf("Expected gender_id %d, got %v", gender.ID, contact.GenderID)
	}
	if contact.PronounID == nil || *contact.PronounID != pronoun.ID {
		t.Errorf("Expected pronoun_id %d, got %v", pronoun.ID, contact.PronounID)
	}
	if contact.TemplateID == nil || *contact.TemplateID != tmpl.ID {
		t.Errorf("Expected template_id %d, got %v", tmpl.ID, contact.TemplateID)
	}
	if !contact.Listed {
		t.Error("Expected listed to be true by default")
	}
	if contact.IsArchived {
		t.Error("Expected is_archived to be false by default")
	}
}

func TestCreateContact_WithListedFalse(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	contact, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Unlisted",
		Listed:    ptrBool(false),
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	if contact.Listed {
		t.Error("Expected listed to be false")
	}
	if !contact.IsArchived {
		t.Error("Expected is_archived to be true when listed is false")
	}
}

func TestUpdateContact_WithAllFields(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)

	created, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Original",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	db := svc.db
	gender := models.Gender{AccountID: accountID, Name: strPtrOrNil("NonBinary")}
	db.Create(&gender)
	pronoun := models.Pronoun{AccountID: accountID, Name: strPtrOrNil("ze/zir")}
	db.Create(&pronoun)

	var tmpl models.Template
	db.Where("account_id = ?", accountID).First(&tmpl)

	updated, err := svc.UpdateContact(created.ID, vaultID, dto.UpdateContactRequest{
		FirstName:  "Updated",
		LastName:   "NewLast",
		MiddleName: "NewMiddle",
		Nickname:   "NewNick",
		MaidenName: "NewMaiden",
		Prefix:     "Dr.",
		Suffix:     "III",
		GenderID:   ptrUint(gender.ID),
		PronounID:  ptrUint(pronoun.ID),
		TemplateID: ptrUint(tmpl.ID),
	})
	if err != nil {
		t.Fatalf("UpdateContact failed: %v", err)
	}
	if updated.FirstName != "Updated" {
		t.Errorf("Expected first_name 'Updated', got '%s'", updated.FirstName)
	}
	if updated.LastName != "NewLast" {
		t.Errorf("Expected last_name 'NewLast', got '%s'", updated.LastName)
	}
	if updated.MiddleName != "NewMiddle" {
		t.Errorf("Expected middle_name 'NewMiddle', got '%s'", updated.MiddleName)
	}
	if updated.Nickname != "NewNick" {
		t.Errorf("Expected nickname 'NewNick', got '%s'", updated.Nickname)
	}
	if updated.MaidenName != "NewMaiden" {
		t.Errorf("Expected maiden_name 'NewMaiden', got '%s'", updated.MaidenName)
	}
	if updated.Prefix != "Dr." {
		t.Errorf("Expected prefix 'Dr.', got '%s'", updated.Prefix)
	}
	if updated.Suffix != "III" {
		t.Errorf("Expected suffix 'III', got '%s'", updated.Suffix)
	}
	if updated.GenderID == nil || *updated.GenderID != gender.ID {
		t.Errorf("Expected gender_id %d, got %v", gender.ID, updated.GenderID)
	}
	if updated.PronounID == nil || *updated.PronounID != pronoun.ID {
		t.Errorf("Expected pronoun_id %d, got %v", pronoun.ID, updated.PronounID)
	}
	if updated.TemplateID == nil || *updated.TemplateID != tmpl.ID {
		t.Errorf("Expected template_id %d, got %v", tmpl.ID, updated.TemplateID)
	}
}

func TestQuickSearch(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)

	_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice", LastName: "Johnson"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Bob", LastName: "Smith"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	_, err = svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Charlie", LastName: "Johnson"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	results, err := svc.QuickSearch(vaultID, "Johnson")
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if results[0].ID == "" {
		t.Error("Expected non-empty ID")
	}
	if results[0].Name == "" {
		t.Error("Expected non-empty Name")
	}

	results, err = svc.QuickSearch(vaultID, "Bob")
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Name != "Bob Smith" {
		t.Errorf("Expected name 'Bob Smith', got '%s'", results[0].Name)
	}

	results, err = svc.QuickSearch(vaultID, "nonexistent")
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}

	results, err = svc.QuickSearch(vaultID, "")
	if err != nil {
		t.Fatalf("QuickSearch with empty term failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty term, got %d", len(results))
	}
}
