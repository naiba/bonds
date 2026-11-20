package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func createJournalPostForPhotoTest(t *testing.T, ts *testServer, token, vaultID string) string {
	t.Helper()
	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vaultID+"/journals", `{"name":"Photo Journal"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create journal: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var journal struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(parseResponse(t, rec).Data, &journal); err != nil {
		t.Fatalf("parse journal: %v", err)
	}
	postsPath := fmt.Sprintf("/api/vaults/%s/journals/%d/posts", vaultID, journal.ID)
	rec = ts.doRequest(http.MethodPost, postsPath, `{"title":"Photo Post","written_at":"2024-01-01T00:00:00Z"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create post: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(parseResponse(t, rec).Data, &post); err != nil {
		t.Fatalf("parse post: %v", err)
	}
	return fmt.Sprintf("%s/%d/photos", postsPath, post.ID)
}

// Post photos are served inline with the stored Content-Type, so anything the
// browser would execute (SVG) must never be stored as a photo.
func TestPostPhotoUpload_RejectsScriptableOrMislabelledContent(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "post-photo-content@example.com")
	vault := ts.createTestVault(t, token, "Post Photo Vault")
	photosPath := createJournalPostForPhotoTest(t, ts, token, vault.ID)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`)

	rec := ts.doMultipartUpload(t, photosPath, token, "file", "payload.svg", "image/svg+xml", svg)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("SVG upload: status=%d, want 400: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doMultipartUpload(t, photosPath, token, "file", "payload.png", "image/png", svg)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("SVG bytes declared as image/png: status=%d, want 400: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, photosPath, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list photos: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var stored []json.RawMessage
	if err := json.Unmarshal(parseResponse(t, rec).Data, &stored); err != nil {
		t.Fatalf("parse photo list: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("%d rejected upload(s) were stored anyway", len(stored))
	}
}

func TestPostPhotoUpload_AcceptsRealImage(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "post-photo-real@example.com")
	vault := ts.createTestVault(t, token, "Post Photo Vault")
	photosPath := createJournalPostForPhotoTest(t, ts, token, vault.ID)
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))

	rec := ts.doMultipartUpload(t, photosPath, token, "file", "trip.png", "image/png", png)
	if rec.Code != http.StatusCreated {
		t.Fatalf("PNG upload: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var stored struct {
		MimeType string `json:"mime_type"`
	}
	if err := json.Unmarshal(parseResponse(t, rec).Data, &stored); err != nil {
		t.Fatalf("parse upload response: %v", err)
	}
	if stored.MimeType != "image/png" {
		t.Fatalf("stored mime_type = %q, want image/png", stored.MimeType)
	}
}
