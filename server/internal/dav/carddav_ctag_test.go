package dav

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
)

func TestDAVAddressBookPropfindAdvertisesCTag(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, _ := createDAVHTTPTestContact(t, db, userID, "Ada", "Lovelace")

	requestBody := `<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:" xmlns:CS="http://calendarserver.org/ns/">
  <D:prop>
    <D:resourcetype/>
    <CS:getctag/>
  </D:prop>
</D:propfind>`

	req := httptest.NewRequest("PROPFIND", "/dav/addressbooks/"+userID+"/"+vaultID+"/", strings.NewReader(requestBody))
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(email, password)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207 Multi-Status, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	assertBodyContains(t, body, "http://calendarserver.org/ns/")
	assertBodyContains(t, body, "getctag")

	// A getctag named in the 404 propstat would satisfy the substring
	// assertions above, so check that it actually carries a value.
	if strings.Contains(body, `<getctag xmlns="http://calendarserver.org/ns/"></getctag>`) {
		t.Errorf("getctag was advertised but empty: %s", body)
	}
}

func TestCardDAVCTagChangesWhenRelatedRowChanges(t *testing.T) {
	backend, db, _, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Grace", "Hopper")

	before, err := backend.addressBookCTag(vaultID)
	if err != nil {
		t.Fatalf("addressBookCTag: %v", err)
	}

	// Writing contact information does not touch contacts.updated_at, which is
	// exactly why the CTag cannot be derived from the contacts table alone.
	info := models.ContactInformation{
		ContactID: contact.ID,
		TypeID:    1,
		Data:      "grace@example.com",
	}
	if err := db.Create(&info).Error; err != nil {
		t.Fatalf("create contact information: %v", err)
	}

	after, err := backend.addressBookCTag(vaultID)
	if err != nil {
		t.Fatalf("addressBookCTag: %v", err)
	}
	if before == after {
		t.Errorf("CTag did not change after a related row was written: %s", after)
	}
}

func TestCardDAVCTagStableWithoutChanges(t *testing.T) {
	backend, db, _, vaultID, userID := setupCardDAVTest(t)
	createTestContact(t, db, vaultID, userID, "Alan", "Turing")

	first, err := backend.addressBookCTag(vaultID)
	if err != nil {
		t.Fatalf("addressBookCTag: %v", err)
	}
	second, err := backend.addressBookCTag(vaultID)
	if err != nil {
		t.Fatalf("addressBookCTag: %v", err)
	}
	if first != second {
		t.Errorf("CTag changed with no intervening write: %s then %s", first, second)
	}
	if first == "" {
		t.Error("CTag was empty")
	}
}
