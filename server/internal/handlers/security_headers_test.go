package handlers_test

import (
	"net/http"
	"testing"
)

// Session tokens can appear in URLs (file previews, downloads). Without a
// referrer policy the browser forwards those URLs to any third-party site a
// user navigates to from the app.
func TestResponsesCarryBaselineSecurityHeaders(t *testing.T) {
	ts := setupTestServer(t)

	for _, path := range []string{"/api/instance/info", "/api/vaults"} {
		rec := ts.doRequest(http.MethodGet, path, "", "")
		if got := rec.Header().Get("Referrer-Policy"); got != "same-origin" {
			t.Errorf("%s: Referrer-Policy = %q, want same-origin", path, got)
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", path, got)
		}
	}
}
