package dav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/avatar"
)

func TestCardDAVMetadataReleasesPhotoPayloads(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	picture := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	for i := range picture.Pix {
		picture.Pix[i] = byte(i*31 + i/17)
	}
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		contact := createTestContact(t, db, vaultID, userID, fmt.Sprintf("Portrait %d", i), "Collection")
		if _, err := services.NewVaultFileService(db, backend.uploadDir).UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", int64(encoded.Len()), bytes.NewReader(encoded.Bytes())); err != nil {
			t.Fatal(err)
		}
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	// Same request shape go-webdav uses for PROPFIND/getetag: no address-data.
	objects, err := backend.ListAddressObjects(ctx, fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID), &carddav.AddressDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 3 {
		t.Fatalf("objects=%d", len(objects))
	}
	retained := 0
	for _, object := range objects {
		if object.ETag == "" {
			t.Fatal("missing metadata ETag")
		}
		retained += len(object.Card.Value(vcard.FieldPhoto))
		full, err := backend.GetAddressObject(ctx, object.Path, nil)
		if err != nil {
			t.Fatal(err)
		}
		assertCardDAVPhoto(t, full.Card, encoded.Bytes(), "PNG")
		if object.ETag != full.ETag {
			t.Fatal("metadata ETag differs from full representation")
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(objects)
	t.Logf("metadata-only list: contacts=%d, source PNG bytes=%d each, retained PHOTO bytes=%d, live heap delta=%d", len(objects), encoded.Len(), retained, int64(after.HeapAlloc)-int64(before.HeapAlloc))
	observed := &cardDAVMetadataObserver{CardDAVBackend: backend}
	handler := &carddav.Handler{Backend: observed, Prefix: "/dav"}
	req := httptest.NewRequest("PROPFIND", fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID), strings.NewReader(`<D:propfind xmlns:D="DAV:"><D:prop><D:getetag/></D:prop></D:propfind>`)).WithContext(ctx)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMultiStatus {
		t.Fatalf("metadata PROPFIND failed: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "address-data") || strings.Contains(rec.Body.String(), "PHOTO;") {
		t.Fatal("metadata request unexpectedly included photos on the wire")
	}
	if !observed.metadataOnly || observed.photoBytes != retained {
		t.Fatalf("PROPFIND used a different path: metadata=%v photo bytes=%d", observed.metadataOnly, observed.photoBytes)
	}
	var report cardDAVMultigetReport
	if err := xml.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	for _, object := range objects {
		found := false
		for _, response := range report.Responses {
			if response.Href == object.Path {
				found = true
				if response.Propstat.Status != "HTTP/1.1 200 OK" || response.Propstat.Prop.ETag != "\""+object.ETag+"\"" {
					t.Fatal("PROPFIND returned incorrect ETag or status")
				}
			}
		}
		if !found {
			t.Fatalf("PROPFIND omitted %s", object.Path)
		}
	}
	t.Logf("actual PROPFIND/getetag response: %d bytes, PHOTO bytes retained internally: %d", rec.Body.Len(), observed.photoBytes)
	if retained != 0 {
		t.Errorf("metadata-only list unnecessarily retains %d bytes of PHOTO payloads", retained)
	}
}

type cardDAVMetadataObserver struct {
	*CardDAVBackend
	metadataOnly bool
	photoBytes   int
}

func (b *cardDAVMetadataObserver) ListAddressObjects(ctx context.Context, path string, request *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	b.metadataOnly = request != nil && !request.AllProp && len(request.Props) == 0
	objects, err := b.CardDAVBackend.ListAddressObjects(ctx, path, request)
	for _, object := range objects {
		b.photoBytes += len(object.Card.Value(vcard.FieldPhoto))
	}
	return objects, err
}

func TestCardDAVPhotoQueryPreservesDefaultDataAndFilters(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Selected", "Photo")
	createTestContact(t, db, vaultID, userID, "Without", "Photo")
	picture := avatar.GenerateInitials("Query photo", 32)
	files := services.NewVaultFileService(db, backend.uploadDir)
	if _, err := files.UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", int64(len(picture)), bytes.NewReader(picture)); err != nil {
		t.Fatal(err)
	}
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	href := book + contact.ID + ".vcf"
	full, err := backend.GetAddressObject(ctx, href, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []carddav.AddressDataRequest{{}, {AllProp: true}, {Props: []string{vcard.FieldFormattedName}}} {
		query := &carddav.AddressBookQuery{DataRequest: request, PropFilters: []carddav.PropFilter{{Name: vcard.FieldPhoto}}}
		objects, err := backend.QueryAddressObjects(ctx, book, query)
		if err != nil {
			t.Fatal(err)
		}
		if len(objects) != 1 || objects[0].Path != href || objects[0].ETag != full.ETag {
			t.Fatalf("photo query returned incorrect metadata: %#v", objects)
		}
		// Property selection must not discard PHOTO before evaluating filters.
		// Default address-data and full requests still include the complete photo.
		if len(request.Props) == 0 {
			assertCardDAVPhoto(t, objects[0].Card, picture, "PNG")
		}
	}
	// Filtering must see the same PHOTO bytes as GET, even when the caller
	// requests only FN. A nonmatching PHOTO must not leak into the result.
	for _, negate := range []bool{false, true} {
		objects, err := backend.QueryAddressObjects(ctx, book, &carddav.AddressBookQuery{
			DataRequest: carddav.AddressDataRequest{Props: []string{vcard.FieldFormattedName}},
			PropFilters: []carddav.PropFilter{{Name: vcard.FieldPhoto, TextMatches: []carddav.TextMatch{{
				Text: full.Card.Value(vcard.FieldPhoto), MatchType: carddav.MatchEquals, NegateCondition: negate,
			}}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if negate {
			if len(objects) != 0 {
				t.Fatal("negated PHOTO match returned a contact")
			}
		} else {
			if len(objects) != 1 || objects[0].Path != href || objects[0].ETag != full.ETag {
				t.Fatal("PHOTO value filter changed matching contact or ETag")
			}
			assertCardDAVPhoto(t, objects[0].Card, picture, "PNG")
		}
	}
	missing, err := backend.QueryAddressObjects(ctx, book, &carddav.AddressBookQuery{PropFilters: []carddav.PropFilter{{Name: vcard.FieldPhoto, IsNotDefined: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0].Path == href || len(missing[0].Card[vcard.FieldPhoto]) != 0 {
		t.Fatal("PHOTO absence filter returned wrong contacts")
	}

	// A metadata-only sync must keep detecting changes even though it no longer
	// retains the PHOTO field. Verify both replacement and removal.
	previousETag := full.ETag
	for _, remove := range []bool{false, true} {
		if remove {
			if _, err := services.NewContactAvatarService(db).DeleteAvatar(contact.ID, vaultID, userID); err != nil {
				t.Fatal(err)
			}
		} else {
			picture = avatar.GenerateInitials("Replacement portrait", 64)
			if _, err := files.UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", int64(len(picture)), bytes.NewReader(picture)); err != nil {
				t.Fatal(err)
			}
		}
		metadata, err := backend.ListAddressObjects(ctx, book, &carddav.AddressDataRequest{})
		if err != nil {
			t.Fatal(err)
		}
		full, err := backend.GetAddressObject(ctx, href, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, object := range metadata {
			if len(object.Card[vcard.FieldPhoto]) != 0 {
				t.Fatal("metadata retained PHOTO")
			}
			if object.Path == href {
				found = true
				if object.ETag == previousETag || object.ETag != full.ETag {
					t.Fatal("metadata ETag does not track changed representation")
				}
			}
		}
		if !found {
			t.Fatal("metadata lost the contact")
		}
		if remove {
			if len(full.Card[vcard.FieldPhoto]) != 0 {
				t.Fatal("removed photo remains in GET")
			}
		} else {
			assertCardDAVPhoto(t, full.Card, picture, "PNG")
		}
		previousETag = full.ETag
	}
}
