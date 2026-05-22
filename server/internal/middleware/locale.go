package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/naiba/bonds/internal/i18n"
)

// Locale parses the Accept-Language header and stores the matched code in the
// echo context. The whitelist comes from i18n.Supported so adding a language
// is a one-line change there. Tags are normalized to the primary subtag
// (e.g. "zh-CN" → "zh") so callers that send region-qualified codes still
// match the loaded bundles. Anything not in the bundle defaults to "en".
func Locale() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			lang := "en"
			accept := c.Request().Header.Get("Accept-Language")
			if accept != "" {
				tag := strings.SplitN(accept, ",", 2)[0]
				tag = strings.SplitN(tag, ";", 2)[0]
				tag = strings.TrimSpace(tag)
				primary := strings.ToLower(strings.SplitN(tag, "-", 2)[0])
				if i18n.IsSupported(primary) {
					lang = primary
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
