package crypto

import (
	"bytes"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key, err := AESKey(256)
	if err != nil {
		t.Fatalf("AESKey(256) error: %v", err)
	}
	return key
}

func TestAESKey(t *testing.T) {
	for _, bits := range []int{128, 192, 256} {
		key, err := AESKey(bits)
		if err != nil {
			t.Fatalf("AESKey(%d) error: %v", bits, err)
		}
		if len(key) != bits/8 {
			t.Fatalf("AESKey(%d) returned %d bytes, want %d", bits, len(key), bits/8)
		}
	}
}

func TestAESKeyInvalidSize(t *testing.T) {
	for _, bits := range []int{0, 64, 127, 255, 512} {
		_, err := AESKey(bits)
		if err == nil {
			t.Fatalf("AESKey(%d) should fail", bits)
		}
	}
}

func TestAESKeyUniqueness(t *testing.T) {
	k1, _ := AESKey(256)
	k2, _ := AESKey(256)
	if bytes.Equal(k1, k2) {
		t.Fatal("AESKey should generate unique keys")
	}
}

func TestAESGCMEncryptAndDecrypt(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("hello, world!")

	ct, err := AESGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("AESGCMEncrypt() error: %v", err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	pt, err := AESGCMDecrypt(key, ct)
	if err != nil {
		t.Fatalf("AESGCMDecrypt() error: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("AESGCMDecrypt() got %q, want %q", pt, plaintext)
	}
}

func TestAESGCMEncryptStringAndDecryptString(t *testing.T) {
	key := testKey(t)
	original := "sensitive data 你好"

	ct, err := AESGCMEncryptString(key, original)
	if err != nil {
		t.Fatalf("AESGCMEncryptString() error: %v", err)
	}
	if ct == original {
		t.Fatal("ciphertext should differ from plaintext")
	}

	pt, err := AESGCMDecryptString(key, ct)
	if err != nil {
		t.Fatalf("AESGCMDecryptString() error: %v", err)
	}
	if pt != original {
		t.Fatalf("AESGCMDecryptString() got %q, want %q", pt, original)
	}
}

func TestWrongKey(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)

	ct, _ := AESGCMEncrypt(key1, []byte("secret"))
	_, err := AESGCMDecrypt(key2, ct)
	if err == nil {
		t.Fatal("AESGCMDecrypt() should fail with wrong key")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	for _, n := range []int{0, 15, 16, 24, 31, 33, 64} {
		key := make([]byte, n)
		_, err := AESGCMEncrypt(key, []byte("test"))
		if n == 16 || n == 24 || n == 32 {
			// AES accepts 16, 24, 32 byte keys
			if err != nil {
				t.Errorf("AESGCMEncrypt() with %d-byte key should work: %v", n, err)
			}
		} else {
			if err == nil {
				t.Errorf("AESGCMEncrypt() with %d-byte key should fail", n)
			}
		}
	}
}

func TestEmptyPlaintext(t *testing.T) {
	key := testKey(t)
	ct, err := AESGCMEncrypt(key, []byte{})
	if err != nil {
		t.Fatalf("AESGCMEncrypt() empty plaintext error: %v", err)
	}
	pt, err := AESGCMDecrypt(key, ct)
	if err != nil {
		t.Fatalf("AESGCMDecrypt() empty plaintext error: %v", err)
	}
	if len(pt) != 0 {
		t.Fatalf("expected empty plaintext, got %d bytes", len(pt))
	}
}

func TestCiphertextTooShort(t *testing.T) {
	key := testKey(t)
	_, err := AESGCMDecrypt(key, []byte("short"))
	if err == nil {
		t.Fatal("AESGCMDecrypt() should fail for short ciphertext")
	}
}

func TestTamperedCiphertext(t *testing.T) {
	key := testKey(t)
	ct, _ := AESGCMEncrypt(key, []byte("data"))

	// Flip a byte
	ct[len(ct)-1] ^= 0xFF
	_, err := AESGCMDecrypt(key, ct)
	if err == nil {
		t.Fatal("AESGCMDecrypt() should fail for tampered ciphertext")
	}
}

func TestDifferentCiphertextEachTime(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("same input")

	ct1, _ := AESGCMEncrypt(key, plaintext)
	ct2, _ := AESGCMEncrypt(key, plaintext)
	if bytes.Equal(ct1, ct2) {
		t.Fatal("same plaintext should produce different ciphertext (random nonce)")
	}

	// Both should decrypt to the same plaintext
	pt1, _ := AESGCMDecrypt(key, ct1)
	pt2, _ := AESGCMDecrypt(key, ct2)
	if !bytes.Equal(pt1, pt2) {
		t.Fatal("both ciphertexts should decrypt to the same plaintext")
	}
}

func TestInvalidBase64(t *testing.T) {
	key := testKey(t)
	_, err := AESGCMDecryptString(key, "not-valid-base64!!!")
	if err == nil {
		t.Fatal("AESGCMDecryptString() should fail for invalid base64")
	}
}
