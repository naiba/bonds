package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// Locale parses the Accept-Language header and stores the locale in the context.
// Supports "zh", "zh-CN", "zh-Hans" → "zh"; everything else → "en".
func Locale() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			lang := "en"
			accept := c.Request().Header.Get("Accept-Language")
			if accept != "" {
				// Take the first language tag (before any comma)
				tag := strings.SplitN(accept, ",", 2)[0]
				// Remove quality value if present (e.g. "zh-CN;q=0.9")
				tag = strings.SplitN(tag, ";", 2)[0]
				tag = strings.TrimSpace(tag)
				primary := strings.SplitN(tag, "-", 2)[0]
				primary = strings.ToLower(primary)
				if primary == "zh" {
					lang = "zh"
				}
			}
			c.Set("locale", lang)
			return next(c)
		}
	}
}

// GetLocale returns the locale stored in the echo context, defaulting to "en".
func GetLocale(c echo.Context) string {
	if locale, ok := c.Get("locale").(string); ok && locale != "" {
		return locale
	}
	return "en"
}
