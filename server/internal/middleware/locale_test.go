package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestLocaleAcceptLanguageParsing covers the supported codes plus the most
// common ways they arrive in the wild (region-qualified, multi-tag with
// q-values, garbage). The middleware must never panic and must never set
// a locale that's outside the embedded i18n bundle — anything unrecognized
// has to fall through to "en" so i18n.T() finds the message instead of
// returning the key.
func TestLocaleAcceptLanguageParsing(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		want           string
	}{
		{"empty header", "", "en"},
		{"english", "en", "en"},
		{"english us", "en-US", "en"},
		{"chinese plain", "zh", "zh"},
		{"chinese mainland", "zh-CN", "zh"},
		{"chinese simplified", "zh-Hans", "zh"},
		{"spanish", "es", "es"},
		{"spanish argentina", "es-AR", "es"},
		{"french", "fr", "fr"},
		{"french france", "fr-FR", "fr"},
		{"german", "de", "de"},
		{"german with q", "de;q=0.9", "de"},
		{"multi-tag picks first", "zh-CN,en;q=0.8", "zh"},
		{"uppercase", "ZH", "zh"},
		{"whitespace", "  zh-CN  ;q=0.9  ", "zh"},
		{"garbage", "xx-YY", "en"},
	}

	e := echo.New()
	handler := Locale()(func(c echo.Context) error {
		return c.String(http.StatusOK, GetLocale(c))
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if err := handler(c); err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if got := rec.Body.String(); got != tt.want {
				t.Errorf("Accept-Language %q → %q, want %q", tt.acceptLanguage, got, tt.want)
			}
		})
	}
}

func TestGetLocaleDefault(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if got := GetLocale(c); got != "en" {
		t.Errorf("GetLocale on bare context = %q, want %q", got, "en")
	}
}
