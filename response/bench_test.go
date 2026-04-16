package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkSuccess(b *testing.B) {
	data := map[string]any{"id": 1, "name": "alice"}
	b.ReportAllocs()
	for b.Loop() {
		_ = Success(data)
	}
}

func BenchmarkWrap(b *testing.B) {
	data := map[string]any{"id": 1, "name": "alice"}
	b.ReportAllocs()
	for b.Loop() {
		_ = Wrap(data)
	}
}

func BenchmarkBizError_WithMessage(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = ErrNotFound.WithMessage("user not found")
	}
}

func BenchmarkBizError_WithCause(b *testing.B) {
	cause := errors.New("sql: no rows")
	b.ReportAllocs()
	for b.Loop() {
		_ = ErrInternal.WithCause(cause)
	}
}

func BenchmarkIsBizError_Direct(b *testing.B) {
	err := ErrNotFound.WithMessage("gone")
	b.ReportAllocs()
	for b.Loop() {
		_, _ = IsBizError(err)
	}
}

func BenchmarkIsBizError_Wrapped(b *testing.B) {
	err := ErrNotFound.WithMessage("gone")
	wrapped := errors.Join(errors.New("context"), err)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = IsBizError(wrapped)
	}
}

func BenchmarkErrorToResponse_BizError(b *testing.B) {
	err := ErrNotFound.WithMessage("user not found")
	b.ReportAllocs()
	for b.Loop() {
		_ = ErrorToResponse(err)
	}
}

func BenchmarkErrorToResponse_DuckTyped(b *testing.B) {
	err := &duckError{status: 422, code: 42200}
	b.ReportAllocs()
	for b.Loop() {
		_ = ErrorToResponse(err)
	}
}

type duckError struct {
	status int
	code   int
}

func (e *duckError) Error() string   { return "duck" }
func (e *duckError) HTTPStatus() int { return e.status }
func (e *duckError) BizCode() int    { return e.code }

func BenchmarkErrorToResponse_PlainError(b *testing.B) {
	err := errors.New("something went wrong")
	b.ReportAllocs()
	for b.Loop() {
		_ = ErrorToResponse(err)
	}
}

func BenchmarkHTTPErrorEncoder(b *testing.B) {
	encoder := NewHTTPErrorEncoder()
	err := ErrNotFound.WithMessage("user not found")
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		encoder(w, r, err)
	}
}

func BenchmarkHTTPResponseEncoder(b *testing.B) {
	encoder := NewHTTPResponseEncoder()
	data := map[string]any{"id": 1, "name": "alice"}
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		_ = encoder(w, r, data)
	}
}

func BenchmarkPageResponse(b *testing.B) {
	items := make([]map[string]any, 20)
	for i := range items {
		items[i] = map[string]any{"id": i, "name": "item"}
	}
	req := PageRequest{Page: 1, PageSize: 20}
	b.ReportAllocs()
	for b.Loop() {
		_ = NewPageResponse(items, 100, req)
	}
}

func BenchmarkFromKratosError(b *testing.B) {
	err := errors.New("upstream timeout")
	b.ReportAllocs()
	for b.Loop() {
		_ = FromKratosError(err)
	}
}

func BenchmarkNewHTTPErrorEncoder_FullPipeline(b *testing.B) {
	encoder := NewHTTPErrorEncoder()
	errs := []error{
		ErrBadRequest.WithMessage("invalid input"),
		ErrNotFound.WithMessage("not found"),
		ErrInternal.WithCause(errors.New("db down")),
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, err := range errs {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			encoder(w, r, err)
		}
	}
}

func BenchmarkNewHTTPResponseEncoder_FullPipeline(b *testing.B) {
	encoder := NewHTTPResponseEncoder()
	payloads := []any{
		map[string]string{"ok": "true"},
		[]int{1, 2, 3},
		NewPageResponse([]string{"a", "b"}, 10, PageRequest{Page: 1, PageSize: 20}),
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, p := range payloads {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			_ = encoder(w, r, p)
		}
	}
}

func BenchmarkHTTPResponseEncoder_CustomWrapper(b *testing.B) {
	encoder := NewHTTPResponseEncoder(WithSuccessWrapper(func(data any) any {
		return map[string]any{"ok": true, "data": data}
	}))
	data := map[string]any{"id": 1, "name": "alice"}
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		_ = encoder(w, r, data)
	}
}

func BenchmarkHTTPErrorEncoder_CustomWrapper(b *testing.B) {
	encoder := NewHTTPErrorEncoder(WithErrorWrapper(func(err error) any {
		return map[string]any{"ok": false, "error": err.Error()}
	}))
	err := ErrNotFound.WithMessage("not found")
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		encoder(w, r, err)
	}
}

// Baseline: raw net/http JSON for comparison.
func BenchmarkBaseline_RawHTTPJSON(b *testing.B) {
	data := map[string]any{"code": 0, "message": "ok", "data": map[string]any{"id": 1, "name": "alice"}}
	b.ReportAllocs()
	for b.Loop() {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(data)
	}
}
