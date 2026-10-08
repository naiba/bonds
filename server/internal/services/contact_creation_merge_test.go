package services

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// Pause only scheduling after a real validation query. Both operations use
// production services; no callback fabricates ownership or database results.
func TestContactCreationOverlappingMerge(t *testing.T) {
	for _, kind := range []string{"information", "goal", "note", "call", "address", "loan", "task", "quick_fact", "group", "label", "introducer", "relationship_owner", "relationship_related", "activity", "vault_task", "company_employee", "legacy_job", "monica", "relationship_update"} {
		for _, mergeFirst := range []bool{true, false} {
			order := "create_first"
			if mergeFirst {
				order = "merge_first"
			}
			t.Run(kind+"/"+order, func(t *testing.T) {
				svc, vaultID, userID, _ := setupContactTest(t)
				target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
				if err != nil {
					t.Fatal(err)
				}
				source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Duplicate"})
				if err != nil {
					t.Fatal(err)
				}
				db := svc.db
				var record any
				ownerColumn := "contact_id"
				var create func() error
				switch kind {
				case "information":
					var email models.ContactInformationType
					if err := db.Where("type = ?", "email").First(&email).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.ContactInformation{}
					create = func() error {
						_, err := NewContactInformationService(db).Create(source.ID, vaultID, dto.CreateContactInformationRequest{TypeID: email.ID, Data: "overlap@example.test"})
						return err
					}
				case "goal":
					record = &models.Goal{}
					create = func() error {
						_, err := NewGoalService(db).Create(source.ID, vaultID, dto.CreateGoalRequest{Name: "Keep in touch"})
						return err
					}
				case "note":
					record = &models.Note{}
					create = func() error {
						_, err := NewNoteService(db).Create(source.ID, vaultID, userID, dto.CreateNoteRequest{Body: "A synthetic note"})
						return err
					}
				case "call":
					record = &models.Call{}
					create = func() error {
						_, err := NewCallService(db).Create(source.ID, vaultID, userID, dto.CreateCallRequest{})
						return err
					}
				case "address":
					record = &models.ContactAddress{}
					create = func() error {
						_, err := NewAddressService(db).Create(source.ID, vaultID, dto.CreateAddressRequest{City: "London"})
						return err
					}
				case "loan":
					record = &models.ContactLoan{}
					ownerColumn = "loaner_id"
					create = func() error {
						_, err := NewLoanService(db).Create(source.ID, vaultID, dto.CreateLoanRequest{Name: "A book"})
						return err
					}
				case "task":
					record = &models.TaskContact{}
					create = func() error {
						_, err := NewTaskService(db).Create(source.ID, vaultID, userID, dto.CreateTaskRequest{Label: "Call"})
						return err
					}
				case "quick_fact":
					var template models.VaultQuickFactsTemplate
					if err := db.Where("vault_id = ?", vaultID).First(&template).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.QuickFact{}
					create = func() error {
						_, err := NewQuickFactService(db).Create(source.ID, vaultID, template.ID, dto.CreateQuickFactRequest{Content: "Hiking"})
						return err
					}
				case "group":
					group := models.Group{VaultID: vaultID, Name: "Friends"}
					if err := db.Create(&group).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.ContactGroup{}
					create = func() error {
						return NewGroupService(db).AddContactToGroup(source.ID, vaultID, dto.AddContactToGroupRequest{GroupID: group.ID})
					}
				case "label":
					label := models.Label{VaultID: vaultID, Name: "Friends"}
					if err := db.Create(&label).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.ContactLabel{}
					create = func() error {
						_, err := NewContactLabelService(db).Add(source.ID, vaultID, dto.AddContactLabelRequest{LabelID: label.ID})
						return err
					}
				case "relationship_owner", "relationship_related":
					neighbor, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Neighbor"})
					if err != nil {
						t.Fatal(err)
					}
					var relationshipType models.RelationshipType
					if err := db.First(&relationshipType).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.Relationship{}
					ownerID, relatedID := source.ID, neighbor.ID
					if kind == "relationship_related" {
						ownerID, relatedID = neighbor.ID, source.ID
						ownerColumn = "related_contact_id"
					}
					create = func() error {
						_, err := NewRelationshipService(db).Create(ownerID, vaultID, userID, dto.CreateRelationshipRequest{RelatedContactID: relatedID, RelationshipTypeID: relationshipType.ID})
						return err
					}
				case "activity":
					var activityType models.ActivityType
					if err := db.Joins("JOIN activity_categories ON activity_categories.id = activity_types.activity_category_id").Where("activity_categories.vault_id = ?", vaultID).First(&activityType).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.ActivityParticipant{}
					create = func() error {
						_, err := NewActivityService(db).Create(vaultID, dto.ActivityUpsertRequest{PrimaryContactID: source.ID, ActivityTypeID: activityType.ID, Title: "A visit"})
						return err
					}
				case "vault_task":
					record = &models.TaskContact{}
					create = func() error {
						_, err := NewVaultTaskService(db).Create(vaultID, userID, dto.CreateVaultTaskRequest{Label: "Call", ContactIDs: []string{source.ID}})
						return err
					}
				case "company_employee", "legacy_job":
					company := models.Company{VaultID: vaultID, Name: "Synthetic company"}
					if err := db.Create(&company).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.ContactCompany{}
					create = func() error {
						if kind == "company_employee" {
							_, err := NewContactJobService(db).AddEmployee(company.ID, vaultID, dto.AddEmployeeRequest{ContactID: source.ID, JobPosition: "Engineer"})
							return err
						}
						_, err := NewContactJobService(db).LegacyUpdate(source.ID, vaultID, userID, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"})
						return err
					}
				case "monica":
					if err := db.Model(&models.Contact{}).Where("id = ?", source.ID).Update("distant_uuid", "synthetic-import").Error; err != nil {
						t.Fatal(err)
					}
					record = &models.Note{}
					data := []byte(`{"version":"1.0-preview.1","account":{"data":[{"type":"contacts","values":[{"uuid":"synthetic-import","properties":{"first_name":"Duplicate"},"data":[{"type":"notes","values":[{"properties":{"body":"Imported note"}}]}]}]}]}}`)
					var validatedJSON any
					if err := json.Unmarshal(data, &validatedJSON); err != nil {
						t.Fatal(err)
					}
					create = func() error {
						result, err := NewMonicaImportService(db, "").Import(vaultID, userID, data)
						if err != nil {
							return err
						}
						if len(result.Errors) > 0 {
							if result.ImportedNotes != 0 {
								return errors.New("failed import counted unsaved notes")
							}
							return ErrContactNotFound
						}
						if result.ImportedNotes != 1 {
							return errors.New("import did not count its note")
						}
						return nil
					}
				case "relationship_update":
					neighbor, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Neighbor"})
					if err != nil {
						t.Fatal(err)
					}
					another, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Another"})
					if err != nil {
						t.Fatal(err)
					}
					var relationshipType models.RelationshipType
					if err := db.First(&relationshipType).Error; err != nil {
						t.Fatal(err)
					}
					relation := models.Relationship{ContactID: neighbor.ID, RelatedContactID: another.ID, RelationshipTypeID: relationshipType.ID}
					if err := db.Create(&relation).Error; err != nil {
						t.Fatal(err)
					}
					record = &models.Relationship{}
					ownerColumn = "related_contact_id"
					create = func() error {
						_, err := NewRelationshipService(db).Update(relation.ID, neighbor.ID, vaultID, dto.UpdateRelationshipRequest{RelatedContactID: source.ID, RelationshipTypeID: relationshipType.ID}, userID)
						return err
					}
				case "introducer":
					record = &models.Contact{}
					ownerColumn = "first_met_through_contact_id"
					create = func() error {
						_, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Introduced", FirstMetThroughContactID: &source.ID})
						return err
					}
				}
				reviewed := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
				if mergeFirst {
					validated, resume := make(chan struct{}), make(chan struct{})
					var paused atomic.Bool
					hook := "merge:pause_creation_after_validation"
					if err := db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
						row, ok := tx.Statement.Dest.(*models.Contact)
						matches := ok && row.ID == source.ID
						if count, counting := tx.Statement.Dest.(*int64); counting && (kind == "activity" || kind == "vault_task" || kind == "company_employee") && tx.Statement.Table == "contacts" && *count > 0 {
							matches = true
						}
						if matches && tx.Error == nil && paused.CompareAndSwap(false, true) {
							close(validated)
							select {
							case <-resume:
							case <-time.After(10 * time.Second):
								tx.AddError(errors.New("test scheduling timeout"))
							}
						}
					}); err != nil {
						t.Fatal(err)
					}
					defer db.Callback().Query().Remove(hook)
					done := make(chan error, 1)
					go func() { done <- create() }()
					select {
					case <-validated:
					case <-time.After(10 * time.Second):
						close(resume)
						t.Fatal("validation was not observed")
					}
					_, mergeErr := svc.MergeContacts(vaultID, userID, reviewed)
					close(resume)
					var createErr error
					select {
					case createErr = <-done:
					case <-time.After(10 * time.Second):
						t.Fatal("create did not finish")
					}
					if mergeErr != nil {
						t.Fatalf("merge: %v", mergeErr)
					}
					if !errors.Is(createErr, ErrContactNotFound) {
						t.Fatalf("stale creation must reject deleted owner, got %v", createErr)
					}
				} else {
					if err := create(); err != nil {
						t.Fatal(err)
					}
					// The newly created row must be included in the user's refreshed review.
					reviewed = reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
					if _, err := svc.MergeContacts(vaultID, userID, reviewed); err != nil {
						t.Fatal(err)
					}
				}
				var sourceCount, targetCount int64
				if err := db.Model(record).Where(ownerColumn+" = ?", source.ID).Count(&sourceCount).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Model(record).Where(ownerColumn+" = ?", target.ID).Count(&targetCount).Error; err != nil {
					t.Fatal(err)
				}
				want := int64(1)
				if mergeFirst {
					want = 0
				}
				if mergeFirst {
					aggregates := map[string]any{"address": &models.Address{}, "loan": &models.Loan{}, "task": &models.ContactTask{}, "vault_task": &models.ContactTask{}, "activity": &models.Activity{}}
					if aggregate, exists := aggregates[kind]; exists {
						var count int64
						if err := db.Model(aggregate).Count(&count).Error; err != nil {
							t.Fatal(err)
						}
						if count != 0 {
							t.Fatalf("rejected creation left %d aggregate rows", count)
						}
					}
				}
				if sourceCount != 0 || targetCount != want {
					t.Fatalf("persisted ownership: source=%d survivor=%d want=%d", sourceCount, targetCount, want)
				}
				if _, err := svc.GetContact(source.ID, userID, vaultID); !errors.Is(err, ErrContactNotFound) {
					t.Fatalf("source still accessible: %v", err)
				}
			})
		}
	}
}

// Files arrive outside the DB transaction. A merge during streaming must reject
// the new attachment and clean its bytes, not leave an inaccessible upload.
func TestContactUploadOverlappingMerge(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	directory := t.TempDir()
	files := NewVaultFileService(svc.db, directory)
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	done := make(chan error, 1)
	go func() {
		_, err := files.Upload(vaultID, source.ID, userID, "document", "synthetic.txt", "text/plain", 1, reader)
		done <- err
	}()
	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, req); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrContactNotFound) {
			t.Fatalf("upload should reject deleted owner: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("upload did not finish")
	}
	var count int64
	if err := svc.db.Model(&models.File{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(entries) != 0 {
		t.Fatalf("rejected upload left %d rows and %d files", count, len(entries))
	}
}

func TestContactCreationHoldsOwnerUntilCommit(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	reviewed := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	inserting, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "merge:pause_goal_insertion"
	if err := svc.db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
		row, ok := tx.Statement.Dest.(*models.Goal)
		if ok && row.ContactID == source.ID && paused.CompareAndSwap(false, true) {
			close(inserting)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("test scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Create().Remove(hook)
	created := make(chan error, 1)
	go func() {
		_, err := NewGoalService(svc.db).Create(source.ID, vaultID, dto.CreateGoalRequest{Name: "Keep in touch"})
		created <- err
	}()
	select {
	case <-inserting:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("insertion was not observed")
	}
	merged := make(chan error, 1)
	go func() { _, err := svc.MergeContacts(vaultID, userID, reviewed); merged <- err }()
	var premature error
	finished := false
	select {
	case premature = <-merged:
		finished = true
	case <-time.After(100 * time.Millisecond):
	}
	close(resume)
	select {
	case err := <-created:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("creation did not finish")
	}
	if finished {
		t.Fatalf("merge passed the owner's creation transaction before commit: %v", premature)
	}
	select {
	case err := <-merged:
		if !errors.Is(err, ErrContactMergeReviewChanged) {
			t.Fatalf("new goal must require a refreshed review: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("merge did not finish")
	}
	goals, err := NewGoalService(svc.db).List(source.ID, vaultID)
	if err != nil || len(goals) != 1 {
		t.Fatalf("successful creation lost before refreshed merge: %+v, %v", goals, err)
	}
	reviewed = reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	if _, err := svc.MergeContacts(vaultID, userID, reviewed); err != nil {
		t.Fatal(err)
	}
	moved, err := NewGoalService(svc.db).List(target.ID, vaultID)
	if err != nil || len(moved) != 1 || moved[0].ID != goals[0].ID || moved[0].Name != "Keep in touch" {
		t.Fatalf("goal identity/data lost: %+v, %v", moved, err)
	}
}

func TestCSVImportReportsMergeDuringChildCreation(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "merge:pause_csv_information_type"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*models.ContactInformationType); ok && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(loaded)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("test scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(hook)
	type outcome struct {
		response *dto.CSVImportResponse
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		r, e := NewCSVImportService(svc.db).Import(vaultID, userID, []byte("Name,Email,Notes,City\nDuplicate,synthetic@example.test,Imported note,London\n"), dto.CSVColumnMapping{FirstName: "Name", Email: "Email", Notes: "Notes", AddressCity: "City"})
		done <- outcome{r, e}
	}()
	select {
	case <-loaded:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("CSV child creation not observed")
	}
	var source models.Contact
	if err := svc.db.Where("vault_id = ? AND first_name = ?", vaultID, "Duplicate").First(&source).Error; err != nil {
		close(resume)
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	_, mergeErr := svc.MergeContacts(vaultID, userID, req)
	close(resume)
	if mergeErr != nil {
		t.Fatal(mergeErr)
	}
	var result outcome
	select {
	case result = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("CSV import did not finish")
	}
	if result.err != nil {
		t.Fatal(result.err)
	}
	if result.response.ImportedContacts != 1 || len(result.response.Errors) != 3 {
		t.Fatalf("partial CSV import must report rejected email, note and address: %+v", result.response)
	}
	for _, record := range []any{&models.ContactInformation{}, &models.Note{}, &models.ContactAddress{}} {
		var count int64
		if err := svc.db.Model(record).Where("contact_id = ?", source.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("CSV import left %d hidden %T rows", count, record)
		}
	}
	var addresses int64
	if err := svc.db.Model(&models.Address{}).Count(&addresses).Error; err != nil {
		t.Fatal(err)
	}
	if addresses != 0 {
		t.Fatalf("CSV import left %d orphan addresses", addresses)
	}
}

func TestContactCreationIncludesFeedBeforeMerge(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL concurrent feed transaction visibility")
	}
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	noteService := NewNoteService(svc.db)
	noteService.SetFeedRecorder(NewFeedRecorder(svc.db))
	recording, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "merge:pause_note_feed"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		row, ok := tx.Statement.Dest.(*models.Contact)
		if ok && row.ID == source.ID && tx.Statement.Unscoped && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(recording)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("test scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(hook)
	created := make(chan error, 1)
	go func() {
		_, err := noteService.Create(source.ID, vaultID, userID, dto.CreateNoteRequest{Body: "Audited note"})
		created <- err
	}()
	select {
	case <-recording:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("feed record was not observed")
	}
	// Preview while the feed is pending: a committed note must not expose a
	// window in which a matching merge can leave its event on the old owner.
	req = reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	merged := make(chan error, 1)
	go func() { _, err := svc.MergeContacts(vaultID, userID, req); merged <- err }()
	completed := false
	var mergeErr error
	select {
	case mergeErr = <-merged:
		completed = true
	case <-time.After(100 * time.Millisecond):
	}
	close(resume)
	select {
	case err := <-created:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("note creation did not finish")
	}
	if !completed {
		select {
		case mergeErr = <-merged:
		case <-time.After(10 * time.Second):
			t.Fatal("merge did not finish")
		}
	}
	if mergeErr != nil {
		if !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
			t.Fatal(mergeErr)
		}
		if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
			t.Fatal(err)
		}
	}
	var events []models.ContactFeedItem
	if err := svc.db.Where("action = ?", ActionNoteCreated).Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ContactID != target.ID {
		t.Fatalf("creation feed was left behind: %+v", events)
	}
	projected, err := NewFeedService(svc.db).sourceAvailable(events[0])
	if err != nil {
		t.Fatal(err)
	}
	if projected == nil || !projected.Available {
		t.Fatalf("created note's feed source is unavailable: %+v", projected)
	}
}
