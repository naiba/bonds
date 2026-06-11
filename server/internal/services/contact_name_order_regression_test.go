package services

import (
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type nameOrderRegressionContext struct {
	db      *gorm.DB
	userID  string
	vaultID string
	contact *dto.ContactResponse
}

func setupNameOrderRegressionTest(t *testing.T, email string) *nameOrderRegressionContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Format",
		LastName:  "Tester",
		Email:     email,
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Formatting Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	override := "%last_name%, %first_name% {nickname? (%nickname%)}"
	if err := db.Model(&models.Vault{}).Where("id = ?", vault.ID).Update("name_order", override).Error; err != nil {
		t.Fatalf("Update vault name_order failed: %v", err)
	}

	contact, err := NewContactService(db).CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{
		FirstName: "Alice",
		LastName:  "Zephyr",
		Nickname:  "Ace",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return &nameOrderRegressionContext{db: db, userID: resp.User.ID, vaultID: vault.ID, contact: contact}
}

func TestVaultAwareTaskAssigneeNamesUseVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-task@example.com")

	task, err := NewVaultTaskService(ctx.db).Create(ctx.vaultID, ctx.userID, dto.CreateVaultTaskRequest{
		Label:      "Call Alice",
		ContactIDs: []string{ctx.contact.ID},
	})
	if err != nil {
		t.Fatalf("Create vault task failed: %v", err)
	}
	if len(task.Contacts) != 1 {
		t.Fatalf("expected 1 assignee, got %d", len(task.Contacts))
	}
	if task.Contacts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("vault task assignee name = %q, want %q", task.Contacts[0].Name, "Zephyr, Alice (Ace)")
	}

	contactTask, err := NewTaskService(ctx.db).Create(ctx.contact.ID, ctx.vaultID, ctx.userID, dto.CreateTaskRequest{Label: "Contact task"})
	if err != nil {
		t.Fatalf("Create contact task failed: %v", err)
	}
	if len(contactTask.Contacts) != 1 {
		t.Fatalf("expected 1 contact task assignee, got %d", len(contactTask.Contacts))
	}
	if contactTask.Contacts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("contact task assignee name = %q, want %q", contactTask.Contacts[0].Name, "Zephyr, Alice (Ace)")
	}
}

func TestFeedContactNameUsesVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-feed@example.com")
	desc := "Created note"
	if err := ctx.db.Create(&models.ContactFeedItem{
		ContactID:   ctx.contact.ID,
		AuthorID:    &ctx.userID,
		Action:      ActionNoteCreated,
		Description: &desc,
	}).Error; err != nil {
		t.Fatalf("Create feed item failed: %v", err)
	}

	items, _, err := NewFeedService(ctx.db).GetFeed(ctx.vaultID, 1, 15, ctx.userID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 feed item, got %d", len(items))
	}
	if items[0].ContactName != "Zephyr, Alice (Ace)" {
		t.Fatalf("feed contact_name = %q, want %q", items[0].ContactName, "Zephyr, Alice (Ace)")
	}
}

func TestRelationshipNamesUseContactVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-relationship@example.com")
	relatedVault, err := NewVaultService(ctx.db).CreateVault(loadAccountID(t, ctx.db, ctx.userID), ctx.userID, dto.CreateVaultRequest{Name: "Related Vault"}, "en")
	if err != nil {
		t.Fatalf("Create related vault failed: %v", err)
	}
	relatedOverride := "%last_name% / %first_name%"
	if err := ctx.db.Model(&models.Vault{}).Where("id = ?", relatedVault.ID).Update("name_order", relatedOverride).Error; err != nil {
		t.Fatalf("Update related vault name_order failed: %v", err)
	}
	related, err := NewContactService(ctx.db).CreateContact(relatedVault.ID, ctx.userID, dto.CreateContactRequest{FirstName: "Bob", LastName: "Yellow"})
	if err != nil {
		t.Fatalf("Create related contact failed: %v", err)
	}
	relType := createNameOrderRelationshipType(t, ctx.db, loadAccountID(t, ctx.db, ctx.userID))

	rel, err := NewRelationshipService(ctx.db).Create(ctx.contact.ID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
		RelatedContactID:   related.ID,
		RelationshipTypeID: relType.ID,
	})
	if err != nil {
		t.Fatalf("Create relationship failed: %v", err)
	}
	if rel.RelatedContactName != "Yellow / Bob" {
		t.Fatalf("related_contact_name = %q, want %q", rel.RelatedContactName, "Yellow / Bob")
	}

	picker, err := NewRelationshipService(ctx.db).ListContactsAcrossVaults(ctx.userID)
	if err != nil {
		t.Fatalf("ListContactsAcrossVaults failed: %v", err)
	}
	foundRelated := false
	for _, item := range picker {
		if item.ContactID == related.ID {
			foundRelated = true
			if item.ContactName != "Yellow / Bob" {
				t.Fatalf("cross-vault picker name = %q, want %q", item.ContactName, "Yellow / Bob")
			}
		}
	}
	if !foundRelated {
		t.Fatal("related contact missing from cross-vault picker")
	}

	graph, err := NewRelationshipService(ctx.db).GetContactGraph(ctx.contact.ID, ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("GetContactGraph failed: %v", err)
	}
	wantLabels := map[string]string{ctx.contact.ID: "Zephyr, Alice (Ace)", related.ID: "Yellow / Bob"}
	for _, node := range graph.Nodes {
		if want, ok := wantLabels[node.ID]; ok && node.Label != want {
			t.Fatalf("graph node %s label = %q, want %q", node.ID, node.Label, want)
		}
	}
}

func TestContactQuickSearchUsesVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-quick-search@example.com")

	items, err := NewContactService(ctx.db).QuickSearch(ctx.vaultID, "Alice", ctx.userID)
	if err != nil {
		t.Fatalf("QuickSearch failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 quick search result, got %d", len(items))
	}
	if items[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("quick search name = %q, want %q", items[0].Name, "Zephyr, Alice (Ace)")
	}
}

func TestFirstMetThroughContactNameUsesVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-met-through@example.com")

	introducer, err := NewContactService(ctx.db).CreateContact(ctx.vaultID, ctx.userID, dto.CreateContactRequest{
		FirstName: "Alice",
		LastName:  "Zephyr",
		Nickname:  "Ace",
	})
	if err != nil {
		t.Fatalf("Create introducer failed: %v", err)
	}
	firstMetAt := time.Date(2026, 3, 12, 9, 15, 0, 0, time.UTC)
	contact, err := NewContactService(ctx.db).CreateContact(ctx.vaultID, ctx.userID, dto.CreateContactRequest{
		FirstName:                "Met",
		FirstMetAt:               &firstMetAt,
		FirstMetThroughContactID: &introducer.ID,
	})
	if err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	got, err := NewContactService(ctx.db).GetContact(contact.ID, ctx.userID, ctx.vaultID)
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if got.FirstMetThroughContact == nil {
		t.Fatal("expected first_met_through_contact to be populated")
	}
	if got.FirstMetThroughContact.Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("first_met_through_contact.name = %q, want %q", got.FirstMetThroughContact.Name, "Zephyr, Alice (Ace)")
	}

	listed, _, err := NewContactService(ctx.db).ListContacts(ctx.vaultID, ctx.userID, 1, 15, "", "first_name", "")
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}
	var matched *dto.ContactResponse
	for i := range listed {
		if listed[i].ID == contact.ID {
			matched = &listed[i]
			break
		}
	}
	if matched == nil {
		t.Fatal("expected listed contact to include the met-through contact")
	}
	if matched.FirstMetThroughContact == nil {
		t.Fatal("expected listed contact to include first_met_through_contact")
	}
	if matched.FirstMetThroughContact.Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("listed first_met_through_contact.name = %q, want %q", matched.FirstMetThroughContact.Name, "Zephyr, Alice (Ace)")
	}
}

func TestSearchServiceHydratesContactNameWithVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-search@example.com")
	engine, err := search.NewBleveEngine(t.TempDir() + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()
	searchSvc := NewSearchServiceWithDB(ctx.db, engine)

	var contact models.Contact
	if err := ctx.db.First(&contact, "id = ?", ctx.contact.ID).Error; err != nil {
		t.Fatalf("load contact failed: %v", err)
	}
	if err := searchSvc.IndexContact(&contact); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}

	result, err := searchSvc.SearchForUser(ctx.vaultID, ctx.userID, "Alice", 1, 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Contacts) != 1 {
		t.Fatalf("expected 1 contact hit, got %d", len(result.Contacts))
	}
	if result.Contacts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("search contact name = %q, want %q", result.Contacts[0].Name, "Zephyr, Alice (Ace)")
	}
}

func TestSearchServiceDropsContactHitsOutsideRequestedVault(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-search-scope@example.com")
	otherVault, err := NewVaultService(ctx.db).CreateVault(loadAccountID(t, ctx.db, ctx.userID), ctx.userID, dto.CreateVaultRequest{Name: "Other Vault"}, "en")
	if err != nil {
		t.Fatalf("Create other vault failed: %v", err)
	}
	otherContact, err := NewContactService(ctx.db).CreateContact(otherVault.ID, ctx.userID, dto.CreateContactRequest{FirstName: "Eve", LastName: "Outside"})
	if err != nil {
		t.Fatalf("Create other contact failed: %v", err)
	}
	hiddenContact, err := NewContactService(ctx.db).CreateContact(ctx.vaultID, ctx.userID, dto.CreateContactRequest{FirstName: "Hidden", LastName: "Shadow"})
	if err != nil {
		t.Fatalf("Create hidden contact failed: %v", err)
	}
	if err := ctx.db.Model(&models.Contact{}).Where("id = ?", hiddenContact.ID).Update("listed", false).Error; err != nil {
		t.Fatalf("Hide contact failed: %v", err)
	}

	engine := &fixedContactSearchEngine{contacts: []search.SearchResult{
		{ID: ctx.contact.ID, Type: "contact", Name: "stale in-vault name", Score: 1},
		{ID: otherContact.ID, Type: "contact", Name: "stale cross-vault name", Score: 1},
		{ID: hiddenContact.ID, Type: "contact", Name: "stale hidden name", Score: 1},
	}}
	result, err := NewSearchServiceWithDB(ctx.db, engine).SearchForUser(ctx.vaultID, ctx.userID, "Alice", 1, 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Contacts) != 1 {
		t.Fatalf("expected only the requested vault contact hit, got %+v", result.Contacts)
	}
	if result.Contacts[0].ID != ctx.contact.ID {
		t.Fatalf("search returned contact ID %q, want %q", result.Contacts[0].ID, ctx.contact.ID)
	}
	if result.Contacts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("search contact name = %q, want %q", result.Contacts[0].Name, "Zephyr, Alice (Ace)")
	}
	if result.Total != 1 {
		t.Fatalf("search total = %d, want 1 after filtering stale and hidden hits", result.Total)
	}
	if result.Contacts[0].ID == otherContact.ID {
		t.Fatalf("search returned cross-vault contact %q", otherContact.ID)
	}
	if result.Contacts[0].ID == hiddenContact.ID {
		t.Fatalf("search returned hidden contact %q", hiddenContact.ID)
	}
}

type fixedContactSearchEngine struct {
	contacts []search.SearchResult
}

func (e *fixedContactSearchEngine) IndexContact(id, vaultID, firstName, lastName, nickname, jobPosition string) error {
	return nil
}

func (e *fixedContactSearchEngine) IndexNote(id string, vaultID, contactID, title, body string) error {
	return nil
}

func (e *fixedContactSearchEngine) DeleteDocument(id string) error {
	return nil
}

func (e *fixedContactSearchEngine) Search(vaultID, query string, limit, offset int) (*search.SearchResponse, error) {
	contacts := append([]search.SearchResult(nil), e.contacts...)
	return &search.SearchResponse{Contacts: contacts, Notes: []search.SearchResult{}, Total: len(contacts)}, nil
}

func (e *fixedContactSearchEngine) Rebuild() error {
	return nil
}

func (e *fixedContactSearchEngine) Close() error {
	return nil
}

func TestCatchUpPromptsUseVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-catch-up@example.com")
	lastTalkedTo := time.Now().AddDate(0, 0, -45)
	frequencyDays := 30
	if err := ctx.db.Model(&models.Contact{}).Where("id = ?", ctx.contact.ID).Updates(map[string]interface{}{
		"last_talked_to":               lastTalkedTo,
		"stay_in_touch_frequency_days": frequencyDays,
		"stay_in_touch_trigger_date":   lastTalkedTo.AddDate(0, 0, frequencyDays),
	}).Error; err != nil {
		t.Fatalf("Update contact catch-up fields failed: %v", err)
	}

	prompts, err := NewContactService(ctx.db).ListCatchUpPrompts(ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("ListCatchUpPrompts failed: %v", err)
	}
	if len(prompts) != 1 {
		t.Fatalf("expected 1 catch-up prompt, got %d: %+v", len(prompts), prompts)
	}
	if prompts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("catch-up prompt name = %q, want %q", prompts[0].Name, "Zephyr, Alice (Ace)")
	}
}

func TestCompanyEmployeeBriefsUseVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-company@example.com")
	company := models.Company{VaultID: ctx.vaultID, Name: "Acme"}
	if err := ctx.db.Create(&company).Error; err != nil {
		t.Fatalf("Create company failed: %v", err)
	}
	position := "Engineer"
	job := models.ContactCompany{ContactID: ctx.contact.ID, CompanyID: company.ID, JobPosition: &position}
	if err := ctx.db.Create(&job).Error; err != nil {
		t.Fatalf("Create contact company failed: %v", err)
	}

	listed, err := NewCompanyService(ctx.db).List(ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("List companies failed: %v", err)
	}
	if len(listed) != 1 || len(listed[0].Contacts) != 1 {
		t.Fatalf("expected one listed company employee, got %+v", listed)
	}
	if listed[0].Contacts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("listed employee name = %q, want %q", listed[0].Contacts[0].Name, "Zephyr, Alice (Ace)")
	}

	got, err := NewCompanyService(ctx.db).Get(company.ID, ctx.vaultID, ctx.userID)
	if err != nil {
		t.Fatalf("Get company failed: %v", err)
	}
	if len(got.Contacts) != 1 {
		t.Fatalf("expected one company employee, got %d", len(got.Contacts))
	}
	if got.Contacts[0].Name != "Zephyr, Alice (Ace)" {
		t.Fatalf("company employee name = %q, want %q", got.Contacts[0].Name, "Zephyr, Alice (Ace)")
	}
}

func TestReminderSchedulerPayloadUsesVaultNameOrder(t *testing.T) {
	ctx := setupNameOrderRegressionTest(t, "name-order-reminder@example.com")
	reminder := models.ContactReminder{ContactID: ctx.contact.ID, Label: "Check in", Type: "one_time"}
	if err := ctx.db.Create(&reminder).Error; err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}
	now := time.Now()
	channel := models.UserNotificationChannel{UserID: &ctx.userID, Type: "email", Content: "name-order-reminder@example.com", Active: true, VerifiedAt: &now}
	if err := ctx.db.Create(&channel).Error; err != nil {
		t.Fatalf("Create channel failed: %v", err)
	}
	if err := ctx.db.Create(&models.ContactReminderScheduled{
		UserNotificationChannelID: channel.ID,
		ContactReminderID:         reminder.ID,
		ScheduledAt:               time.Now().Add(-time.Minute),
	}).Error; err != nil {
		t.Fatalf("Create scheduled reminder failed: %v", err)
	}

	mailer := &recordingMailer{}
	NewReminderSchedulerService(ctx.db, mailer, nil).ProcessDueReminders()
	got := mailer.last(t)
	if !strings.Contains(got.body, "Zephyr, Alice (Ace)") {
		t.Fatalf("reminder body %q does not contain formatted contact name %q", got.body, "Zephyr, Alice (Ace)")
	}
}

func loadAccountID(t *testing.T, db *gorm.DB, userID string) string {
	t.Helper()
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatalf("load user failed: %v", err)
	}
	return user.AccountID
}

func createNameOrderRelationshipType(t *testing.T, db *gorm.DB, accountID string) models.RelationshipType {
	t.Helper()
	name := "Knows"
	groupName := "Custom"
	typeGroup := models.RelationshipGroupType{AccountID: accountID, Name: &groupName}
	if err := db.Create(&typeGroup).Error; err != nil {
		t.Fatalf("create relationship group type failed: %v", err)
	}
	relType := models.RelationshipType{RelationshipGroupTypeID: typeGroup.ID, Name: &name}
	if err := db.Create(&relType).Error; err != nil {
		t.Fatalf("create relationship type failed: %v", err)
	}
	return relType
}
