package id

import (
	"strings"
	"sync"
	"testing"
)

func TestUUID(t *testing.T) {
	id := UUID()
	// UUID v4 format: 8-4-4-4-12
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("UUID() invalid format: %s", id)
	}
}

func TestUUIDv7(t *testing.T) {
	a := UUIDv7()
	b := UUIDv7()
	if a == "" || b == "" {
		t.Fatal("UUIDv7() returned empty string")
	}
	if a == b {
		t.Fatal("UUIDv7() returned duplicate")
	}
	// v7 is time-sortable: a < b lexicographically
	if a > b {
		t.Fatalf("UUIDv7() not time-sorted: %s > %s", a, b)
	}
}

func TestULID(t *testing.T) {
	a := ULID()
	b := ULID()
	if a == b {
		t.Fatal("ULID() returned duplicate")
	}
	// ULID is time-sortable
	if a > b {
		t.Fatalf("ULID() not time-sorted: %s > %s", a, b)
	}
}

func TestShort(t *testing.T) {
	id := Short()
	if len(id) != 12 {
		t.Fatalf("Short() expected 12 chars, got %d", len(id))
	}
	for _, c := range id {
		if !strings.ContainsRune(base62, c) {
			t.Fatalf("Short() contains invalid char: %c", c)
		}
	}
}

func TestShortN(t *testing.T) {
	for _, n := range []int{1, 8, 16, 32, 64} {
		id := ShortN(n)
		if len(id) != n {
			t.Fatalf("ShortN(%d) returned %d chars", n, len(id))
		}
	}
}

func TestUniqueness(t *testing.T) {
	const count = 10000
	seen := make(map[string]bool, count)
	for i := range count {
		id := ULID()
		if seen[id] {
			t.Fatalf("duplicate at iteration %d: %s", i, id)
		}
		seen[id] = true
	}
}

func TestConcurrency(t *testing.T) {
	const goroutines = 100
	const perGoroutine = 100

	ids := make([]string, goroutines*perGoroutine)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func() {
			defer wg.Done()
			for i := range perGoroutine {
				ids[g*perGoroutine+i] = ULID()
			}
		}()
	}
	wg.Wait()

	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" {
			t.Fatal("empty ID from concurrent generation")
		}
		if seen[id] {
			t.Fatalf("duplicate from concurrent generation: %s", id)
		}
		seen[id] = true
	}
}
