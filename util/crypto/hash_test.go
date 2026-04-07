package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func TestSHA256(t *testing.T) {
	// Known vector: SHA-256 of "hello"
	got := SHA256([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("SHA256(\"hello\") = %s, want %s", got, want)
	}
}

func TestSHA256Empty(t *testing.T) {
	got := SHA256([]byte{})
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("SHA256(\"\") = %s, want %s", got, want)
	}
}

func TestSHA256Deterministic(t *testing.T) {
	data := []byte("same input")
	a, b := SHA256(data), SHA256(data)
	if a != b {
		t.Fatal("SHA256 should be deterministic")
	}
}

func TestHmacSHA256Key(t *testing.T) {
	k1, err := HmacSHA256Key()
	if err != nil {
		t.Fatalf("HmacSHA256Key() error: %v", err)
	}
	if len(k1) != 32 {
		t.Fatalf("HmacSHA256Key() returned %d bytes, want 32", len(k1))
	}
	k2, _ := HmacSHA256Key()
	if bytes.Equal(k1, k2) {
		t.Fatal("HmacSHA256Key should generate unique keys")
	}
}

func TestHmacSHA256(t *testing.T) {
	key := []byte("secret-key")
	data := []byte("hello")
	got := HmacSHA256(key, data)
	if len(got) != 64 {
		t.Fatalf("HmacSHA256 should return 64 hex chars, got %d", len(got))
	}
	if strings.ToLower(got) != got {
		t.Fatal("HmacSHA256 should return lowercase hex")
	}
}

func TestHmacSHA256DifferentKeys(t *testing.T) {
	data := []byte("hello")
	h1 := HmacSHA256([]byte("key1"), data)
	h2 := HmacSHA256([]byte("key2"), data)
	if h1 == h2 {
		t.Fatal("different keys should produce different HMACs")
	}
}

func TestHmacSHA256Deterministic(t *testing.T) {
	key := []byte("key")
	data := []byte("data")
	a, b := HmacSHA256(key, data), HmacSHA256(key, data)
	if a != b {
		t.Fatal("HmacSHA256 should be deterministic")
	}
}

func TestBcryptHashAndVerify(t *testing.T) {
	pwd := "mySecretPassword123"
	hashed, err := BcryptHash(pwd)
	if err != nil {
		t.Fatalf("BcryptHash() error: %v", err)
	}
	if hashed == "" {
		t.Fatal("BcryptHash() returned empty string")
	}
	if hashed == pwd {
		t.Fatal("BcryptHash() returned plaintext")
	}

	// Verify correct password
	if err := BcryptVerify(hashed, pwd); err != nil {
		t.Fatalf("BcryptVerify() failed for correct password: %v", err)
	}

	// Verify wrong password
	if err := BcryptVerify(hashed, "wrong"); err == nil {
		t.Fatal("BcryptVerify() should fail for wrong password")
	}
}

func TestBcryptHashWithCost(t *testing.T) {
	pwd := "test"
	hashed, err := BcryptHashWithCost(pwd, 4) // min cost for speed
	if err != nil {
		t.Fatalf("BcryptHashWithCost() error: %v", err)
	}
	if err := BcryptVerify(hashed, pwd); err != nil {
		t.Fatalf("BcryptVerify() failed: %v", err)
	}
}

func TestDifferentHashesForSameBcryptHash(t *testing.T) {
	pwd := "samePassword"
	h1, _ := BcryptHash(pwd)
	h2, _ := BcryptHash(pwd)
	if h1 == h2 {
		t.Fatal("same password should produce different hashes (random salt)")
	}
	// Both should verify
	if err := BcryptVerify(h1, pwd); err != nil {
		t.Fatalf("Verify h1 failed: %v", err)
	}
	if err := BcryptVerify(h2, pwd); err != nil {
		t.Fatalf("Verify h2 failed: %v", err)
	}
}

func TestEmptyBcryptHash(t *testing.T) {
	hashed, err := BcryptHash("")
	if err != nil {
		t.Fatalf("BcryptHash(\"\") error: %v", err)
	}
	if err := BcryptVerify(hashed, ""); err != nil {
		t.Fatalf("Verify empty password failed: %v", err)
	}
	if err := BcryptVerify(hashed, "notempty"); err == nil {
		t.Fatal("Verify should fail for non-empty against empty hash")
	}
}

func TestUnicodeBcryptHash(t *testing.T) {
	pwd := "密码🔐パスワード"
	hashed, err := BcryptHash(pwd)
	if err != nil {
		t.Fatalf("BcryptHash() error: %v", err)
	}
	if err := BcryptVerify(hashed, pwd); err != nil {
		t.Fatalf("BcryptVerify() failed for unicode password: %v", err)
	}
}
