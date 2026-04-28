package secret

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := New("super-secret-key-material")

	got, err := c.Encrypt("hello world")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if !IsCiphertext(got) {
		t.Fatalf("expected ciphertext prefix, got %q", got)
	}
	if got == "hello world" {
		t.Fatal("ciphertext must differ from plaintext")
	}

	pt, err := c.Decrypt(got)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if pt != "hello world" {
		t.Fatalf("round-trip mismatch: got %q", pt)
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	c := New("k")
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("ciphertexts must use random nonces and differ for the same plaintext")
	}
}

func TestDisabledCipherIsPassthrough(t *testing.T) {
	c := New("")
	if c.Enabled() {
		t.Fatal("empty key must yield disabled cipher")
	}

	got, err := c.Encrypt("plain")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if got != "plain" {
		t.Fatalf("disabled Encrypt must passthrough, got %q", got)
	}

	pt, err := c.Decrypt("plain")
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if pt != "plain" {
		t.Fatalf("disabled Decrypt must passthrough, got %q", pt)
	}
}

func TestDecryptPlaintextWithEnabledCipher(t *testing.T) {
	c := New("k")
	got, err := c.Decrypt("legacy-plaintext")
	if err != nil {
		t.Fatalf("Decrypt of legacy plaintext failed: %v", err)
	}
	if got != "legacy-plaintext" {
		t.Fatalf("enabled cipher must read legacy plaintext rows: got %q", got)
	}
}

func TestDecryptCiphertextWithoutKeyFails(t *testing.T) {
	c := New("k")
	ct, _ := c.Encrypt("secret")

	noKey := New("")
	if _, err := noKey.Decrypt(ct); err == nil {
		t.Fatal("decrypting ciphertext with no key must fail")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	c1 := New("alpha")
	c2 := New("beta")
	ct, _ := c1.Encrypt("secret")

	if _, err := c2.Decrypt(ct); err == nil {
		t.Fatal("decryption with wrong key must fail")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	c := New("k")
	ct, _ := c.Encrypt("secret")

	tampered := strings.TrimSuffix(ct, ct[len(ct)-1:]) + "0"
	if _, err := c.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext must fail GCM auth")
	}

	if _, err := c.Decrypt(CiphertextPrefix + "not-hex"); err == nil {
		t.Fatal("non-hex body must fail")
	}

	if _, err := c.Decrypt(CiphertextPrefix + "00"); err == nil {
		t.Fatal("ciphertext shorter than nonce must fail")
	}
}

func TestEncryptEmptyString(t *testing.T) {
	c := New("k")
	ct, err := c.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt('') failed: %v", err)
	}
	if !IsCiphertext(ct) {
		t.Fatalf("encrypted empty value should still be tagged: %q", ct)
	}
	pt, err := c.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if pt != "" {
		t.Fatalf("expected empty plaintext, got %q", pt)
	}
}

func TestIsCiphertext(t *testing.T) {
	if IsCiphertext("plain") {
		t.Error("plain string is not ciphertext")
	}
	if !IsCiphertext(CiphertextPrefix + "abcd") {
		t.Error("prefixed value should be detected as ciphertext")
	}
	if IsCiphertext("") {
		t.Error("empty string is not ciphertext")
	}
}
