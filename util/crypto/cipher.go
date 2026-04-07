// Package crypto provides cryptographic utilities (encryption, hashing, etc.).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// AESKey generates a random AES key of the given bit size (128, 192, or 256).
func AESKey(bits int) ([]byte, error) {
	switch bits {
	case 128, 192, 256:
	default:
		return nil, fmt.Errorf("crypto: invalid AES key size %d, must be 128, 192, or 256", bits)
	}
	key := make([]byte, bits/8)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// AESGCMEncrypt encrypts plaintext using AES-GCM. Key must be 16, 24, or 32 bytes.
func AESGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	aead, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// AESGCMDecrypt decrypts ciphertext using AES-GCM.
func AESGCMDecrypt(key, ciphertext []byte) ([]byte, error) {
	aead, err := newAESGCM(key)
	if err != nil {
		return nil, err
	}
	ns := aead.NonceSize()
	if len(ciphertext) <= ns {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}
	return aead.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
}

// AESGCMEncryptString encrypts a string and returns base64-encoded ciphertext.
func AESGCMEncryptString(key []byte, plaintext string) (string, error) {
	ct, err := AESGCMEncrypt(key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

// AESGCMDecryptString decrypts a base64-encoded ciphertext string.
func AESGCMDecryptString(key []byte, ciphertext string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	pt, err := AESGCMDecrypt(key, ct)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func newAESGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}
	return cipher.NewGCM(block)
}
