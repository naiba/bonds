package mcp

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

func RequireAllowedOrigin(allowedOrigins ...string) echo.MiddlewareFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if normalized := normalizeOrigin(origin); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get(echo.HeaderOrigin)
			if origin == "" {
				return next(c)
			}
			if _, ok := allowed[normalizeOrigin(origin)]; !ok {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

func normalizeOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
