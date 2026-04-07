// Package id provides ID generation utilities.
package id

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// UUID returns a new UUID v4 string.
func UUID() string {
	return uuid.New().String()
}

// UUIDv7 returns a new time-sortable UUID v7 string.
func UUIDv7() string {
	v, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("id: failed to generate UUIDv7: %v", err))
	}
	return v.String()
}

var gen = &ulidGen{entropy: ulid.Monotonic(rand.Reader, 0)}

type ulidGen struct {
	mu      sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// ULID returns a new ULID string.
func ULID() string {
	gen.mu.Lock()
	defer gen.mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), gen.entropy).String()
}

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Short returns a 12-char URL-safe random ID (base62).
func Short() string {
	return ShortN(12)
}

// ShortN returns an n-char URL-safe random ID (base62). Panics if n < 1.
func ShortN(n int) string {
	if n < 1 {
		panic(fmt.Sprintf("id: ShortN length must be >= 1, got %d", n))
	}
	b := make([]byte, n)
	max := big.NewInt(int64(len(base62)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(fmt.Sprintf("id: crypto/rand failed: %v", err))
		}
		b[i] = base62[idx.Int64()]
	}
	return string(b)
}
