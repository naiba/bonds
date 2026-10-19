package dav

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/naiba/bonds/pkg/avatar"
)

func TestCardDAVReturnsUploadedAvatar(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Photo", "Contact")
	uploadDir := backend.uploadDir
	files := services.NewVaultFileService(db, uploadDir)
	photo := avatar.GenerateInitials("Photo Contact", 32)
	if _, err := files.UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", int64(len(photo)), bytes.NewReader(photo)); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID)
	object, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, object.Card, photo, "PNG")
}

func assertCardDAVPhoto(t *testing.T, card vcard.Card, want []byte, imageType string) {
	t.Helper()
	if card.Value(vcard.FieldVersion) != "3.0" {
		t.Fatal("expected vCard 3.0")
	}
	photos := card[vcard.FieldPhoto]
	if len(photos) != 1 {
		t.Fatalf("expected one embedded PHOTO, got %d", len(photos))
	}
	photo := photos[0]
	if photo.Params.Get("ENCODING") != "b" || photo.Params.Get(vcard.ParamType) != imageType || photo.Params.Get(vcard.ParamValue) != "" || photo.Params.Get(vcard.ParamMediaType) != "" {
		t.Fatalf("invalid vCard 3.0 PHOTO parameters: %#v", photo.Params)
	}
	decoded, err := base64.StdEncoding.DecodeString(photo.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, want) {
		t.Fatal("PHOTO bytes differ from uploaded avatar")
	}
}

func TestCardDAVPhotoLifecycleAndETags(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Photo", "Lifecycle")
	files := services.NewVaultFileService(db, backend.uploadDir)
	path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID)
	bookPath := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	original, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(original.Card[vcard.FieldPhoto]) != 0 {
		t.Fatal("unexpected generated initials in CardDAV")
	}
	photo := avatar.GenerateInitials("Selected Avatar", 32)
	uploaded, err := files.UploadContactAvatar(contact.ID, vaultID, userID, "selected.png", "image/png", int64(len(photo)), bytes.NewReader(photo))
	if err != nil {
		t.Fatal(err)
	}
	// Other gallery photos must not replace or multiply the selected avatar.
	if _, err := files.Upload(vaultID, contact.ID, userID, "photo", "gallery.png", "image/png", int64(len(photo)), bytes.NewReader(avatar.GenerateInitials("Gallery", 32))); err != nil {
		t.Fatal(err)
	}
	current, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, current.Card, photo, "PNG")
	if current.ETag == original.ETag {
		t.Fatal("upload did not change ETag")
	}
	listed, err := backend.ListAddressObjects(ctx, bookPath, nil)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: %v, count %d", err, len(listed))
	}
	queried, err := backend.QueryAddressObjects(ctx, bookPath, nil)
	if err != nil || len(queried) != 1 {
		t.Fatalf("query: %v, count %d", err, len(queried))
	}
	for _, object := range []carddav.AddressObject{listed[0], queried[0]} {
		assertCardDAVPhoto(t, object.Card, photo, "PNG")
		if object.ETag != current.ETag {
			t.Fatal("inconsistent ETags between DAV entry points")
		}
	}
	// PUT is still limited to supported contact fields; round-tripping a card
	// must preserve the avatar reference and its file instead of duplicating it.
	updated, err := backend.PutAddressObject(ctx, path, current.Card, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, updated.Card, photo, "PNG")
	var count int64
	if err := db.Model(&models.File{}).Where("vault_id = ?", vaultID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("round-trip changed file count: %d", count)
	}
	replacement := avatar.GenerateInitials("Replacement", 48)
	// Content changes must be visible even if no Contact timestamp changes.
	if err := os.WriteFile(filepath.Join(backend.uploadDir, uploaded.UUID), replacement, 0600); err != nil {
		t.Fatal(err)
	}
	replaced, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, replaced.Card, replacement, "PNG")
	if replaced.ETag == updated.ETag {
		t.Fatal("changed photo bytes did not change ETag")
	}
	stale := &carddav.PutAddressObjectOptions{IfMatch: webdav.ConditionalMatch(`"` + updated.ETag + `"`)}
	if _, err := backend.PutAddressObject(ctx, path, current.Card, stale); err == nil {
		t.Fatal("stale photo ETag accepted")
	}
	if _, err := services.NewContactAvatarService(db).DeleteAvatar(contact.ID, vaultID, userID); err != nil {
		t.Fatal(err)
	}
	removed, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Card[vcard.FieldPhoto]) != 0 || removed.ETag == replaced.ETag {
		t.Fatal("avatar removal was not reflected in card and ETag")
	}
	if _, err := os.Stat(filepath.Join(backend.uploadDir, uploaded.UUID)); err != nil {
		t.Fatalf("clearing avatar unexpectedly removed gallery file: %v", err)
	}
}

func TestCardDAVPhotoStorageBoundaries(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Storage", "Boundary")
	path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID)
	photo := avatar.GenerateInitials("Legacy Photo", 32)
	legacyPath := filepath.Join(backend.uploadDir, "legacy", "portrait.png")
	if err := os.Mkdir(filepath.Dir(legacyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, photo, 0600); err != nil {
		t.Fatal(err)
	}
	file := models.File{VaultID: vaultID, UUID: "legacy-portrait", OriginalURL: &legacyPath, MimeType: "image/png", Name: "portrait.png", Type: "photo", Size: len(photo)}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(contact).Update("file_id", file.ID).Error; err != nil {
		t.Fatal(err)
	}
	object, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, object.Card, photo, "PNG")
	// Missing files must not prevent the rest of an address book from syncing.
	if err := os.Remove(legacyPath); err != nil {
		t.Fatal(err)
	}
	missing, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing.Card[vcard.FieldPhoto]) != 0 || missing.ETag == object.ETag {
		t.Fatal("missing photo not reflected in card and ETag")
	}
	// Keep existing remote URI support, without fetching that URL server-side.
	remoteURL := "https://example.invalid/portrait.png"
	if err := db.Model(&file).Update("original_url", remoteURL).Error; err != nil {
		t.Fatal(err)
	}
	remote, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if remote.Card.Value(vcard.FieldPhoto) != remoteURL || remote.Card.Get(vcard.FieldPhoto).Params.Get(vcard.ParamValue) != "uri" {
		t.Fatal("remote PHOTO URI compatibility lost")
	}
	// Even a corrupt avatar reference must not leak a different vault's URL.
	if err := db.Model(&file).Update("vault_id", "inaccessible-vault").Error; err != nil {
		t.Fatal(err)
	}
	foreign, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(foreign.Card[vcard.FieldPhoto]) != 0 {
		t.Fatal("foreign vault photo exposed")
	}
	if err := db.Model(&file).Updates(map[string]interface{}{"vault_id": vaultID, "original_url": nil}).Error; err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private.png")
	if err := os.WriteFile(outside, photo, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(backend.uploadDir, file.UUID)); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.GetAddressObject(ctx, path, nil); err == nil {
		t.Fatal("symlink outside upload directory accepted")
	}
	if err := os.Remove(filepath.Join(backend.uploadDir, file.UUID)); err != nil {
		t.Fatal(err)
	}
	outsideRelative, err := filepath.Rel(backend.uploadDir, outside)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&file).Update("uuid", outsideRelative).Error; err != nil {
		t.Fatal(err)
	}
	traversal, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(traversal.Card[vcard.FieldPhoto]) != 0 {
		t.Fatal("path traversal photo exposed")
	}
	if _, err := backend.GetAddressObject(WithUserID(ctx, "unauthorized-user"), path, nil); err == nil {
		t.Fatal("unauthorized DAV read accepted")
	}
}

func TestDAVHTTPReturnsPhotoInGETAndReports(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uploadDir := t.TempDir()
	e := echo.New()
	SetupDAVRoutes(e, db, uploadDir)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "HTTP", "Photo")
	photo := avatar.GenerateInitials("HTTP Photo", 64)
	uploaded, err := services.NewVaultFileService(db, uploadDir).UploadContactAvatar(contactID, vaultID, userID, "portrait.png", "image/png", int64(len(photo)), bytes.NewReader(photo))
	if err != nil {
		t.Fatal(err)
	}
	bookPath := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	href := bookPath + contactID + ".vcf"
	get := func(auth bool, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, href, nil)
		if auth {
			req.SetBasicAuth(email, password)
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	if rec := get(false, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET: %d", rec.Code)
	}
	response := get(true, "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET: %d %s", response.Code, response.Body.String())
	}
	card, err := vcard.NewDecoder(strings.NewReader(response.Body.String())).Decode()
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, card, photo, "PNG")
	etag := response.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing GET ETag")
	}
	if rec := get(true, ""); rec.Code != http.StatusOK || rec.Header().Get("ETag") != etag {
		t.Fatalf("repeated GET changed status or ETag: %d %q", rec.Code, rec.Header().Get("ETag"))
	}
	for _, reportType := range []string{"addressbook-multiget", "addressbook-query"} {
		t.Run(reportType, func(t *testing.T) {
			body := `<C:` + reportType + ` xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><D:getetag/><C:address-data content-type="text/vcard" version="3.0"/></D:prop>`
			if reportType == "addressbook-multiget" {
				body += `<D:href>` + href + `</D:href>`
			} else {
				body += `<C:filter/>`
			}
			body += `</C:` + reportType + `>`
			req := httptest.NewRequest("REPORT", bookPath, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/xml")
			req.Header.Set("Depth", "1")
			req.SetBasicAuth(email, password)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusMultiStatus {
				t.Fatalf("REPORT: %d %s", rec.Code, rec.Body.String())
			}
			var report cardDAVMultigetReport
			if err := xml.Unmarshal(rec.Body.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if len(report.Responses) != 1 {
				t.Fatalf("expected one response, got %d", len(report.Responses))
			}
			object := report.Responses[0]
			if object.Href != href || object.Propstat.Status != "HTTP/1.1 200 OK" || object.Propstat.Prop.ETag != etag {
				t.Fatalf("incorrect REPORT metadata: %#v", object)
			}
			card, err := vcard.NewDecoder(strings.NewReader(object.Propstat.Prop.AddressData)).Decode()
			if err != nil {
				t.Fatal(err)
			}
			assertCardDAVPhoto(t, card, photo, "PNG")
		})
	}
	if _, err := services.NewContactAvatarService(db).DeleteAvatar(contactID, vaultID, userID); err != nil {
		t.Fatal(err)
	}
	removed := get(true, etag)
	if removed.Code != http.StatusOK || removed.Header().Get("ETag") == etag {
		t.Fatal("conditional GET did not detect photo removal")
	}
	card, err = vcard.NewDecoder(strings.NewReader(removed.Body.String())).Decode()
	if err != nil {
		t.Fatal(err)
	}
	if len(card[vcard.FieldPhoto]) != 0 {
		t.Fatal("removed avatar still returned")
	}
	// Broken storage must fail without sending local filesystem paths to clients.
	if err := db.Model(&models.Contact{}).Where("id = ?", contactID).Update("file_id", uploaded.ID).Error; err != nil {
		t.Fatal(err)
	}
	selectedPath := filepath.Join(uploadDir, uploaded.UUID)
	if err := os.Remove(selectedPath); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private-photo.png")
	if err := os.WriteFile(outside, photo, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, selectedPath); err != nil {
		t.Fatal(err)
	}
	broken := get(true, "")
	if broken.Code != http.StatusInternalServerError || !strings.Contains(broken.Body.String(), "contact photo unavailable") {
		t.Fatalf("unexpected storage error response: %d %s", broken.Code, broken.Body.String())
	}
	if strings.Contains(broken.Body.String(), uploadDir) || strings.Contains(broken.Body.String(), outside) || strings.Contains(broken.Body.String(), base64.StdEncoding.EncodeToString(photo)) {
		t.Fatal("storage error leaked path or photo contents")
	}
}

func TestCardDAVPhotoImageTypes(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Image", "Types")
	path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID)
	var jpegData, gifData bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if err := jpeg.Encode(&jpegData, picture, nil); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifData, picture, nil); err != nil {
		t.Fatal(err)
	}
	webpData, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		data      []byte
		imageType string
	}{
		{"jpeg", jpegData.Bytes(), "JPEG"},
		{"gif", gifData.Bytes(), "GIF"},
		{"webp", webpData, "WEBP"},
		{"non-image", []byte("private document text"), ""},
		{"empty", nil, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Detect the actual image type instead of trusting upload metadata.
			if _, err := services.NewVaultFileService(db, backend.uploadDir).UploadContactAvatar(contact.ID, vaultID, userID, "portrait", "image/png", int64(len(test.data)), bytes.NewReader(test.data)); err != nil {
				t.Fatal(err)
			}
			object, err := backend.GetAddressObject(ctx, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if test.imageType == "" {
				if len(object.Card[vcard.FieldPhoto]) != 0 {
					t.Fatal("non-image content exposed as photo")
				}
			} else {
				assertCardDAVPhoto(t, object.Card, test.data, test.imageType)
			}
		})
	}
}
