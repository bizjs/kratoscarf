package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// SHA256 returns the hex-encoded SHA-256 hash of data.
func SHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HmacSHA256Key generates a random 32-byte key for use with HmacSHA256.
func HmacSHA256Key() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// HmacSHA256 returns the hex-encoded HMAC-SHA256 of data with the given key.
func HmacSHA256(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// BcryptHash hashes a password using bcrypt with the default cost (10).
func BcryptHash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// BcryptHashWithCost hashes a password using bcrypt with a custom cost.
func BcryptHashWithCost(password string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(b), err
}

// BcryptVerify compares a bcrypt hash with a plaintext password. Returns nil on match.
func BcryptVerify(hashed, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}
