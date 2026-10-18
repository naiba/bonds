package dav

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

func TestCardDAVQueryReleasesUnmatchedPhotoPayloads(t *testing.T) {
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
	const contacts = 9
	for i := 0; i < contacts; i++ {
		contact := createTestContact(t, db, vaultID, userID, fmt.Sprintf("Portrait %d", i), "Memory")
		if _, err := services.NewVaultFileService(db, backend.uploadDir).UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", int64(encoded.Len()), bytes.NewReader(encoded.Bytes())); err != nil {
			t.Fatal(err)
		}
	}
	reads := 0
	var first, last uint64
	// Observe live memory between individual ResolvePath queries. At this point
	// rejected PHOTO strings must already be unreachable before the next read.
	callback := "observe_filtered_photo_memory"
	if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*models.File); !ok || tx.RowsAffected != 1 {
			return
		}
		reads++
		if reads == 1 || reads == contacts {
			runtime.GC()
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			if reads == 1 {
				first = memory.HeapAlloc
			} else {
				last = memory.HeapAlloc
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Query().Remove(callback)
	objects, err := backend.QueryAddressObjects(ctx, fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID), &carddav.AddressBookQuery{
		DataRequest: carddav.AddressDataRequest{Props: []string{vcard.FieldFormattedName}},
		PropFilters: []carddav.PropFilter{{Name: vcard.FieldFormattedName, TextMatches: []carddav.TextMatch{{Text: "No such contact", MatchType: carddav.MatchEquals}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 0 {
		t.Fatalf("unexpected matching contacts: %d", len(objects))
	}
	if reads != contacts || first == 0 || last == 0 {
		t.Fatalf("invalid observation: reads=%d first=%d last=%d", reads, first, last)
	}
	delta := int64(last) - int64(first)
	photoBytes := base64.StdEncoding.EncodedLen(encoded.Len())
	t.Logf("query requests only FN and matches zero contacts; source PNG=%d bytes; contacts=%d; observed per-file queries=%d; live heap growth before final photo=%d bytes; eight preceding PHOTO values=%d bytes", encoded.Len(), contacts, reads, delta, (contacts-1)*photoBytes)
	// Nonmatching photos need only a transient per-contact representation for
	// filtering and ETag calculation, not an accumulated address-book photo set.
	if delta > int64(2*photoBytes) {
		t.Errorf("query retains unmatched photo payloads: live heap grew %d bytes, exceeding two-photo allowance %d", delta, 2*photoBytes)
	}
}
