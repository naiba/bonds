package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/naiba/bonds/internal/i18n"
)

// normalizeTag converts a BCP-47 tag from any casing (e.g. "pt-br", "PT-BR")
// to the canonical form used in i18n.Supported: language lowercase, region
// uppercase (e.g. "pt-BR").
func normalizeTag(tag string) string {
	parts := strings.SplitN(tag, "-", 2)
	if len(parts) == 2 {
		return strings.ToLower(parts[0]) + "-" + strings.ToUpper(parts[1])
	}
	return strings.ToLower(parts[0])
}

// regionFallbacks maps bare primary codes without a bundle (e.g. "pt") to a
// region-specific variant that does exist. Keep in sync with REGION_FALLBACKS
// in web/src/i18n.ts.
var regionFallbacks = map[string]string{
	"pt": "pt-PT",
}

// Locale parses the Accept-Language header and stores the matched code in the
// echo context. The whitelist comes from i18n.Supported so adding a language
// is a one-line change there. Tags are first tried as-is (e.g. "pt-BR"), then
// the primary subtag (e.g. "zh-CN" → "zh"), then regionFallbacks (e.g. "pt" →
// "pt-PT"), so region-qualified codes for languages with region-specific
// bundles (pt-BR/pt-PT) resolve correctly. Anything not in the bundle defaults
// to "en".
func Locale() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			lang := "en"
			accept := c.Request().Header.Get("Accept-Language")
			if accept != "" {
				tag := strings.SplitN(accept, ",", 2)[0]
				tag = strings.SplitN(tag, ";", 2)[0]
				tag = normalizeTag(strings.TrimSpace(tag))
				if i18n.IsSupported(tag) {
					lang = tag
				} else {
					primary := strings.SplitN(tag, "-", 2)[0]
					if i18n.IsSupported(primary) {
						lang = primary
					} else if fallback, ok := regionFallbacks[primary]; ok && i18n.IsSupported(fallback) {
						lang = fallback
					}
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
