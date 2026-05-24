package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequireAllowedOriginBlocksUnexpectedOrigins(t *testing.T) {
	e := echo.New()
	middleware := RequireAllowedOrigin("http://localhost:5173")
	handler := middleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set(echo.HeaderOrigin, "https://evil.example")
	rec := httptest.NewRecorder()

	if err := handler(e.NewContext(req, rec)); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unexpected origin, got %d", rec.Code)
	}
}

func TestRequireAllowedOriginAllowsConfiguredOriginAndNoOrigin(t *testing.T) {
	e := echo.New()
	middleware := RequireAllowedOrigin("http://localhost:5173")
	handler := middleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	for _, origin := range []string{"http://localhost:5173", ""} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if origin != "" {
			req.Header.Set(echo.HeaderOrigin, origin)
		}
		rec := httptest.NewRecorder()
		if err := handler(e.NewContext(req, rec)); err != nil {
			t.Fatalf("handler returned error for origin %q: %v", origin, err)
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 for origin %q, got %d", origin, rec.Code)
		}
	}
}
