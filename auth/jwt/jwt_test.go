package jwt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	kratosmiddleware "github.com/go-kratos/kratos/v2/middleware"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	gojwt "github.com/golang-jwt/jwt/v5"
)

// ---------------------------------------------------------------------------
// In-memory TokenStore for tests
// ---------------------------------------------------------------------------

type memoryStore struct {
	mu    sync.RWMutex
	items map[string]time.Time // token -> expiry
}

func newMemoryStore() *memoryStore {
	return &memoryStore{items: make(map[string]time.Time)}
}

func (s *memoryStore) Store(_ context.Context, token string, exp time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[token] = time.Now().Add(exp)
	return nil
}

func (s *memoryStore) Exists(_ context.Context, token string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	expiry, ok := s.items[token]
	if !ok {
		return false, nil
	}
	return time.Now().Before(expiry), nil
}

func (s *memoryStore) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, token)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testSecret = "this-is-a-32-byte-secret-key!!!!" // exactly 32 bytes

func validConfig() Config {
	return Config{
		Secret:        testSecret,
		Issuer:        "test-issuer",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
}

func mustNew(t *testing.T, cfg Config, opts ...Option) *Authenticator {
	t.Helper()
	a, err := New(cfg, opts...)
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	return a
}

// ---------------------------------------------------------------------------
// 1. New()
// ---------------------------------------------------------------------------

func TestNew_SecretTooShort(t *testing.T) {
	_, err := New(Config{Secret: "short"})
	if err == nil {
		t.Fatal("expected error for short secret, got nil")
	}
}

func TestNew_SecretExactlyMinLength(t *testing.T) {
	_, err := New(Config{Secret: testSecret}) // 32 bytes
	if err != nil {
		t.Fatalf("expected no error for 32-byte secret, got: %v", err)
	}
}

func TestNew_DefaultExpiry(t *testing.T) {
	a := mustNew(t, Config{Secret: testSecret})

	if a.cfg.AccessExpiry != 2*time.Hour {
		t.Fatalf("expected default AccessExpiry 2h, got %v", a.cfg.AccessExpiry)
	}
	if a.cfg.RefreshExpiry != 168*time.Hour {
		t.Fatalf("expected default RefreshExpiry 168h, got %v", a.cfg.RefreshExpiry)
	}
}

func TestNew_CustomExpiry(t *testing.T) {
	cfg := validConfig()
	a := mustNew(t, cfg)

	if a.cfg.AccessExpiry != 15*time.Minute {
		t.Fatalf("expected AccessExpiry 15m, got %v", a.cfg.AccessExpiry)
	}
	if a.cfg.RefreshExpiry != 24*time.Hour {
		t.Fatalf("expected RefreshExpiry 24h, got %v", a.cfg.RefreshExpiry)
	}
}

func TestNew_WithTokenStore(t *testing.T) {
	store := newMemoryStore()
	a := mustNew(t, validConfig(), WithTokenStore(store))

	if a.tokenStore == nil {
		t.Fatal("expected tokenStore to be set")
	}
}

func TestNew_WithSigningMethod(t *testing.T) {
	a := mustNew(t, validConfig(), WithSigningMethod(gojwt.SigningMethodHS384))

	if a.signingMethod.Alg() != "HS384" {
		t.Fatalf("expected HS384, got %s", a.signingMethod.Alg())
	}
}

func TestNew_ConfigSigningMethod(t *testing.T) {
	cfg := validConfig()
	cfg.SigningMethod = "HS512"
	a := mustNew(t, cfg)

	if a.signingMethod.Alg() != "HS512" {
		t.Fatalf("expected HS512, got %s", a.signingMethod.Alg())
	}
}

// ---------------------------------------------------------------------------
// 2. GenerateTokenPair
// ---------------------------------------------------------------------------

func TestGenerateTokenPair_Success(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()

	pair, err := a.GenerateTokenPair(ctx, Claims{UserID: "user-1", Username: "alice"})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access and refresh tokens should differ")
	}
	if pair.ExpiresAt.IsZero() {
		t.Fatal("expected non-zero ExpiresAt")
	}
}

func TestGenerateTokenPair_TokensAreParseable(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()

	pair, err := a.GenerateTokenPair(ctx, Claims{
		UserID:   "user-2",
		Username: "bob",
		Roles:    []string{"admin", "editor"},
		Extra:    map[string]string{"org": "acme"},
	})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error: %v", err)
	}

	// Parse access token
	claims, err := a.ValidateToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken(access) error: %v", err)
	}
	if claims.UserID != "user-2" {
		t.Fatalf("expected UserID user-2, got %q", claims.UserID)
	}
	if claims.Username != "bob" {
		t.Fatalf("expected Username bob, got %q", claims.Username)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "admin" {
		t.Fatalf("unexpected Roles: %v", claims.Roles)
	}
	if claims.Extra["org"] != "acme" {
		t.Fatalf("expected Extra[org]=acme, got %q", claims.Extra["org"])
	}
	if claims.Issuer != "test-issuer" {
		t.Fatalf("expected Issuer test-issuer, got %q", claims.Issuer)
	}

	// Parse refresh token
	refreshClaims, err := a.ValidateToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateToken(refresh) error: %v", err)
	}
	if refreshClaims.UserID != "user-2" {
		t.Fatalf("expected refresh UserID user-2, got %q", refreshClaims.UserID)
	}
}

func TestGenerateTokenPair_ExpiresAtMatchesConfig(t *testing.T) {
	cfg := validConfig()
	cfg.AccessExpiry = 30 * time.Minute
	a := mustNew(t, cfg)

	before := time.Now()
	pair, err := a.GenerateTokenPair(context.Background(), Claims{UserID: "u1"})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error: %v", err)
	}

	expectedMin := before.Add(30 * time.Minute)
	expectedMax := time.Now().Add(30 * time.Minute)

	if pair.ExpiresAt.Before(expectedMin) || pair.ExpiresAt.After(expectedMax) {
		t.Fatalf("ExpiresAt %v not within expected range [%v, %v]", pair.ExpiresAt, expectedMin, expectedMax)
	}
}

// ---------------------------------------------------------------------------
// 3. ValidateToken
// ---------------------------------------------------------------------------

func TestValidateToken_ValidToken(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})
	claims, err := a.ValidateToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("expected UserID user-1, got %q", claims.UserID)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	cfg := validConfig()
	cfg.AccessExpiry = 1 * time.Millisecond // almost immediate expiry
	a := mustNew(t, cfg)
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})
	time.Sleep(10 * time.Millisecond) // wait for expiry

	_, err := a.ValidateToken(ctx, pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	// Create token with HS256
	a256 := mustNew(t, validConfig())
	ctx := context.Background()
	pair, _ := a256.GenerateTokenPair(ctx, Claims{UserID: "user-1"})

	// Validate with authenticator expecting HS384
	a384 := mustNew(t, validConfig(), WithSigningMethod(gojwt.SigningMethodHS384))
	_, err := a384.ValidateToken(ctx, pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for wrong signing method")
	}
}

func TestValidateToken_TamperedToken(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})

	// Tamper with the token by modifying a character in the signature
	tampered := pair.AccessToken[:len(pair.AccessToken)-4] + "XXXX"

	_, err := a.ValidateToken(ctx, tampered)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestValidateToken_CompletelyBogusToken(t *testing.T) {
	a := mustNew(t, validConfig())
	_, err := a.ValidateToken(context.Background(), "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for bogus token")
	}
}

func TestValidateToken_RevokedToken(t *testing.T) {
	store := newMemoryStore()
	a := mustNew(t, validConfig(), WithTokenStore(store))
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})

	// Revoke the token
	if err := a.RevokeToken(ctx, pair.AccessToken); err != nil {
		t.Fatalf("RevokeToken() error: %v", err)
	}

	// Should now fail validation
	_, err := a.ValidateToken(ctx, pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
}

func TestValidateToken_DifferentSecret(t *testing.T) {
	a1 := mustNew(t, validConfig())
	pair, _ := a1.GenerateTokenPair(context.Background(), Claims{UserID: "u1"})

	cfg2 := validConfig()
	cfg2.Secret = "a-completely-different-secret-key!"
	a2 := mustNew(t, cfg2)

	_, err := a2.ValidateToken(context.Background(), pair.AccessToken)
	if err == nil {
		t.Fatal("expected error when validating with different secret")
	}
}

// ---------------------------------------------------------------------------
// 4. RefreshToken
// ---------------------------------------------------------------------------

func TestRefreshToken_Success(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()

	original, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1", Username: "alice"})

	newPair, err := a.RefreshToken(ctx, original.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}
	if newPair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if newPair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if newPair.AccessToken == original.AccessToken {
		t.Fatal("new access token should differ from original")
	}

	// Verify new access token is valid
	claims, err := a.ValidateToken(ctx, newPair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken(new access) error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("expected UserID user-1, got %q", claims.UserID)
	}
}

func TestRefreshToken_OldTokenRevokedWithStore(t *testing.T) {
	store := newMemoryStore()
	a := mustNew(t, validConfig(), WithTokenStore(store))
	ctx := context.Background()

	original, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})

	_, err := a.RefreshToken(ctx, original.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}

	// Old refresh token should be revoked
	revoked, err := store.Exists(ctx, original.RefreshToken)
	if err != nil {
		t.Fatalf("store.Exists() error: %v", err)
	}
	if !revoked {
		t.Fatal("expected old refresh token to be revoked")
	}

	// Attempting to use old refresh token should fail
	_, err = a.RefreshToken(ctx, original.RefreshToken)
	if err == nil {
		t.Fatal("expected error when reusing revoked refresh token")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	a := mustNew(t, validConfig())
	_, err := a.RefreshToken(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

// ---------------------------------------------------------------------------
// 5. RevokeToken
// ---------------------------------------------------------------------------

func TestRevokeToken_WithStore(t *testing.T) {
	store := newMemoryStore()
	a := mustNew(t, validConfig(), WithTokenStore(store))
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})
	if err := a.RevokeToken(ctx, pair.AccessToken); err != nil {
		t.Fatalf("RevokeToken() error: %v", err)
	}

	revoked, err := store.Exists(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("store.Exists() error: %v", err)
	}
	if !revoked {
		t.Fatal("expected token to be marked revoked")
	}
}

func TestRevokeToken_WithoutStore(t *testing.T) {
	a := mustNew(t, validConfig()) // no token store
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1"})
	err := a.RevokeToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("expected nil error without store, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 6. Extractors
// ---------------------------------------------------------------------------

func TestBearerExtractor_Valid(t *testing.T) {
	ext := BearerExtractor()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-token-123")

	token, err := ext.Extract(req)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if token != "my-token-123" {
		t.Fatalf("expected my-token-123, got %q", token)
	}
}

func TestBearerExtractor_CaseInsensitive(t *testing.T) {
	ext := BearerExtractor()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "bearer my-token")

	token, err := ext.Extract(req)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if token != "my-token" {
		t.Fatalf("expected my-token, got %q", token)
	}
}

func TestBearerExtractor_MissingHeader(t *testing.T) {
	ext := BearerExtractor()
	req := httptest.NewRequest("GET", "/", nil)

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken, got: %v", err)
	}
}

func TestBearerExtractor_InvalidFormat(t *testing.T) {
	ext := BearerExtractor()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrInvalidAuthHeader) {
		t.Fatalf("expected ErrInvalidAuthHeader, got: %v", err)
	}
}

func TestBearerExtractor_EmptyToken(t *testing.T) {
	ext := BearerExtractor()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer   ")

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken for empty bearer value, got: %v", err)
	}
}

func TestCookieExtractor_Valid(t *testing.T) {
	ext := CookieExtractor("access_token")
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token-abc"})

	token, err := ext.Extract(req)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if token != "cookie-token-abc" {
		t.Fatalf("expected cookie-token-abc, got %q", token)
	}
}

func TestCookieExtractor_Missing(t *testing.T) {
	ext := CookieExtractor("access_token")
	req := httptest.NewRequest("GET", "/", nil)

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken, got: %v", err)
	}
}

func TestCookieExtractor_EmptyValue(t *testing.T) {
	ext := CookieExtractor("access_token")
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: ""})

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken for empty cookie, got: %v", err)
	}
}

func TestQueryExtractor_Valid(t *testing.T) {
	ext := QueryExtractor("token")
	req := httptest.NewRequest("GET", "/?token=query-tok-xyz", nil)

	token, err := ext.Extract(req)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if token != "query-tok-xyz" {
		t.Fatalf("expected query-tok-xyz, got %q", token)
	}
}

func TestQueryExtractor_Missing(t *testing.T) {
	ext := QueryExtractor("token")
	req := httptest.NewRequest("GET", "/", nil)

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken, got: %v", err)
	}
}

func TestChainExtractor_FirstSucceeds(t *testing.T) {
	ext := ChainExtractor(
		BearerExtractor(),
		CookieExtractor("token"),
	)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})

	token, err := ext.Extract(req)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if token != "header-token" {
		t.Fatalf("expected header-token (first), got %q", token)
	}
}

func TestChainExtractor_FallsThrough(t *testing.T) {
	ext := ChainExtractor(
		BearerExtractor(),
		CookieExtractor("token"),
		QueryExtractor("token"),
	)
	// No Authorization header, no cookie, but query param is present
	req := httptest.NewRequest("GET", "/?token=query-fallback", nil)

	token, err := ext.Extract(req)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	if token != "query-fallback" {
		t.Fatalf("expected query-fallback, got %q", token)
	}
}

func TestChainExtractor_AllFail(t *testing.T) {
	ext := ChainExtractor(
		BearerExtractor(),
		CookieExtractor("token"),
	)
	req := httptest.NewRequest("GET", "/", nil)

	_, err := ext.Extract(req)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected ErrNoToken when all extractors fail, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7. Middleware (integration via Kratos HTTP server)
//
// Kratos server-level middleware (kratoshttp.Middleware()) only applies to
// proto-registered service handlers. For Route().GET() handlers, middleware
// must be invoked within the route handler — the same pattern used by
// kratoscarf/router. We use this pattern here to test the JWT middleware
// in a realistic integration setup.
// ---------------------------------------------------------------------------

// wrapWithMiddleware registers a route on the Kratos server that manually
// invokes the given Kratos middleware chain before calling the handler.
// This mirrors how kratoscarf/router applies middleware to route handlers.
func wrapWithMiddleware(
	srv *kratoshttp.Server,
	method, path string,
	mw kratosmiddleware.Middleware,
	handler func(ctx context.Context) error,
) {
	srv.Route("/").Handle(method, path, func(ctx kratoshttp.Context) error {
		var inner kratosmiddleware.Handler = func(mwCtx context.Context, _ any) (any, error) {
			return nil, handler(mwCtx)
		}
		chain := mw(inner)
		_, err := chain(ctx, nil)
		if err != nil {
			return err
		}
		return nil
	})
}

func TestMiddleware_SkipPaths(t *testing.T) {
	a := mustNew(t, validConfig())
	mw := Middleware(a, WithSkipPaths("/login", "/health"))

	srv := kratoshttp.NewServer()

	for _, p := range []string{"/login", "/health"} {
		wrapWithMiddleware(srv, "GET", p, mw, func(ctx context.Context) error {
			return nil
		})
	}

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/login", 200},
		{"/health", 200},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", tt.path, nil)
		srv.ServeHTTP(w, req)

		if w.Code != tt.wantStatus {
			t.Errorf("path %s: expected %d, got %d (body: %s)", tt.path, tt.wantStatus, w.Code, w.Body.String())
		}
	}
}

func TestMiddleware_SkipPathNormalization(t *testing.T) {
	a := mustNew(t, validConfig())
	// Register "/login/" which should be normalized to "/login"
	mw := Middleware(a, WithSkipPaths("/login/"))

	srv := kratoshttp.NewServer()
	wrapWithMiddleware(srv, "GET", "/login", mw, func(ctx context.Context) error {
		return nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for normalized skip path, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestMiddleware_ValidTokenInjectsClaims(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()

	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-1", Username: "alice"})

	var gotClaims *Claims
	mw := Middleware(a)

	srv := kratoshttp.NewServer()
	wrapWithMiddleware(srv, "GET", "/me", mw, func(ctx context.Context) error {
		gotClaims = ClaimsFromContext(ctx)
		return nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if gotClaims == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if gotClaims.UserID != "user-1" {
		t.Fatalf("expected UserID user-1, got %q", gotClaims.UserID)
	}
	if gotClaims.Username != "alice" {
		t.Fatalf("expected Username alice, got %q", gotClaims.Username)
	}
}

func TestMiddleware_MissingTokenReturnsError(t *testing.T) {
	a := mustNew(t, validConfig())
	mw := Middleware(a)

	handlerCalled := false
	srv := kratoshttp.NewServer()
	wrapWithMiddleware(srv, "GET", "/protected", mw, func(ctx context.Context) error {
		handlerCalled = true
		return nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	srv.ServeHTTP(w, req)

	if handlerCalled {
		t.Fatal("handler should not be called when token is missing")
	}
}

func TestMiddleware_InvalidTokenReturnsError(t *testing.T) {
	a := mustNew(t, validConfig())
	mw := Middleware(a)

	handlerCalled := false
	srv := kratoshttp.NewServer()
	wrapWithMiddleware(srv, "GET", "/protected", mw, func(ctx context.Context) error {
		handlerCalled = true
		return nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	srv.ServeHTTP(w, req)

	if handlerCalled {
		t.Fatal("handler should not be called when token is invalid")
	}
}

func TestMiddleware_WithCustomExtractor(t *testing.T) {
	a := mustNew(t, validConfig())
	ctx := context.Background()
	pair, _ := a.GenerateTokenPair(ctx, Claims{UserID: "user-q"})

	var gotClaims *Claims
	mw := Middleware(a, WithExtractor(QueryExtractor("access_token")))

	srv := kratoshttp.NewServer()
	wrapWithMiddleware(srv, "GET", "/api", mw, func(ctx context.Context) error {
		gotClaims = ClaimsFromContext(ctx)
		return nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api?access_token="+pair.AccessToken, nil)
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if gotClaims == nil {
		t.Fatal("expected claims from query extractor")
	}
	if gotClaims.UserID != "user-q" {
		t.Fatalf("expected UserID user-q, got %q", gotClaims.UserID)
	}
}

// ---------------------------------------------------------------------------
// 8. Context helpers
// ---------------------------------------------------------------------------

func TestContextWithClaims_Roundtrip(t *testing.T) {
	original := &Claims{
		UserID:   "user-42",
		Username: "charlie",
		Roles:    []string{"admin"},
	}

	ctx := ContextWithClaims(context.Background(), original)
	got := ClaimsFromContext(ctx)

	if got == nil {
		t.Fatal("expected claims from context, got nil")
	}
	if got.UserID != "user-42" {
		t.Fatalf("expected UserID user-42, got %q", got.UserID)
	}
	if got.Username != "charlie" {
		t.Fatalf("expected Username charlie, got %q", got.Username)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "admin" {
		t.Fatalf("unexpected Roles: %v", got.Roles)
	}
}

func TestClaimsFromContext_Empty(t *testing.T) {
	got := ClaimsFromContext(context.Background())
	if got != nil {
		t.Fatalf("expected nil from empty context, got %+v", got)
	}
}

func TestMustClaimsFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from MustClaimsFromContext on empty context")
		}
	}()
	MustClaimsFromContext(context.Background())
}

func TestMustClaimsFromContext_Success(t *testing.T) {
	original := &Claims{UserID: "user-99"}
	ctx := ContextWithClaims(context.Background(), original)

	got := MustClaimsFromContext(ctx)
	if got.UserID != "user-99" {
		t.Fatalf("expected UserID user-99, got %q", got.UserID)
	}
}

// ---------------------------------------------------------------------------
// cleanPath
// ---------------------------------------------------------------------------

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/login/", "/login"},
		{"/login", "/login"},
		{"login", "/login"},
		{"/api//v1/", "/api/v1"},
		{"", "/"},
	}
	for _, tt := range tests {
		if got := cleanPath(tt.input); got != tt.want {
			t.Errorf("cleanPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
