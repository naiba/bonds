// Package secret provides AES-256-GCM encryption for sensitive system
// settings. The cipher is keyed off SETTINGS_ENC_KEY (env-only, never
// persisted) so that a stolen database backup is not enough to recover
// plaintext secrets. When no key is configured the Cipher behaves as a
// no-op, preserving backwards-compatibility for existing deployments.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

// CiphertextPrefix identifies values produced by Encrypt. The version segment
// allows us to introduce new schemes without breaking existing rows.
const CiphertextPrefix = "enc:v1:"

var (
	ErrEncryption = errors.New("encryption failed")
	ErrDecryption = errors.New("decryption failed")
)

type Cipher struct {
	key []byte
}

// New returns a Cipher that encrypts when keyMaterial is non-empty and acts
// as a passthrough otherwise. keyMaterial of any length is accepted and
// stretched to 32 bytes via SHA-256.
func New(keyMaterial string) *Cipher {
	if keyMaterial == "" {
		return &Cipher{}
	}
	hash := sha256.Sum256([]byte(keyMaterial))
	return &Cipher{key: hash[:]}
}

func (c *Cipher) Enabled() bool { return len(c.key) > 0 }

// IsCiphertext reports whether v was produced by Encrypt. Callers use this
// to distinguish legacy plaintext rows from encrypted ones during read /
// migration paths.
func IsCiphertext(v string) bool {
	return strings.HasPrefix(v, CiphertextPrefix)
}

// Encrypt produces an opaque, prefix-tagged ciphertext. When the cipher is
// disabled (no key configured) the plaintext is returned unchanged so the
// caller can transparently support both modes.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if !c.Enabled() {
		return plaintext, nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", ErrEncryption
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrEncryption
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrEncryption
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return CiphertextPrefix + hex.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt. Plaintext values (without the prefix) are
// returned as-is, which lets a freshly-keyed deployment read pre-existing
// rows that were never encrypted.
func (c *Cipher) Decrypt(stored string) (string, error) {
	if !IsCiphertext(stored) {
		return stored, nil
	}
	if !c.Enabled() {
		return "", ErrDecryption
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(stored, CiphertextPrefix))
	if err != nil {
		return "", ErrDecryption
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", ErrDecryption
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrDecryption
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", ErrDecryption
	}
	nonce, ct := raw[:nonceSize], raw[nonceSize:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrDecryption
	}
	return string(pt), nil
}
