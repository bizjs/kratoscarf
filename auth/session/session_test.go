package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// 1. NewManager — default values applied
// ---------------------------------------------------------------------------

func TestNewManager_Defaults(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})

	if m.maxAge != 24*time.Hour {
		t.Fatalf("expected default maxAge 24h, got %v", m.maxAge)
	}
	if m.cookieName != "session_id" {
		t.Fatalf("expected default cookieName 'session_id', got %q", m.cookieName)
	}
	if m.cookiePath != "/" {
		t.Fatalf("expected default cookiePath '/', got %q", m.cookiePath)
	}
	if m.httpOnly != true {
		t.Fatalf("expected default httpOnly true, got %v", m.httpOnly)
	}
	if m.sameSite != http.SameSiteLaxMode {
		t.Fatalf("expected default sameSite Lax, got %v", m.sameSite)
	}
}

func TestNewManager_ConfigOverrides(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{
		MaxAge:     2 * time.Hour,
		CookieName: "my_sess",
		CookiePath: "/app",
		Domain:     "example.com",
		Secure:     true,
		HTTPOnly:   true,
		SameSite:   "strict",
	})

	if m.maxAge != 2*time.Hour {
		t.Fatalf("expected maxAge 2h, got %v", m.maxAge)
	}
	if m.cookieName != "my_sess" {
		t.Fatalf("expected cookieName 'my_sess', got %q", m.cookieName)
	}
	if m.cookiePath != "/app" {
		t.Fatalf("expected cookiePath '/app', got %q", m.cookiePath)
	}
	if m.domain != "example.com" {
		t.Fatalf("expected domain 'example.com', got %q", m.domain)
	}
	if !m.secure {
		t.Fatal("expected secure true")
	}
	if m.sameSite != http.SameSiteStrictMode {
		t.Fatalf("expected sameSite Strict, got %v", m.sameSite)
	}
}

func TestNewManager_SameSiteNone(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{SameSite: "none"})
	if m.sameSite != http.SameSiteNoneMode {
		t.Fatalf("expected sameSite None, got %v", m.sameSite)
	}
}

func TestNewManager_WithOptions(t *testing.T) {
	store := NewMemoryStore()
	called := false
	customGen := func() string { called = true; return "custom-id" }

	m := NewManager(store, Config{},
		WithMaxAge(30*time.Minute),
		WithCookieName("opt_sess"),
		WithCookiePath("/v2"),
		WithCookieDomain("opts.dev"),
		WithCookieSecure(true),
		WithCookieHTTPOnly(false),
		WithCookieSameSite(http.SameSiteStrictMode),
		WithIDGenerator(customGen),
	)

	if m.maxAge != 30*time.Minute {
		t.Fatalf("option maxAge not applied")
	}
	if m.cookieName != "opt_sess" {
		t.Fatalf("option cookieName not applied")
	}
	if m.cookiePath != "/v2" {
		t.Fatalf("option cookiePath not applied")
	}
	if m.domain != "opts.dev" {
		t.Fatalf("option domain not applied")
	}
	if !m.secure {
		t.Fatal("option secure not applied")
	}
	if m.httpOnly {
		t.Fatal("option httpOnly not applied")
	}
	if m.sameSite != http.SameSiteStrictMode {
		t.Fatal("option sameSite not applied")
	}

	// verify custom generator is wired
	_ = m.genID()
	if !called {
		t.Fatal("custom ID generator was not invoked")
	}
}

// ---------------------------------------------------------------------------
// 2. Session lifecycle — GetSession / SaveSession / DestroySession
// ---------------------------------------------------------------------------

func TestSessionLifecycle(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})
	ctx := context.Background()

	// --- GetSession with no cookie creates a new session ---
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, err := m.GetSession(ctx, req)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if !sess.IsNew {
		t.Fatal("expected new session")
	}
	if sess.ID == "" {
		t.Fatal("session ID should not be empty")
	}

	sess.Set("user", "alice")

	// --- SaveSession persists the session ---
	w := httptest.NewRecorder()
	if err := m.SaveSession(ctx, w, sess); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}

	// --- Subsequent GetSession with cookie retrieves the same session ---
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	sess2, err := m.GetSession(ctx, req2)
	if err != nil {
		t.Fatalf("GetSession (2nd) error: %v", err)
	}
	if sess2.IsNew {
		t.Fatal("expected existing session, got IsNew=true")
	}
	if sess2.ID != sess.ID {
		t.Fatalf("session ID mismatch: %q vs %q", sess2.ID, sess.ID)
	}
	v, ok := sess2.Get("user")
	if !ok || v != "alice" {
		t.Fatalf("expected user=alice, got %v (ok=%v)", v, ok)
	}

	// --- DestroySession removes the session ---
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	w3 := httptest.NewRecorder()
	if err := m.DestroySession(ctx, w3, req3); err != nil {
		t.Fatalf("DestroySession error: %v", err)
	}

	// --- After destroy, GetSession with same cookie creates a new session ---
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	sess4, err := m.GetSession(ctx, req4)
	if err != nil {
		t.Fatalf("GetSession (post-destroy) error: %v", err)
	}
	if !sess4.IsNew {
		t.Fatal("expected new session after destroy")
	}
	if sess4.ID == sess.ID {
		t.Fatal("new session should have a different ID")
	}
}

// ---------------------------------------------------------------------------
// 3. Session values — Set / Get / Delete
// ---------------------------------------------------------------------------

func TestSessionValues(t *testing.T) {
	sess := &Session{
		ID:     "test-id",
		Values: make(map[string]any),
	}

	// Get on missing key returns false
	_, ok := sess.Get("foo")
	if ok {
		t.Fatal("expected ok=false for missing key")
	}

	// Set stores the value
	sess.Set("foo", 42)
	v, ok := sess.Get("foo")
	if !ok {
		t.Fatal("expected ok=true after Set")
	}
	if v != 42 {
		t.Fatalf("expected 42, got %v", v)
	}

	// Overwrite
	sess.Set("foo", "bar")
	v2, _ := sess.Get("foo")
	if v2 != "bar" {
		t.Fatalf("expected 'bar' after overwrite, got %v", v2)
	}

	// Delete removes the key
	sess.Delete("foo")
	_, ok = sess.Get("foo")
	if ok {
		t.Fatal("expected ok=false after Delete")
	}

	// Delete on non-existent key doesn't panic
	sess.Delete("nonexistent")
}

// ---------------------------------------------------------------------------
// 4. MemoryStore — Get / Save / Delete / expiry
// ---------------------------------------------------------------------------

func TestMemoryStore_GetSaveDelete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Get on nonexistent key returns nil
	sess, err := store.Get(ctx, "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Fatal("expected nil for missing session")
	}

	// Save then Get
	s := &Session{
		ID:        "abc123",
		Values:    map[string]any{"role": "admin"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(ctx, s); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	got, err := store.Get(ctx, "abc123")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.ID != "abc123" {
		t.Fatalf("ID mismatch: %q", got.ID)
	}
	role, _ := got.Get("role")
	if role != "admin" {
		t.Fatalf("expected role=admin, got %v", role)
	}

	// Delete then Get
	if err := store.Delete(ctx, "abc123"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	got2, err := store.Get(ctx, "abc123")
	if err != nil {
		t.Fatalf("Get after Delete error: %v", err)
	}
	if got2 != nil {
		t.Fatal("expected nil after Delete")
	}
}

func TestMemoryStore_Expiry(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Save a session that is already expired
	s := &Session{
		ID:        "expired-sess",
		Values:    map[string]any{},
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if err := store.Save(ctx, s); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Get should return nil for expired session
	got, err := store.Get(ctx, "expired-sess")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for expired session")
	}
}

// ---------------------------------------------------------------------------
// 5. generateID — 64-char hex, unique
// ---------------------------------------------------------------------------

func TestGenerateID_Format(t *testing.T) {
	id := generateID()
	if len(id) != 64 {
		t.Fatalf("expected 64-char hex string, got length %d: %q", len(id), id)
	}
	// Verify it's valid hex
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("invalid hex character %c in ID %q", c, id)
		}
	}
}

func TestGenerateID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := generateID()
		if seen[id] {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}

// ---------------------------------------------------------------------------
// 6. Context helpers — ContextWithSession / FromContext roundtrip
// ---------------------------------------------------------------------------

func TestContextRoundtrip(t *testing.T) {
	sess := &Session{
		ID:     "ctx-test",
		Values: map[string]any{"k": "v"},
	}
	ctx := ContextWithSession(context.Background(), sess)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("expected session from context, got nil")
	}
	if got.ID != "ctx-test" {
		t.Fatalf("ID mismatch: %q", got.ID)
	}
	v, ok := got.Get("k")
	if !ok || v != "v" {
		t.Fatalf("expected k=v, got %v (ok=%v)", v, ok)
	}
}

func TestFromContext_NoSession(t *testing.T) {
	got := FromContext(context.Background())
	if got != nil {
		t.Fatalf("expected nil from empty context, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// 7. Cookie handling — SaveSession sets cookie, DestroySession clears it
// ---------------------------------------------------------------------------

func TestSaveSession_SetsCookie(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{
		CookieName: "my_sid",
		CookiePath: "/app",
		Domain:     "example.com",
		Secure:     true,
		HTTPOnly:   true,
		SameSite:   "strict",
	})
	ctx := context.Background()

	sess := &Session{
		ID:        "cookie-test-id",
		Values:    make(map[string]any),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	w := httptest.NewRecorder()
	if err := m.SaveSession(ctx, w, sess); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected at least one cookie set")
	}

	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == "my_sid" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("cookie 'my_sid' not found")
	}
	if found.Value != "cookie-test-id" {
		t.Fatalf("cookie value mismatch: %q", found.Value)
	}
	if found.Path != "/app" {
		t.Fatalf("cookie path mismatch: %q", found.Path)
	}
	if found.Domain != "example.com" {
		t.Fatalf("cookie domain mismatch: %q", found.Domain)
	}
	if !found.Secure {
		t.Fatal("expected cookie Secure=true")
	}
	if !found.HttpOnly {
		t.Fatal("expected cookie HttpOnly=true")
	}
	if found.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie SameSite mismatch: %v", found.SameSite)
	}
	expectedMaxAge := int((24 * time.Hour).Seconds()) // default maxAge
	if found.MaxAge != expectedMaxAge {
		t.Fatalf("cookie MaxAge mismatch: expected %d, got %d", expectedMaxAge, found.MaxAge)
	}
}

func TestDestroySession_ClearsCookie(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{CookieName: "sid"})
	ctx := context.Background()

	// First save a session so the store has data
	sess := &Session{
		ID:        "destroy-me",
		Values:    make(map[string]any),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	_ = store.Save(ctx, sess)

	// Destroy
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "destroy-me"})
	w := httptest.NewRecorder()
	if err := m.DestroySession(ctx, w, req); err != nil {
		t.Fatalf("DestroySession error: %v", err)
	}

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == "sid" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected cookie 'sid' to be set for clearing")
	}
	if found.Value != "" {
		t.Fatalf("expected empty cookie value, got %q", found.Value)
	}
	if found.MaxAge != -1 {
		t.Fatalf("expected MaxAge=-1 for clearing, got %d", found.MaxAge)
	}

	// Verify store no longer has the session
	got, _ := store.Get(ctx, "destroy-me")
	if got != nil {
		t.Fatal("session should be removed from store after destroy")
	}
}

func TestDestroySession_NoCookie(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})
	ctx := context.Background()

	// Destroy with no cookie should still succeed and set clearing cookie
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	if err := m.DestroySession(ctx, w, req); err != nil {
		t.Fatalf("DestroySession (no cookie) error: %v", err)
	}

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_id" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected clearing cookie even when no session cookie present")
	}
	if found.MaxAge != -1 {
		t.Fatalf("expected MaxAge=-1, got %d", found.MaxAge)
	}
}

// ---------------------------------------------------------------------------
// 8. Session.Modified flag
// ---------------------------------------------------------------------------

func TestSession_ModifiedFlag(t *testing.T) {
	sess := &Session{
		ID:     "mod-test",
		Values: make(map[string]any),
	}

	// Initially not modified
	if sess.Modified {
		t.Fatal("new session should not be marked modified")
	}

	// Set marks as modified
	sess.Set("key", "value")
	if !sess.Modified {
		t.Fatal("Set should mark session as modified")
	}

	// Reset and test Delete
	sess.Modified = false
	sess.Delete("key")
	if !sess.Modified {
		t.Fatal("Delete should mark session as modified")
	}

	// Get does NOT set modified
	sess.Modified = false
	sess.Get("key")
	if sess.Modified {
		t.Fatal("Get should not mark session as modified")
	}
}

func TestGetSession_NewSession_IsModified(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, err := m.GetSession(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if !sess.Modified {
		t.Fatal("new session from GetSession should be marked Modified")
	}
	if !sess.IsNew {
		t.Fatal("new session from GetSession should be marked IsNew")
	}
}

func TestGetSession_ExistingSession_NotModified(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})
	ctx := context.Background()

	// Create and save a session
	sess := &Session{
		ID:        "existing",
		Values:    map[string]any{"x": 1},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	_ = store.Save(ctx, sess)

	// Retrieve it via cookie
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "existing"})
	sess2, err := m.GetSession(ctx, req)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if sess2.Modified {
		t.Fatal("existing session should not be marked modified on retrieval")
	}
	if sess2.IsNew {
		t.Fatal("existing session should not be marked IsNew")
	}
}

// ---------------------------------------------------------------------------
// Additional edge cases
// ---------------------------------------------------------------------------

func TestGetSession_InvalidCookieID_CreatesNew(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "nonexistent-id"})

	sess, err := m.GetSession(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if !sess.IsNew {
		t.Fatal("expected new session when cookie references nonexistent ID")
	}
	if sess.ID == "nonexistent-id" {
		t.Fatal("new session should not reuse the invalid ID")
	}
}

func TestMemoryStore_DeleteNonexistent(t *testing.T) {
	store := NewMemoryStore()
	// Deleting a nonexistent key should not error
	err := store.Delete(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("Delete nonexistent should not error, got: %v", err)
	}
}

func TestSaveSession_DefaultCookieName(t *testing.T) {
	store := NewMemoryStore()
	m := NewManager(store, Config{})
	ctx := context.Background()

	sess := &Session{
		ID:        "default-cookie-test",
		Values:    make(map[string]any),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	w := httptest.NewRecorder()
	_ = m.SaveSession(ctx, w, sess)

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_id" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected default cookie name 'session_id'")
	}
	if found.Path != "/" {
		t.Fatalf("expected default cookie path '/', got %q", found.Path)
	}
}
