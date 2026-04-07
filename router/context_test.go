package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestContext(r *http.Request) *Context {
	return &Context{
		request:  r,
		response: httptest.NewRecorder(),
	}
}

func recorder(c *Context) *httptest.ResponseRecorder {
	return c.response.(*httptest.ResponseRecorder)
}

// --- P0: ClientIP ---

func TestClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")
	ctx := newTestContext(req)

	if got := ctx.ClientIP(); got != "203.0.113.50" {
		t.Fatalf("expected 203.0.113.50, got %q", got)
	}
}

func TestClientIP_XForwardedFor_Single(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	ctx := newTestContext(req)

	if got := ctx.ClientIP(); got != "203.0.113.50" {
		t.Fatalf("expected 203.0.113.50, got %q", got)
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-Ip", "10.0.0.1")
	ctx := newTestContext(req)

	if got := ctx.ClientIP(); got != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %q", got)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	ctx := newTestContext(req)

	if got := ctx.ClientIP(); got != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %q", got)
	}
}

func TestClientIP_IPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:8080"
	ctx := newTestContext(req)

	if got := ctx.ClientIP(); got != "::1" {
		t.Fatalf("expected ::1, got %q", got)
	}
}

// --- P0: QueryArray ---

func TestQueryArray(t *testing.T) {
	req := httptest.NewRequest("GET", "/?ids=1&ids=2&ids=3", nil)
	ctx := newTestContext(req)

	ids := ctx.QueryArray("ids")
	if len(ids) != 3 {
		t.Fatalf("expected 3 values, got %d", len(ids))
	}
	if ids[0] != "1" || ids[1] != "2" || ids[2] != "3" {
		t.Fatalf("unexpected values: %v", ids)
	}
}

func TestQueryArray_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := newTestContext(req)

	if got := ctx.QueryArray("missing"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// --- P1: String ---

func TestString(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := newTestContext(req)

	ctx.String(200, "hello %s", "world")
	w := recorder(ctx)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "hello world" {
		t.Fatalf("expected 'hello world', got %q", got)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content-type: %q", ct)
	}
}

func TestString_NoFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := newTestContext(req)

	ctx.String(200, "plain text")
	if got := recorder(ctx).Body.String(); got != "plain text" {
		t.Fatalf("expected 'plain text', got %q", got)
	}
}

// --- P1: Data ---

func TestData(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := newTestContext(req)

	ctx.Data(200, "image/png", []byte{0x89, 0x50, 0x4E, 0x47})
	w := recorder(ctx)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("unexpected content-type: %q", ct)
	}
	if w.Body.Len() != 4 {
		t.Fatalf("expected 4 bytes, got %d", w.Body.Len())
	}
}

// --- P1: File / Attachment ---

func TestFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmp, []byte("file content"), 0644)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := newTestContext(req)

	ctx.File(tmp)
	if got := recorder(ctx).Body.String(); got != "file content" {
		t.Fatalf("expected 'file content', got %q", got)
	}
}

func TestAttachment(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "report.pdf")
	os.WriteFile(tmp, []byte("pdf data"), 0644)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := newTestContext(req)

	ctx.Attachment(tmp, "monthly-report.pdf")
	w := recorder(ctx)

	cd := w.Header().Get("Content-Disposition")
	if cd == "" {
		t.Fatal("expected Content-Disposition header")
	}
	if got := w.Body.String(); got != "pdf data" {
		t.Fatalf("expected 'pdf data', got %q", got)
	}
}

// --- P2: ContentType / QueryString / QueryValues / FormValue ---

func TestContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	ctx := newTestContext(req)

	if got := ctx.ContentType(); got != "application/json" {
		t.Fatalf("expected 'application/json', got %q", got)
	}
}

func TestQueryString(t *testing.T) {
	req := httptest.NewRequest("GET", "/?foo=bar&baz=1", nil)
	ctx := newTestContext(req)

	if got := ctx.QueryString(); got != "foo=bar&baz=1" {
		t.Fatalf("expected 'foo=bar&baz=1', got %q", got)
	}
}

func TestQueryValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/?a=1&b=2&a=3", nil)
	ctx := newTestContext(req)

	vals := ctx.QueryValues()
	if len(vals["a"]) != 2 {
		t.Fatalf("expected 2 values for 'a', got %d", len(vals["a"]))
	}
}
