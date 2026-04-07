package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkContext_JSON(b *testing.B) {
	data := map[string]any{"id": 1, "name": "alice", "email": "alice@example.com"}
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		ctx := &Context{response: w}
		_ = ctx.JSON(200, data)
	}
}

func BenchmarkContext_Success_NoWrapper(b *testing.B) {
	data := map[string]string{"status": "ok"}
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		ctx := &Context{response: w}
		_ = ctx.Success(data)
	}
}

func BenchmarkContext_Success_WithWrapper(b *testing.B) {
	wrapper := func(data any) any {
		return map[string]any{"code": 0, "message": "ok", "data": data}
	}
	data := map[string]string{"status": "ok"}
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		ctx := &Context{response: w, wrapper: wrapper}
		_ = ctx.Success(data)
	}
}

func BenchmarkContext_Query(b *testing.B) {
	req := httptest.NewRequest("GET", "/?page=1&size=20&sort=name", nil)
	ctx := &Context{request: req}
	b.ReportAllocs()
	for b.Loop() {
		_ = ctx.Query("page")
		_ = ctx.Query("size")
		_ = ctx.Query("sort")
	}
}

func BenchmarkContext_QueryArray(b *testing.B) {
	req := httptest.NewRequest("GET", "/?ids=1&ids=2&ids=3&ids=4&ids=5", nil)
	ctx := &Context{request: req}
	b.ReportAllocs()
	for b.Loop() {
		_ = ctx.QueryArray("ids")
	}
}

func BenchmarkContext_ClientIP_XForwardedFor(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	ctx := &Context{request: req}
	b.ReportAllocs()
	for b.Loop() {
		_ = ctx.ClientIP()
	}
}

func BenchmarkContext_String(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		ctx := &Context{response: w}
		_ = ctx.String(200, "hello %s, id=%d", "world", 42)
	}
}

func BenchmarkContext_Header(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Request-Id", "abc-def-123")
	ctx := &Context{request: req}
	b.ReportAllocs()
	for b.Loop() {
		_ = ctx.Header("Authorization")
		_ = ctx.Header("X-Request-Id")
	}
}

func BenchmarkContext_SetHeader(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		ctx := &Context{response: w}
		ctx.SetHeader("X-Custom", "value")
		ctx.SetHeader("X-Request-Id", "abc-123")
	}
}

func BenchmarkContext_Redirect(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/old", nil)
		ctx := &Context{request: req, response: w}
		_ = ctx.Redirect(http.StatusFound, "/new")
	}
}
