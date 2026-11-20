package config

import "testing"

// Outside production the built-in secret is accepted so existing development
// and self-hosted setups keep starting. It is public, though, so the server
// has to be able to tell and warn that anyone can forge sessions.
func TestUsesBuiltInJWTSecret(t *testing.T) {
	cases := map[string]bool{
		"change-me-in-production":          true,
		" change-me-in-production ":        true,
		"a-long-random-secret-of-my-own-1": false,
	}
	for secret, want := range cases {
		cfg := &Config{JWT: JWTConfig{Secret: secret}}
		if got := cfg.UsesBuiltInJWTSecret(); got != want {
			t.Errorf("UsesBuiltInJWTSecret(%q) = %v, want %v", secret, got, want)
		}
	}
}
