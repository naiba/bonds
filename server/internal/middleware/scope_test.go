package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func runWithContext(t *testing.T, setup func(c echo.Context), mw echo.MiddlewareFunc) int {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if setup != nil {
		setup(c)
	}
	handler := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return rec.Code
}

func TestRequireScope_FullAccessTokenPasses(t *testing.T) {
	got := runWithContext(t, nil, RequireScope(ScopeCalendarRead))
	if got != http.StatusOK {
		t.Errorf("JWT/full-access (no scoped PAT) should pass, got status %d", got)
	}
}

func TestRequireScope_ScopedTokenWithMatchingScopePasses(t *testing.T) {
	setup := func(c echo.Context) {
		c.Set(ctxPATScopes, "calendar:read")
		c.Set(ctxIsScopedPAT, true)
	}
	got := runWithContext(t, setup, RequireScope(ScopeCalendarRead))
	if got != http.StatusOK {
		t.Errorf("scoped PAT with matching scope should pass, got status %d", got)
	}
}

func TestRequireScope_ScopedTokenWithoutScopeDenied(t *testing.T) {
	setup := func(c echo.Context) {
		c.Set(ctxPATScopes, "something:else")
		c.Set(ctxIsScopedPAT, true)
	}
	got := runWithContext(t, setup, RequireScope(ScopeCalendarRead))
	if got != http.StatusForbidden {
		t.Errorf("scoped PAT without required scope should be denied, got status %d", got)
	}
}

func TestDenyScopedPAT_FullAccessPasses(t *testing.T) {
	got := runWithContext(t, nil, DenyScopedPAT)
	if got != http.StatusOK {
		t.Errorf("full-access token should pass DenyScopedPAT, got status %d", got)
	}
}

func TestDenyScopedPAT_ScopedTokenDenied(t *testing.T) {
	setup := func(c echo.Context) {
		c.Set(ctxPATScopes, "calendar:read")
		c.Set(ctxIsScopedPAT, true)
	}
	got := runWithContext(t, setup, DenyScopedPAT)
	if got != http.StatusForbidden {
		t.Errorf("scoped PAT should be denied by DenyScopedPAT, got status %d", got)
	}
}
