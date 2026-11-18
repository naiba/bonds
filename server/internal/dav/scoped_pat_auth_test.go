package dav

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func createDAVTestPAT(t *testing.T, db *gorm.DB, user models.User, rawToken, accountID, scopes string) {
	t.Helper()
	h := sha256.Sum256([]byte(rawToken))
	if err := db.Create(&models.PersonalAccessToken{
		UserID: user.ID, AccountID: accountID, Name: "dav-test-pat",
		TokenHash: hex.EncodeToString(h[:]), TokenHint: "bonds_dav...pat", Scopes: scopes,
	}).Error; err != nil {
		t.Fatalf("create PAT: %v", err)
	}
}

func serveDAVWithBasicAuth(e *echo.Echo, method, path, body, email, password string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.SetBasicAuth(email, password)
	switch method {
	case "PROPFIND":
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "application/xml")
	case http.MethodPut:
		req.Header.Set("Content-Type", "text/vcard")
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// GHSA-hc8x-cmcm-2jwf: a calendar:read PAT (meant only for the .ics feed URL)
// used to get full CardDAV/CalDAV read-write access through Basic Auth.
func TestDAVRejectsScopedPATForEveryMethod(t *testing.T) {
	_, db, _, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Secret", "Person")
	var user models.User
	db.First(&user, "id = ?", userID)

	scopedToken := "bonds_scopedcalendarfeedtoken"
	createDAVTestPAT(t, db, user, scopedToken, user.AccountID, "calendar:read")
	fullToken := "bonds_fullaccessdavtoken"
	createDAVTestPAT(t, db, user, fullToken, user.AccountID, "")

	e := echo.New()
	SetupDAVRoutes(e, db, t.TempDir())
	book := "/dav/addressbooks/" + userID + "/" + vaultID + "/"
	contactPath := book + contact.ID + ".vcf"
	propfind := `<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:getetag/></d:prop></d:propfind>`
	newCard := "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:scoped-injected\r\nFN:Injected\r\nN:Injected;;;;\r\nEND:VCARD\r\n"

	attempts := []struct{ method, path, body string }{
		{http.MethodGet, contactPath, ""},
		{"PROPFIND", book, propfind},
		{"PROPFIND", "/dav/calendars/" + userID + "/" + vaultID + "/", propfind},
		{http.MethodPut, book + "scoped-injected.vcf", newCard},
		{http.MethodDelete, contactPath, ""},
	}
	for _, attempt := range attempts {
		rec := serveDAVWithBasicAuth(e, attempt.method, attempt.path, attempt.body, user.Email, scopedToken)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with scoped PAT: expected 401, got %d", attempt.method, attempt.path, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "Secret") {
			t.Errorf("%s %s with scoped PAT leaked contact data", attempt.method, attempt.path)
		}
	}
	var remaining int64
	db.Model(&models.Contact{}).Where("id = ?", contact.ID).Count(&remaining)
	if remaining != 1 {
		t.Fatalf("scoped PAT DELETE removed the contact")
	}
	var injected int64
	db.Model(&models.Contact{}).Where("vault_id = ? AND first_name = ?", vaultID, "Injected").Count(&injected)
	if injected != 0 {
		t.Fatalf("scoped PAT PUT created a contact")
	}

	// A full-access PAT is still the supported DAV credential.
	rec := serveDAVWithBasicAuth(e, http.MethodGet, contactPath, "", user.Email, fullToken)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Secret") {
		t.Fatalf("full-access PAT GET: expected 200 with vCard, got %d: %s", rec.Code, rec.Body.String())
	}
}

// A PAT is bound to the account it was created in; once the user leaves that
// account the token must stop working, matching REST PAT authentication.
func TestDAVRejectsPATAfterAccountMembershipRemoved(t *testing.T) {
	_, db, _, _, userID := setupCardDAVTest(t)
	var user models.User
	db.First(&user, "id = ?", userID)

	foreignAccount := models.Account{}
	if err := db.Create(&foreignAccount).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	membership := models.AccountMembership{AccountID: foreignAccount.ID, UserID: user.ID}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}
	rawToken := "bonds_foreignaccountpat"
	createDAVTestPAT(t, db, user, rawToken, foreignAccount.ID, "")

	mw := BasicAuthMiddleware(db)
	var gotAccountID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccountID = AccountIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	serve := func() int {
		req := httptest.NewRequest("PROPFIND", "/dav/", nil)
		req.SetBasicAuth(user.Email, rawToken)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := serve(); code != http.StatusOK {
		t.Fatalf("PAT with live membership: expected 200, got %d", code)
	}
	if gotAccountID != foreignAccount.ID {
		t.Fatalf("expected DAV account context %q from PAT, got %q", foreignAccount.ID, gotAccountID)
	}

	if err := db.Delete(&membership).Error; err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	if code := serve(); code != http.StatusUnauthorized {
		t.Fatalf("PAT after membership removal: expected 401, got %d", code)
	}
}

// ContactInformationType rows belong to the vault's account. A DAV session
// whose account context differs from the vault (shared cross-account vault)
// must still resolve phone types from the vault's account, otherwise TEL
// values are silently dropped or bound to another account's types.
func TestPutAddressObjectResolvesContactInfoTypesFromVaultAccount(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	var vault models.Vault
	db.First(&vault, "id = ?", vaultID)

	foreignCtx := WithAccountID(context.Background(), "some-other-account")
	foreignCtx = WithUserID(foreignCtx, UserIDFromContext(ctx))

	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetValue(vcard.FieldFormattedName, "Phone Owner")
	card.SetName(&vcard.Name{GivenName: "Phone", FamilyName: "Owner"})
	card.SetValue(vcard.FieldTelephone, "+15551234567")

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/phone-owner.vcf"
	if _, err := backend.PutAddressObject(foreignCtx, path, card, nil); err != nil {
		t.Fatalf("PutAddressObject failed: %v", err)
	}

	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Phone").First(&contact).Error; err != nil {
		t.Fatalf("contact not created: %v", err)
	}
	var infos []models.ContactInformation
	db.Where("contact_id = ?", contact.ID).Find(&infos)
	if len(infos) != 1 || infos[0].Data != "+15551234567" {
		t.Fatalf("expected the TEL value to be stored once, got %+v", infos)
	}
	var infoType models.ContactInformationType
	db.First(&infoType, infos[0].TypeID)
	if infoType.AccountID != vault.AccountID || infoType.Type == nil || *infoType.Type != "phone" {
		t.Fatalf("expected phone type from vault account %q, got account=%q type=%v", vault.AccountID, infoType.AccountID, infoType.Type)
	}
}
