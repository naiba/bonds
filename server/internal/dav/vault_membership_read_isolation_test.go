package dav

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/emersion/go-webdav"
	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/models"
)

const userWithoutVaultMembership = "user-without-vault-membership"

func requireDAVForbidden(t *testing.T, operation string, err error) {
	t.Helper()
	// go-webdav keeps its HTTP error type internal, so compare the rendering.
	forbidden := webdav.NewHTTPError(http.StatusForbidden, nil).Error()
	if err == nil || !strings.HasPrefix(err.Error(), forbidden) {
		t.Errorf("%s by a user outside the vault: error = %v, want HTTP 403", operation, err)
	}
}

// A DAV path is built by the client, so another user's vault ID or object
// UUID can be put into it freely. Every read has to check vault membership;
// before this test the suite stayed green with that check removed.
func TestCalDAVReadsRequireVaultMembership(t *testing.T) {
	backend, db, ownerCtx, vaultID, ownerID := setupCalDAVTest(t)
	contact := createTestContact(t, db, vaultID, ownerID, "Private", "Person")
	dateUID, taskUID := uuid.New().String(), uuid.New().String()
	day, month, year := 2, 1, 2000
	if err := db.Create(&models.ContactImportantDate{
		ContactID: contact.ID, UUID: &dateUID, Label: "Private date", Day: &day, Month: &month, Year: &year,
	}).Error; err != nil {
		t.Fatalf("create important date: %v", err)
	}
	if err := db.Create(&models.ContactTask{VaultID: vaultID, UUID: &taskUID, Label: "Private task", AuthorName: "owner"}).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	ownerCalendar := "/dav/calendars/" + ownerID + "/" + vaultID + "/"
	if _, err := backend.GetCalendarObject(ownerCtx, ownerCalendar+dateUID+".ics", nil); err != nil {
		t.Fatalf("test setup: owner cannot read the important date: %v", err)
	}
	if _, err := backend.GetCalendarObject(ownerCtx, ownerCalendar+taskUID+".ics", nil); err != nil {
		t.Fatalf("test setup: owner cannot read the task: %v", err)
	}
	ownerObjects, err := backend.ListCalendarObjects(ownerCtx, ownerCalendar, nil)
	if err != nil || len(ownerObjects) == 0 {
		t.Fatalf("test setup: owner lists %d calendar objects, err = %v", len(ownerObjects), err)
	}

	outsiderCtx := WithUserID(context.Background(), userWithoutVaultMembership)
	outsiderCalendar := "/dav/calendars/" + userWithoutVaultMembership + "/" + vaultID + "/"

	_, err = backend.GetCalendar(outsiderCtx, outsiderCalendar)
	requireDAVForbidden(t, "GetCalendar", err)
	_, err = backend.GetCalendarObject(outsiderCtx, outsiderCalendar+dateUID+".ics", nil)
	requireDAVForbidden(t, "GetCalendarObject (important date)", err)
	_, err = backend.GetCalendarObject(outsiderCtx, outsiderCalendar+taskUID+".ics", nil)
	requireDAVForbidden(t, "GetCalendarObject (task)", err)
	objects, err := backend.ListCalendarObjects(outsiderCtx, outsiderCalendar, nil)
	requireDAVForbidden(t, "ListCalendarObjects", err)
	if len(objects) != 0 {
		t.Errorf("ListCalendarObjects returned %d objects to a user outside the vault", len(objects))
	}
}

func TestCardDAVReadsRequireVaultMembership(t *testing.T) {
	backend, db, ownerCtx, vaultID, ownerID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, ownerID, "Private", "Person")

	ownerAddressBook := "/dav/addressbooks/" + ownerID + "/" + vaultID + "/"
	if _, err := backend.GetAddressObject(ownerCtx, ownerAddressBook+contact.ID+".vcf", nil); err != nil {
		t.Fatalf("test setup: owner cannot read the contact: %v", err)
	}
	ownerObjects, err := backend.ListAddressObjects(ownerCtx, ownerAddressBook, nil)
	if err != nil || len(ownerObjects) != 1 {
		t.Fatalf("test setup: owner lists %d contacts, err = %v", len(ownerObjects), err)
	}

	outsiderCtx := WithUserID(context.Background(), userWithoutVaultMembership)
	outsiderAddressBook := "/dav/addressbooks/" + userWithoutVaultMembership + "/" + vaultID + "/"

	_, err = backend.GetAddressObject(outsiderCtx, outsiderAddressBook+contact.ID+".vcf", nil)
	requireDAVForbidden(t, "GetAddressObject", err)
	objects, err := backend.ListAddressObjects(outsiderCtx, outsiderAddressBook, nil)
	requireDAVForbidden(t, "ListAddressObjects", err)
	if len(objects) != 0 {
		t.Errorf("ListAddressObjects returned %d contacts to a user outside the vault", len(objects))
	}
}
