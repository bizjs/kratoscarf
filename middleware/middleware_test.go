package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- CORS ---

func TestCORS_DefaultAllowsAll(t *testing.T) {
	handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected *, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for preflight")
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/users", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	handler.ServeHTTP(w, req)

	if w.Code >= 400 {
		t.Fatalf("expected success status, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Allow-Methods header")
	}
}

func TestCORS_WithCredentials(t *testing.T) {
	handler := CORS(
		WithAllowOrigins("https://example.com"),
		WithAllowCredentials(),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials true, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected specific origin, got %q", got)
	}
}

// --- Secure ---

func TestSecure_Defaults(t *testing.T) {
	handler := Secure(SecureConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, req)

	tests := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for header, expected := range tests {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("%s: expected %q, got %q", header, expected, got)
		}
	}
	// HSTS and Permissions-Policy should not be set by default
	for _, header := range []string{"Strict-Transport-Security", "Permissions-Policy"} {
		if got := w.Header().Get(header); got != "" {
			t.Errorf("%s should not be set by default, got %q", header, got)
		}
	}
}

func TestSecure_HSTS(t *testing.T) {
	handler := Secure(SecureConfig{
		HSTSMaxAge:            63072000,
		HSTSIncludeSubDomains: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, req)

	expected := "max-age=63072000; includeSubDomains"
	if got := w.Header().Get("Strict-Transport-Security"); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSecure_CSP(t *testing.T) {
	handler := Secure(SecureConfig{
		ContentSecurityPolicy: "default-src 'self'",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
		t.Fatalf("expected CSP, got %q", got)
	}
}

// --- RequestID ---
// Note: RequestID is a Kratos middleware, not an HTTP handler.
// Full integration test requires Kratos transport context.
// We test RequestIDFromContext in isolation.

func TestRequestIDFromContext_Empty(t *testing.T) {
	if got := RequestIDFromContext(t.Context()); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
