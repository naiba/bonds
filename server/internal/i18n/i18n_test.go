package i18n

import "testing"

func TestTUnknownKeyReturnsKey(t *testing.T) {
	if got := T("en", "nonexistent.key.never.added"); got != "nonexistent.key.never.added" {
		t.Errorf("T fallback chain broken: got %q", got)
	}
}

func TestTFallsBackToEnglish(t *testing.T) {
	// err.validation_error exists in en.json. Asking for a locale that doesn't
	// translate it should hit the English fallback rather than the raw key.
	got := T("es", "err.validation_error")
	if got == "err.validation_error" {
		t.Errorf("English fallback not engaged: got raw key")
	}
}

// TestTtSubstitutesNamedPlaceholders covers the new Tt helper. Backend
// translations carry {{name}} placeholders (matching the i18next pattern the
// frontend uses), so the same source string can be reused on both sides.
// Without substitution the placeholder would leak into emails verbatim.
func TestTtSubstitutesNamedPlaceholders(t *testing.T) {
	tests := []struct {
		name   string
		lang   string
		key    string
		params map[string]string
		// We can't pin exact translations without coupling to the JSON, but we
		// can assert the placeholder is gone and substituted values appear.
		mustContain    []string
		mustNotContain []string
	}{
		{
			name:           "single placeholder",
			lang:           "en",
			key:            "test.greeting",
			params:         map[string]string{"name": "Naiba"},
			mustContain:    []string{"Naiba"},
			mustNotContain: []string{"{{name}}"},
		},
		{
			name:           "missing param leaves placeholder alone",
			lang:           "en",
			key:            "test.greeting",
			params:         map[string]string{},
			mustContain:    []string{"{{name}}"},
			mustNotContain: nil,
		},
		{
			name:           "unknown key returns key (no substitution attempted)",
			lang:           "en",
			key:            "nonexistent.key.with.placeholder",
			params:         map[string]string{"x": "y"},
			mustContain:    []string{"nonexistent.key.with.placeholder"},
			mustNotContain: nil,
		},
	}
	// Seed a minimal in-memory translation for the test.
	once.Do(load)
	if _, ok := translations["en"]; !ok {
		translations["en"] = map[string]string{}
	}
	translations["en"]["test.greeting"] = "Hello, {{name}}!"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tt(tt.lang, tt.key, tt.params)
			for _, sub := range tt.mustContain {
				if !contains(got, sub) {
					t.Errorf("Tt(%q,%q,%v) = %q; expected to contain %q", tt.lang, tt.key, tt.params, got, sub)
				}
			}
			for _, sub := range tt.mustNotContain {
				if contains(got, sub) {
					t.Errorf("Tt(%q,%q,%v) = %q; expected NOT to contain %q", tt.lang, tt.key, tt.params, got, sub)
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
