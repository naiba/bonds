package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRequestLoggerOmitsCredentials(t *testing.T) {
	var output bytes.Buffer
	e := echo.New()
	e.Logger = slog.New(slog.NewJSONHandler(&output, nil))
	e.Use(RequestLogger())
	e.GET("/api/auth/:provider/callback", func(c *echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "raw-secret-error")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/blog/callback?code=secret-code&state=secret-state&token=secret-token&code_verifier=secret-verifier", nil)
	req.Header.Set("Authorization", "Bearer secret-bearer")
	e.ServeHTTP(httptest.NewRecorder(), req)
	if !strings.Contains(output.String(), "/api/auth/:provider/callback") || !strings.Contains(output.String(), "400") {
		t.Fatal("request route and status must remain diagnosable")
	}
	if strings.Contains(output.String(), "secret") {
		t.Fatal("request logging leaked a query parameter, header or error")
	}
}
