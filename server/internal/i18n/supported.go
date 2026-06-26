package i18n

// Supported lists the locale codes the embedded bundle actually loads.
// Adding a language means: drop a `<code>.json` next to en.json, add the
// code here, and add it to web/src/i18n.ts SUPPORTED_LANGUAGES so the
// frontend and backend agree.
//
// This is the single source of truth consulted by:
//   - middleware/locale.go (Accept-Language parsing)
//   - dto/settings.go validator tags (oneof=...) — kept in sync manually
//     because struct tags must be literals
//   - services/preferences.go (defensive check on the service layer)
//
// Keeping these three in sync prevents the trap where a user persists an
// unsupported locale (e.g. "ja") and the UI silently falls back to English.
var Supported = []string{"en", "zh", "es", "fr", "de", "pt-BR", "pt-PT"}

// IsSupported reports whether lang exactly matches one of the loaded
// locale bundles. Callers that need to coerce region tags like "zh-CN"
// should do so before calling this.
func IsSupported(lang string) bool {
	for _, code := range Supported {
		if code == lang {
			return true
		}
	}
	return false
}
