package handlers_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
)

// An SVG is a document that can carry scripts. Uploading one is refused now,
// but files stored by earlier versions are still there: served inline they run
// in the application's origin with the viewer's session.
func TestFilePreview_StoredSVGIsNotServedInline(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "stored-svg-preview@example.com")
	vault := ts.createTestVault(t, token, "Stored SVG Vault")

	storedFiles := map[string]string{
		"image/svg+xml": "stored-svg-uuid",
		"image/x+xml":   "stored-xml-image-uuid",
		"image/png":     "stored-png-uuid",
	}
	fileIDs := map[string]uint{}
	for mimeType, fileUUID := range storedFiles {
		if err := os.WriteFile(filepath.Join(ts.cfg.Storage.UploadDir, fileUUID), []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>1</script></svg>"), 0o600); err != nil {
			t.Fatalf("write stored file: %v", err)
		}
		file := models.File{VaultID: vault.ID, UUID: fileUUID, Name: "picture", MimeType: mimeType, Type: "photo", Size: 64}
		if err := ts.db.Create(&file).Error; err != nil {
			t.Fatalf("create file record: %v", err)
		}
		fileIDs[mimeType] = file.ID
	}
	preview := func(mimeType string) (disposition, contentSecurityPolicy string) {
		rec := ts.doRequest(http.MethodGet, fmt.Sprintf("/api/vaults/%s/files/%d/download?preview=true", vault.ID, fileIDs[mimeType]), "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("preview of %s: status %d: %s", mimeType, rec.Code, rec.Body.String())
		}
		return rec.Header().Get("Content-Disposition"), rec.Header().Get("Content-Security-Policy")
	}

	for _, scriptable := range []string{"image/svg+xml", "image/x+xml"} {
		if disposition, _ := preview(scriptable); !strings.HasPrefix(disposition, "attachment") {
			t.Errorf("%s preview Content-Disposition = %q, want attachment", scriptable, disposition)
		}
	}
	disposition, contentSecurityPolicy := preview("image/png")
	if !strings.HasPrefix(disposition, "inline") {
		t.Errorf("bitmap preview Content-Disposition = %q, want inline", disposition)
	}
	if !strings.Contains(contentSecurityPolicy, "sandbox") {
		t.Errorf("inline preview Content-Security-Policy = %q, want a sandbox", contentSecurityPolicy)
	}
}
