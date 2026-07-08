package dav

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	appMiddleware "github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func TestDAVOptionsReturnsDiscoveryHeadersWithoutAuth(t *testing.T) {
	e := setupDAVHTTPTest(t)

	req := httptest.NewRequest(http.MethodOptions, "/dav/addressbooks/test-user/", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "" {
		t.Errorf("expected no Basic Auth challenge for OPTIONS, got %q", got)
	}
	assertHeaderContains(t, rec.Header(), "DAV", "addressbook")
	assertHeaderContains(t, rec.Header(), "Allow", "OPTIONS")
	assertHeaderContains(t, rec.Header(), "Allow", "PROPFIND")
	assertHeaderContains(t, rec.Header(), "Allow", "REPORT")
}

func TestDAVPreflightReturnsCORSAndDiscoveryHeaders(t *testing.T) {
	e := setupDAVHTTPTest(t)

	req := httptest.NewRequest(http.MethodOptions, "/dav/calendars/test-user/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "REPORT")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Depth,Content-Type")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "" {
		t.Errorf("expected no Basic Auth challenge for DAV preflight, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin for dev frontend, got %q", got)
	}
	assertHeaderContains(t, rec.Header(), "Access-Control-Allow-Methods", "PROPFIND")
	assertHeaderContains(t, rec.Header(), "Access-Control-Allow-Methods", "REPORT")
	assertHeaderContains(t, rec.Header(), "Access-Control-Allow-Headers", "Authorization")
	assertHeaderContains(t, rec.Header(), "Access-Control-Allow-Headers", "Depth")
	assertHeaderContains(t, rec.Header(), "DAV", "calendar-access")
	assertHeaderContains(t, rec.Header(), "Allow", "REPORT")
}

func TestDAVPrincipalOptionsReturnsCombinedDiscoveryHeadersWithoutAuth(t *testing.T) {
	e := setupDAVHTTPTest(t)

	req := httptest.NewRequest(http.MethodOptions, "/dav/principals/test-user/", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "" {
		t.Errorf("expected no Basic Auth challenge for principal OPTIONS, got %q", got)
	}
	assertHeaderContains(t, rec.Header(), "DAV", "addressbook")
	assertHeaderContains(t, rec.Header(), "DAV", "calendar-access")
	assertHeaderContains(t, rec.Header(), "Allow", "PROPFIND")
	assertHeaderContains(t, rec.Header(), "Allow", "REPORT")
}

func TestDAVRealMethodsStillRequireBasicAuth(t *testing.T) {
	e := setupDAVHTTPTest(t)

	req := httptest.NewRequest("PROPFIND", "/dav/addressbooks/test-user/", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated PROPFIND, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Error("expected Basic Auth challenge for unauthenticated PROPFIND")
	}
}

func TestDAVPrincipalPropfindAdvertisesCardDAVAndCalDAVHomeSets(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	requestBody := `<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CR="urn:ietf:params:xml:ns:carddav">
  <D:prop>
    <D:resourcetype/>
    <D:current-user-principal/>
    <C:calendar-home-set/>
    <CR:addressbook-home-set/>
  </D:prop>
</D:propfind>`

	req := httptest.NewRequest("PROPFIND", "/dav/principals/"+userID+"/", strings.NewReader(requestBody))
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(email, password)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207 Multi-Status, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	assertBodyContains(t, body, "/dav/principals/"+userID+"/")
	assertBodyContains(t, body, "/dav/addressbooks/"+userID+"/")
	assertBodyContains(t, body, "/dav/calendars/"+userID+"/")
	assertBodyContains(t, body, "addressbook-home-set")
	assertBodyContains(t, body, "calendar-home-set")
	assertBodyContains(t, body, "resourcetype")
	assertBodyContains(t, body, "principal")
}

func TestDAVGetAddressObject_SetsUTF8ContentType(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Róisín", "Ní")

	req := httptest.NewRequest(http.MethodGet, "/dav/addressbooks/"+userID+"/"+vaultID+"/"+contactID+".vcf", nil)
	req.SetBasicAuth(email, password)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertVCardUTF8ContentType(t, rec.Header().Get("Content-Type"))
	assertBodyContains(t, rec.Body.String(), "Róisín Ní")
}

func TestDAVHeadAddressObject_SetsUTF8ContentType(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Raúl", "García")

	req := httptest.NewRequest(http.MethodHead, "/dav/addressbooks/"+userID+"/"+vaultID+"/"+contactID+".vcf", nil)
	req.SetBasicAuth(email, password)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertVCardUTF8ContentType(t, rec.Header().Get("Content-Type"))
}

func setupDAVHTTPTest(t *testing.T) *echo.Echo {
	t.Helper()
	e, _ := setupDAVHTTPTestWithDB(t)
	return e
}

func setupDAVHTTPTestWithDB(t *testing.T) (*echo.Echo, *gorm.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	e := echo.New()
	e.Use(appMiddleware.CORS())
	SetupDAVRoutes(e, db)
	return e, db
}

func createDAVHTTPTestUser(t *testing.T, db *gorm.DB) (string, string, string) {
	t.Helper()
	password := "password123"
	email := "principal-dav@example.com"
	resp, err := services.NewAuthService(db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{
		FirstName: "Principal",
		LastName:  "User",
		Email:     email,
		Password:  password,
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	return resp.User.ID, email, password
}

func createDAVHTTPTestContact(t *testing.T, db *gorm.DB, userID, firstName, lastName string) (string, string) {
	t.Helper()

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}

	vault, err := services.NewVaultService(db).CreateVault(user.AccountID, user.ID, dto.CreateVaultRequest{Name: "DAV HTTP Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	now := time.Now()
	contact := models.Contact{
		VaultID:       vault.ID,
		FirstName:     &firstName,
		LastName:      &lastName,
		LastUpdatedAt: &now,
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("create contact: %v", err)
	}
	if err := db.Create(&models.ContactVaultUser{ContactID: contact.ID, UserID: userID, VaultID: vault.ID}).Error; err != nil {
		t.Fatalf("create contact vault user: %v", err)
	}

	return vault.ID, contact.ID
}

func assertHeaderContains(t *testing.T, header http.Header, name, want string) {
	t.Helper()
	if got := header.Get(name); !strings.Contains(got, want) {
		t.Errorf("expected %s header to contain %q, got %q", name, want, got)
	}
}

func assertBodyContains(t *testing.T, body, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Errorf("expected response body to contain %q, got %s", want, body)
	}
}

func assertVCardUTF8ContentType(t *testing.T, contentType string) {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse Content-Type %q: %v", contentType, err)
	}
	if mediaType != "text/vcard" {
		t.Fatalf("expected media type text/vcard, got %q", mediaType)
	}
	if params["charset"] != "utf-8" {
		t.Fatalf("expected charset=utf-8, got %q in %q", params["charset"], contentType)
	}
}
