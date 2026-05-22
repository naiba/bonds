package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
)

// TestUpdateLocaleRejectsUnsupported guards the gap that lets a user persist
// e.g. "de" into the locale column, which the i18n bundle cannot honor — the
// UI then silently falls back to English and the saved preference appears to
// do nothing. Only locales listed in i18n.Supported are accepted; anything
// else (including empty, made-up codes, region-only variants the bundle
// doesn't carry) returns ErrUnsupportedLocale.
func TestUpdateLocaleRejectsUnsupported(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	tests := []struct {
		name      string
		locale    string
		wantError bool
	}{
		{"english", "en", false},
		{"chinese", "zh", false},
		{"spanish", "es", false},
		{"german_unsupported", "de", true},
		{"french_unsupported", "fr", true},
		{"japanese_unsupported", "ja", true},
		{"empty", "", true},
		{"region_only", "zh-CN", true},
		{"garbage", "xx", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdateLocale(userID, dto.UpdateLocaleRequest{Locale: tt.locale})
			if tt.wantError {
				if err == nil {
					t.Fatalf("UpdateLocale(%q) expected error, got nil", tt.locale)
				}
				if !errors.Is(err, ErrUnsupportedLocale) {
					t.Fatalf("UpdateLocale(%q) expected ErrUnsupportedLocale, got %v", tt.locale, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateLocale(%q) unexpected error: %v", tt.locale, err)
			}
			prefs, err := svc.Get(userID)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if prefs.Locale != tt.locale {
				t.Fatalf("locale not persisted: want %q got %q", tt.locale, prefs.Locale)
			}
		})
	}
}

// TestUpdateAllRejectsUnsupportedLocale mirrors the guard for the combined
// UpdateAll endpoint, which previously silently accepted unsupported codes
// because it only checked req.Locale != "" before writing.
func TestUpdateAllRejectsUnsupportedLocale(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	_, err := svc.UpdateAll(userID, dto.UpdatePreferencesRequest{
		Locale: "de",
	})
	if err == nil || !errors.Is(err, ErrUnsupportedLocale) {
		t.Fatalf("UpdateAll(locale=de) expected ErrUnsupportedLocale, got %v", err)
	}
}
