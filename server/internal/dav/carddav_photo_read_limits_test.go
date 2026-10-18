package dav

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/avatar"
)

// Generate finite synthetic bodies without allocating the input size in memory.
type cardDAVPhotoPaddingReader struct{}

func (cardDAVPhotoPaddingReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type cardDAVPhotoCountingReader struct {
	source io.Reader
	bytes  int64
}

func (r *cardDAVPhotoCountingReader) Read(p []byte) (int, error) {
	n, err := r.source.Read(p)
	r.bytes += int64(n)
	return n, err
}

type cardDAVPhotoFailureReader struct{ err error }

func (r cardDAVPhotoFailureReader) Read([]byte) (int, error) { return 0, r.err }

func TestCardDAVPhotoReaderLimits(t *testing.T) {
	photo := avatar.GenerateInitials("Read boundary", 32)
	for _, test := range []struct {
		name     string
		header   []byte
		size     int64
		wantType string
		wantRead int64
	}{
		{"valid-small-png", photo, int64(len(photo)), "PNG", int64(len(photo))},
		{"empty", nil, 0, "", 0},
		{"short-non-image", []byte("not an image"), 12, "", 12},
		{"large-non-image", nil, 24 << 20, "", 512},
		// The reader has no Stat or trusted length: this also models growth after Stat.
		{"image-at-limit", photo, maxCardDAVPhotoBytes, "PNG", maxCardDAVPhotoBytes},
		{"image-above-limit", photo, maxCardDAVPhotoBytes + 1, "", maxCardDAVPhotoBytes + 1},
		{"image-growing-past-limit", photo, 24 << 20, "", maxCardDAVPhotoBytes + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := io.LimitReader(io.MultiReader(bytes.NewReader(test.header), cardDAVPhotoPaddingReader{}), test.size)
			reader := &cardDAVPhotoCountingReader{source: input}
			data, imageType, err := readCardDAVPhoto(reader)
			if err != nil {
				t.Fatal(err)
			}
			if reader.bytes != test.wantRead {
				t.Fatalf("read %d bytes, want %d", reader.bytes, test.wantRead)
			}
			if imageType != test.wantType {
				t.Fatalf("type %q, want %q", imageType, test.wantType)
			}
			if test.wantType == "" {
				if len(data) != 0 {
					t.Fatalf("omitted photo retained %d bytes", len(data))
				}
			} else {
				if int64(len(data)) != test.size || !bytes.Equal(data[:len(test.header)], test.header) {
					t.Fatal("accepted image was truncated or changed")
				}
				if !bytes.Equal(data[len(test.header):], make([]byte, int(test.size)-len(test.header))) {
					t.Fatal("photo body changed")
				}
			}
		})
	}
	failure := errors.New("synthetic read failure")
	for _, test := range []struct {
		name   string
		reader io.Reader
	}{
		{"header", cardDAVPhotoFailureReader{failure}},
		{"body", io.MultiReader(bytes.NewReader(append(photo, make([]byte, 512)...)), cardDAVPhotoFailureReader{failure})},
	} {
		t.Run(test.name+"-failure", func(t *testing.T) {
			data, imageType, err := readCardDAVPhoto(test.reader)
			if !errors.Is(err, failure) || data != nil || imageType != "" {
				t.Fatalf("read failure hidden: bytes=%d type=%q err=%v", len(data), imageType, err)
			}
		})
	}
}

func TestDAVHTTPPhotoSizeFallback(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	handler := &carddav.Handler{Backend: backend, Prefix: "/dav"}
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	photo := avatar.GenerateInitials("Healthy contact", 32)
	type fixture struct {
		href      string
		name      string
		fileID    uint
		diskPath  string
		size      int64
		wantPhoto bool
	}
	var contacts []fixture
	for _, test := range []struct {
		name      string
		header    []byte
		size      int64
		wantPhoto bool
		missing   bool
	}{
		{"Healthy", photo, int64(len(photo)), true, false},
		{"NonImage", []byte("not a photo"), 24 << 20, false, false},
		{"OversizedImage", photo, maxCardDAVPhotoBytes + 1, false, false},
		{"Empty", nil, 0, false, false},
		{"Missing", photo, int64(len(photo)), false, true},
	} {
		contact := createTestContact(t, db, vaultID, userID, test.name, "Sync")
		// Lie about both MIME and size, as import/upload metadata cannot bound reads.
		uploaded, err := services.NewVaultFileService(db, backend.uploadDir).UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", 1, io.LimitReader(io.MultiReader(bytes.NewReader(test.header), cardDAVPhotoPaddingReader{}), test.size))
		if err != nil {
			t.Fatal(err)
		}
		diskPath := filepath.Join(backend.uploadDir, uploaded.UUID)
		info, err := os.Stat(diskPath)
		if err != nil || info.Size() != test.size {
			t.Fatalf("fixture size: %v %v", info, err)
		}
		if test.missing {
			if err := os.Remove(diskPath); err != nil {
				t.Fatal(err)
			}
		}
		contacts = append(contacts, fixture{book + contact.ID + ".vcf", test.name + " Sync", uploaded.ID, diskPath, test.size, test.wantPhoto})
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "application/xml")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	assertCard := func(t *testing.T, text string, contact fixture) {
		t.Helper()
		card, err := vcard.NewDecoder(strings.NewReader(text)).Decode()
		if err != nil {
			t.Fatal(err)
		}
		if card.Value(vcard.FieldFormattedName) != contact.name {
			t.Fatalf("lost contact name: %q", card.Value(vcard.FieldFormattedName))
		}
		if contact.wantPhoto {
			assertCardDAVPhoto(t, card, photo, "PNG")
		} else if len(card[vcard.FieldPhoto]) != 0 {
			t.Fatalf("unavailable photo returned for %s", contact.name)
		}
	}
	etags := map[string]string{}
	for _, contact := range contacts {
		rec := request(http.MethodGet, contact.href, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", contact.name, rec.Code, rec.Body.String())
		}
		assertCard(t, rec.Body.String(), contact)
		etags[contact.href] = rec.Header().Get("ETag")
		if etags[contact.href] == "" {
			t.Fatal("GET missing ETag")
		}
	}
	for _, method := range []string{"PROPFIND", "addressbook-query", "addressbook-multiget"} {
		t.Run(method, func(t *testing.T) {
			body := `<D:propfind xmlns:D="DAV:"><D:prop><D:getetag/></D:prop></D:propfind>`
			verb := "PROPFIND"
			if method != "PROPFIND" {
				verb = "REPORT"
				body = `<C:` + method + ` xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav"><D:prop><D:getetag/><C:address-data/></D:prop>`
				if method == "addressbook-query" {
					body += `<C:filter/>`
				} else {
					for _, contact := range contacts {
						body += `<D:href>` + contact.href + `</D:href>`
					}
				}
				body += `</C:` + method + `>`
			}
			rec := request(verb, book, body)
			if rec.Code != http.StatusMultiStatus {
				t.Fatalf("%s: %d %s", method, rec.Code, rec.Body.String())
			}
			var report cardDAVMultigetReport
			if err := xml.Unmarshal(rec.Body.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, response := range report.Responses {
				for _, contact := range contacts {
					if response.Href != contact.href {
						continue
					}
					if seen[contact.href] {
						t.Fatal("duplicate contact response")
					}
					seen[contact.href] = true
					if response.Propstat.Status != "HTTP/1.1 200 OK" || response.Propstat.Prop.ETag != etags[contact.href] {
						t.Fatalf("inconsistent metadata for %s: %#v", contact.name, response)
					}
					if method == "PROPFIND" {
						if response.Propstat.Prop.AddressData != "" {
							t.Fatal("metadata includes address data")
						}
					} else {
						assertCard(t, response.Propstat.Prop.AddressData, contact)
					}
				}
			}
			if len(seen) != len(contacts) {
				t.Fatalf("synced %d of %d contacts", len(seen), len(contacts))
			}
		})
	}
	// Omission is a read-only projection: selected avatars and stored data survive.
	for _, contact := range contacts {
		var selected models.Contact
		id := strings.TrimSuffix(strings.TrimPrefix(contact.href, book), ".vcf")
		if err := db.First(&selected, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if selected.FileID == nil || *selected.FileID != contact.fileID {
			t.Fatal("sync changed selected avatar")
		}
		if contact.name == "Missing Sync" {
			continue
		}
		info, err := os.Stat(contact.diskPath)
		if err != nil || info.Size() != contact.size {
			t.Fatalf("sync changed stored file: %v %v", info, err)
		}
	}
	var count int64
	if err := db.Model(&models.File{}).Where("vault_id = ?", vaultID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(contacts)) {
		t.Fatalf("sync changed file count: %d", count)
	}
}
