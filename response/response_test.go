package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kratosErrors "github.com/go-kratos/kratos/v2/errors"
)

// ---------------------------------------------------------------------------
// 1. Success / Wrap / SuccessWithMessage
// ---------------------------------------------------------------------------

func TestSuccess(t *testing.T) {
	data := map[string]string{"hello": "world"}
	resp := Success(data)

	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got %q", resp.Message)
	}
	if resp.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
}

func TestSuccessNilData(t *testing.T) {
	resp := Success(nil)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data != nil {
		t.Errorf("expected nil data, got %v", resp.Data)
	}
}

func TestSuccessWithMessage(t *testing.T) {
	resp := SuccessWithMessage("payload", "created")
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "created" {
		t.Errorf("expected message 'created', got %q", resp.Message)
	}
	if resp.Data != "payload" {
		t.Errorf("expected data 'payload', got %v", resp.Data)
	}
}

func TestWrap(t *testing.T) {
	data := []int{1, 2, 3}
	result := Wrap(data)

	resp, ok := result.(*Response)
	if !ok {
		t.Fatalf("Wrap should return *Response, got %T", result)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got %q", resp.Message)
	}
}

func TestSuccessJSON(t *testing.T) {
	resp := Success(map[string]int{"count": 42})
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if _, ok := m["code"]; !ok {
		t.Error("json output missing 'code' field")
	}
	if _, ok := m["message"]; !ok {
		t.Error("json output missing 'message' field")
	}
	if _, ok := m["data"]; !ok {
		t.Error("json output missing 'data' field")
	}
}

func TestSuccessNilDataOmitsDataInJSON(t *testing.T) {
	resp := Success(nil)
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if _, ok := m["data"]; ok {
		t.Error("expected 'data' to be omitted for nil, but it was present")
	}
}

// ---------------------------------------------------------------------------
// 2. Error function
// ---------------------------------------------------------------------------

func TestErrorWithBizError(t *testing.T) {
	err := ErrNotFound.WithMessage("user not found")
	resp := Error(err)

	if resp.Code != ErrNotFound.Code {
		t.Errorf("expected code %d, got %d", ErrNotFound.Code, resp.Code)
	}
	if resp.Message != "user not found" {
		t.Errorf("expected message 'user not found', got %q", resp.Message)
	}
}

func TestErrorWithNil(t *testing.T) {
	resp := Error(nil)
	if resp.Code != 0 {
		t.Errorf("expected code 0 for nil error, got %d", resp.Code)
	}
}

func TestErrorWithPlainError(t *testing.T) {
	resp := Error(errors.New("something broke"))
	if resp.Code != ErrInternal.Code {
		t.Errorf("expected internal error code %d, got %d", ErrInternal.Code, resp.Code)
	}
}

// ---------------------------------------------------------------------------
// 3. BizError
// ---------------------------------------------------------------------------

func TestNewBizError(t *testing.T) {
	be := NewBizError(http.StatusTeapot, 41800, "i'm a teapot")
	if be.HTTPCode != http.StatusTeapot {
		t.Errorf("expected HTTP %d, got %d", http.StatusTeapot, be.HTTPCode)
	}
	if be.Code != 41800 {
		t.Errorf("expected biz code 41800, got %d", be.Code)
	}
	if be.Message != "i'm a teapot" {
		t.Errorf("expected message 'i'm a teapot', got %q", be.Message)
	}
}

func TestBizErrorErrorString(t *testing.T) {
	be := NewBizError(400, 40000, "bad request")
	if be.Error() != "bad request" {
		t.Errorf("expected 'bad request', got %q", be.Error())
	}
}

func TestBizErrorErrorStringWithCause(t *testing.T) {
	cause := errors.New("missing field")
	be := NewBizError(400, 40000, "bad request").WithCause(cause)
	expected := "bad request: missing field"
	if be.Error() != expected {
		t.Errorf("expected %q, got %q", expected, be.Error())
	}
}

func TestBizErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	be := NewBizError(500, 50000, "fail").WithCause(cause)

	if !errors.Is(be, cause) {
		t.Error("Unwrap should allow errors.Is to find the cause")
	}
}

// ---------------------------------------------------------------------------
// 4. WithMessage / WithCause create new instances (no mutation)
// ---------------------------------------------------------------------------

func TestWithMessageDoesNotMutateOriginal(t *testing.T) {
	original := NewBizError(404, 40400, "not found")
	derived := original.WithMessage("user not found")

	if original.Message != "not found" {
		t.Errorf("original message mutated: got %q", original.Message)
	}
	if derived.Message != "user not found" {
		t.Errorf("derived message wrong: got %q", derived.Message)
	}
	if original == derived {
		t.Error("WithMessage should return a new pointer, not the same one")
	}
}

func TestWithCauseDoesNotMutateOriginal(t *testing.T) {
	original := NewBizError(500, 50000, "internal")
	cause := errors.New("db timeout")
	derived := original.WithCause(cause)

	if original.cause != nil {
		t.Error("original cause should remain nil")
	}
	if !errors.Is(derived.cause, cause) {
		t.Error("derived cause should be set")
	}
	if original == derived {
		t.Error("WithCause should return a new pointer, not the same one")
	}
}

func TestWithMessageChaining(t *testing.T) {
	cause := errors.New("timeout")
	be := ErrInternal.WithMessage("db error").WithCause(cause)

	if be.Message != "db error" {
		t.Errorf("expected 'db error', got %q", be.Message)
	}
	if !errors.Is(be.cause, cause) {
		t.Error("expected cause to be set")
	}
	expected := "db error: timeout"
	if be.Error() != expected {
		t.Errorf("expected %q, got %q", expected, be.Error())
	}
}

// ---------------------------------------------------------------------------
// 5. IsBizError
// ---------------------------------------------------------------------------

func TestIsBizError(t *testing.T) {
	be := ErrNotFound.WithMessage("item missing")
	found, ok := IsBizError(be)
	if !ok {
		t.Fatal("expected IsBizError to return true")
	}
	if found.Code != ErrNotFound.Code {
		t.Errorf("expected code %d, got %d", ErrNotFound.Code, found.Code)
	}
}

func TestIsBizErrorWrapped(t *testing.T) {
	be := ErrNotFound.WithMessage("wrapped")
	wrapped := fmt.Errorf("outer: %w", be)

	found, ok := IsBizError(wrapped)
	if !ok {
		t.Fatal("expected IsBizError to find wrapped BizError")
	}
	if found.Code != ErrNotFound.Code {
		t.Errorf("expected code %d, got %d", ErrNotFound.Code, found.Code)
	}
}

func TestIsBizErrorReturnsFalse(t *testing.T) {
	_, ok := IsBizError(errors.New("plain error"))
	if ok {
		t.Error("expected IsBizError to return false for plain error")
	}
}

// ---------------------------------------------------------------------------
// 6. Predefined errors
// ---------------------------------------------------------------------------

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *BizError
		httpCode int
		bizCode  int
	}{
		{"ErrBadRequest", ErrBadRequest, 400, 40000},
		{"ErrUnauthorized", ErrUnauthorized, 401, 40100},
		{"ErrForbidden", ErrForbidden, 403, 40300},
		{"ErrNotFound", ErrNotFound, 404, 40400},
		{"ErrConflict", ErrConflict, 409, 40900},
		{"ErrValidation", ErrValidation, 422, 42200},
		{"ErrTooManyRequests", ErrTooManyRequests, 429, 42900},
		{"ErrInternal", ErrInternal, 500, 50000},
		{"ErrServiceUnavailable", ErrServiceUnavailable, 503, 50300},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.HTTPCode != tc.httpCode {
				t.Errorf("expected HTTP %d, got %d", tc.httpCode, tc.err.HTTPCode)
			}
			if tc.err.Code != tc.bizCode {
				t.Errorf("expected biz code %d, got %d", tc.bizCode, tc.err.Code)
			}
			if tc.err.Message == "" {
				t.Error("expected non-empty message")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 7. FromKratosError
// ---------------------------------------------------------------------------

func TestFromKratosError(t *testing.T) {
	ke := kratosErrors.New(http.StatusNotFound, "NOT_FOUND", "resource not found")
	be := FromKratosError(ke)

	if be.HTTPCode != http.StatusNotFound {
		t.Errorf("expected HTTP %d, got %d", http.StatusNotFound, be.HTTPCode)
	}
	if be.Code != http.StatusNotFound*100 {
		t.Errorf("expected biz code %d, got %d", http.StatusNotFound*100, be.Code)
	}
	if be.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", be.Message)
	}
	if be.cause == nil {
		t.Error("expected cause to be set")
	}
}

func TestFromKratosErrorNil(t *testing.T) {
	be := FromKratosError(nil)
	if be != nil {
		t.Error("expected nil for nil input")
	}
}

func TestFromKratosErrorPlainError(t *testing.T) {
	plain := errors.New("something failed")
	be := FromKratosError(plain)

	if be.HTTPCode != http.StatusInternalServerError {
		t.Errorf("expected HTTP 500, got %d", be.HTTPCode)
	}
	if be.Code != 50000 {
		t.Errorf("expected biz code 50000, got %d", be.Code)
	}
	if be.Message != "something failed" {
		t.Errorf("expected message 'something failed', got %q", be.Message)
	}
}

// ---------------------------------------------------------------------------
// 8. Duck typing — HTTPStatus / BizCode / ErrorData
// ---------------------------------------------------------------------------

// duckBizCodeError implements BizCode() only.
type duckBizCodeError struct {
	code int
}

func (e *duckBizCodeError) Error() string { return "biz code error" }
func (e *duckBizCodeError) BizCode() int  { return e.code }

// duckHTTPStatusError implements HTTPStatus() only.
type duckHTTPStatusError struct {
	status int
}

func (e *duckHTTPStatusError) Error() string   { return "http status error" }
func (e *duckHTTPStatusError) HTTPStatus() int { return e.status }

// duckDataError implements ErrorData() only.
type duckDataError struct {
	data any
}

func (e *duckDataError) Error() string  { return "error data error" }
func (e *duckDataError) ErrorData() any { return e.data }

// duckFullError implements all three duck-type interfaces.
type duckFullError struct {
	status int
	code   int
	data   any
}

func (e *duckFullError) Error() string   { return "full duck error" }
func (e *duckFullError) HTTPStatus() int { return e.status }
func (e *duckFullError) BizCode() int    { return e.code }
func (e *duckFullError) ErrorData() any  { return e.data }

func TestErrorToResponseWithBizCode(t *testing.T) {
	err := &duckBizCodeError{code: 42200}
	resp := errorToResponse(err)

	if resp.Code != 42200 {
		t.Errorf("expected code 42200, got %d", resp.Code)
	}
}

func TestErrorToResponseWithErrorData(t *testing.T) {
	fieldErrors := []string{"name is required", "email is invalid"}
	err := &duckDataError{data: fieldErrors}
	resp := errorToResponse(err)

	if resp.Data == nil {
		t.Fatal("expected data to be set")
	}
	if resp.Message != "validation failed" {
		t.Errorf("expected 'validation failed' when ErrorData present, got %q", resp.Message)
	}
}

func TestErrorToResponseWithBizCodeAndErrorData(t *testing.T) {
	err := &duckFullError{status: 422, code: 42200, data: map[string]string{"field": "required"}}
	resp := errorToResponse(err)

	if resp.Code != 42200 {
		t.Errorf("expected code 42200, got %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("expected data to be set")
	}
}

// ---------------------------------------------------------------------------
// 9. HTTP Encoders
// ---------------------------------------------------------------------------

func TestHTTPResponseEncoder(t *testing.T) {
	encoder := NewHTTPResponseEncoder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	payload := map[string]string{"key": "value"}
	if err := encoder(w, r, payload); err != nil {
		t.Fatalf("encoder returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected json content type, got %q", ct)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got %q", resp.Message)
	}
}

func TestHTTPErrorEncoderBizError(t *testing.T) {
	encoder := NewHTTPErrorEncoder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	bizErr := ErrNotFound.WithMessage("item missing")
	encoder(w, r, bizErr)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if resp.Code != ErrNotFound.Code {
		t.Errorf("expected code %d, got %d", ErrNotFound.Code, resp.Code)
	}
	if resp.Message != "item missing" {
		t.Errorf("expected 'item missing', got %q", resp.Message)
	}
}

func TestHTTPErrorEncoderDuckHTTPStatus(t *testing.T) {
	encoder := NewHTTPErrorEncoder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	err := &duckHTTPStatusError{status: http.StatusServiceUnavailable}
	encoder(w, r, err)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestHTTPErrorEncoderDuckFull(t *testing.T) {
	encoder := NewHTTPErrorEncoder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	err := &duckFullError{
		status: http.StatusUnprocessableEntity,
		code:   42200,
		data:   []string{"field error"},
	}
	encoder(w, r, err)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", w.Code)
	}

	var resp Response
	if jsonErr := json.Unmarshal(w.Body.Bytes(), &resp); jsonErr != nil {
		t.Fatalf("json unmarshal failed: %v", jsonErr)
	}
	if resp.Code != 42200 {
		t.Errorf("expected code 42200, got %d", resp.Code)
	}
}

func TestHTTPErrorEncoderPlainError(t *testing.T) {
	encoder := NewHTTPErrorEncoder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	encoder(w, r, errors.New("unknown"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if resp.Code != ErrInternal.Code {
		t.Errorf("expected code %d, got %d", ErrInternal.Code, resp.Code)
	}
}

// ---------------------------------------------------------------------------
// 10. Pagination — PageRequest
// ---------------------------------------------------------------------------

func TestPageRequestNormalize(t *testing.T) {
	tests := []struct {
		name         string
		req          PageRequest
		defaultSize  int
		maxSize      int
		expectedPage int
		expectedSize int
	}{
		{"zero values", PageRequest{}, 20, 100, 1, 20},
		{"page < 1", PageRequest{Page: -1}, 20, 100, 1, 20},
		{"size exceeds max", PageRequest{Page: 1, PageSize: 200}, 20, 100, 1, 100},
		{"valid values", PageRequest{Page: 3, PageSize: 50}, 20, 100, 3, 50},
		{"size zero uses default", PageRequest{Page: 2, PageSize: 0}, 10, 50, 2, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.Normalize(tc.defaultSize, tc.maxSize)
			if tc.req.Page != tc.expectedPage {
				t.Errorf("page: expected %d, got %d", tc.expectedPage, tc.req.Page)
			}
			if tc.req.PageSize != tc.expectedSize {
				t.Errorf("pageSize: expected %d, got %d", tc.expectedSize, tc.req.PageSize)
			}
		})
	}
}

func TestPageRequestOffset(t *testing.T) {
	tests := []struct {
		name     string
		req      PageRequest
		expected int
	}{
		{"page 1", PageRequest{Page: 1, PageSize: 10}, 0},
		{"page 3", PageRequest{Page: 3, PageSize: 10}, 20},
		{"page < 1 returns 0", PageRequest{Page: 0, PageSize: 10}, 0},
		{"page negative returns 0", PageRequest{Page: -1, PageSize: 10}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.req.Offset(); got != tc.expected {
				t.Errorf("expected offset %d, got %d", tc.expected, got)
			}
		})
	}
}

func TestPageRequestSort(t *testing.T) {
	req := PageRequest{Page: 1, PageSize: 10, Sort: "created_at desc"}
	if req.Sort != "created_at desc" {
		t.Errorf("expected sort 'created_at desc', got %q", req.Sort)
	}
}

// ---------------------------------------------------------------------------
// 11. Pagination — PageResponse
// ---------------------------------------------------------------------------

func TestNewPageResponse(t *testing.T) {
	items := []string{"a", "b", "c"}
	req := PageRequest{Page: 2, PageSize: 10}
	resp := NewPageResponse(items, 30, req)

	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}
	if resp.Total != 30 {
		t.Errorf("expected total 30, got %d", resp.Total)
	}
	if resp.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Page)
	}
	if resp.PageSize != 10 {
		t.Errorf("expected pageSize 10, got %d", resp.PageSize)
	}
}

func TestNewPageResponseNilItems(t *testing.T) {
	req := PageRequest{Page: 1, PageSize: 10}
	resp := NewPageResponse[string](nil, 0, req)

	if resp.Items == nil {
		t.Error("expected non-nil items slice (empty, not nil)")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestPageResponseJSON(t *testing.T) {
	items := []int{1, 2}
	req := PageRequest{Page: 1, PageSize: 10}
	resp := NewPageResponse(items, 100, req)

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	for _, key := range []string{"items", "total", "page", "pageSize"} {
		if _, ok := m[key]; !ok {
			t.Errorf("json output missing %q field", key)
		}
	}
}

// ---------------------------------------------------------------------------
// 12. Pagination — CursorRequest / CursorResponse
// ---------------------------------------------------------------------------

func TestCursorRequest(t *testing.T) {
	req := CursorRequest{Cursor: "abc123", Limit: 25}
	if req.Cursor != "abc123" {
		t.Errorf("expected cursor 'abc123', got %q", req.Cursor)
	}
	if req.Limit != 25 {
		t.Errorf("expected limit 25, got %d", req.Limit)
	}
}

func TestNewCursorResponse(t *testing.T) {
	items := []string{"x", "y"}
	resp := NewCursorResponse(items, "next_cursor_val", true)

	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.NextCursor != "next_cursor_val" {
		t.Errorf("expected next cursor 'next_cursor_val', got %q", resp.NextCursor)
	}
	if !resp.HasMore {
		t.Error("expected HasMore to be true")
	}
}

func TestNewCursorResponseNilItems(t *testing.T) {
	resp := NewCursorResponse[int](nil, "", false)

	if resp.Items == nil {
		t.Error("expected non-nil items slice (empty, not nil)")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestNewCursorResponseNoMore(t *testing.T) {
	items := []string{"last"}
	resp := NewCursorResponse(items, "", false)

	if resp.NextCursor != "" {
		t.Errorf("expected empty next cursor, got %q", resp.NextCursor)
	}
	if resp.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestCursorResponseJSON(t *testing.T) {
	resp := NewCursorResponse([]string{"a"}, "cur", true)
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	for _, key := range []string{"items", "nextCursor", "hasMore"} {
		if _, ok := m[key]; !ok {
			t.Errorf("json output missing %q field", key)
		}
	}
}

func TestCursorResponseJSONOmitsEmptyNextCursor(t *testing.T) {
	resp := NewCursorResponse([]string{"a"}, "", false)
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if _, ok := m["nextCursor"]; ok {
		t.Error("expected 'nextCursor' to be omitted when empty, but it was present")
	}
}

// ---------------------------------------------------------------------------
// 13. BizError Data field
// ---------------------------------------------------------------------------

func TestBizErrorWithData(t *testing.T) {
	be := NewBizError(422, 42200, "validation failed")
	be.Data = map[string]string{"field": "name", "error": "required"}

	resp := errorToResponse(be)
	if resp.Data == nil {
		t.Fatal("expected data to be set from BizError.Data")
	}
	if resp.Code != 42200 {
		t.Errorf("expected code 42200, got %d", resp.Code)
	}
}
