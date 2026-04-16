package response

import (
	"encoding/json"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// --- Response encoder options ---

// ResponseEncoderOption configures NewHTTPResponseEncoder.
type ResponseEncoderOption func(*responseEncoderConfig)

type responseEncoderConfig struct {
	wrapper func(data any) any
}

// WithSuccessWrapper overrides the default success response wrapper.
// The signature matches router.ResponseWrapper, so the same function can
// be passed to both router.WithResponseWrapper and this option.
// Default: response.Success(v) → {code: 0, message: "ok", data: v}
func WithSuccessWrapper(w func(data any) any) ResponseEncoderOption {
	return func(c *responseEncoderConfig) { c.wrapper = w }
}

// --- Error encoder options ---

// ErrorEncoderOption configures NewHTTPErrorEncoder.
type ErrorEncoderOption func(*errorEncoderConfig)

type errorEncoderConfig struct {
	wrapper func(err error) any
}

// WithErrorWrapper overrides the default error response wrapper.
// Use ErrorToResponse(err) inside your wrapper to delegate to the default
// duck-typing logic.
// Default: ErrorToResponse(err) → {code: N, message: "...", data: ...}
func WithErrorWrapper(w func(err error) any) ErrorEncoderOption {
	return func(c *errorEncoderConfig) { c.wrapper = w }
}

// NewHTTPResponseEncoder returns a Kratos HTTP ResponseEncoder that wraps
// all responses in the unified Response format.
// Pass WithSuccessWrapper to customize the response body structure.
func NewHTTPResponseEncoder(opts ...ResponseEncoderOption) kratoshttp.EncodeResponseFunc {
	cfg := &responseEncoderConfig{
		wrapper: func(data any) any { return Success(data) },
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(w http.ResponseWriter, r *http.Request, v any) error {
		resp := cfg.wrapper(v)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return json.NewEncoder(w).Encode(resp)
	}
}

// NewHTTPErrorEncoder returns a Kratos HTTP ErrorEncoder that converts
// errors into the unified Response format.
// Pass WithErrorWrapper to customize the response body structure.
//
// HTTP status code is always determined via duck typing (not customizable):
//   - BizError: uses HTTPCode
//   - Any error with HTTPStatus() int: uses that as HTTP status
//   - Other errors: 500
func NewHTTPErrorEncoder(opts ...ErrorEncoderOption) kratoshttp.EncodeErrorFunc {
	cfg := &errorEncoderConfig{
		wrapper: func(err error) any { return ErrorToResponse(err) },
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(w http.ResponseWriter, r *http.Request, err error) {
		resp := cfg.wrapper(err)
		httpCode := http.StatusInternalServerError

		if bizErr, ok := IsBizError(err); ok {
			httpCode = bizErr.HTTPCode
		} else if hs, ok := err.(interface{ HTTPStatus() int }); ok {
			httpCode = hs.HTTPStatus()
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(httpCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// ErrorToResponse converts an error to a *Response, checking duck-typed interfaces.
// Exported so that custom error wrappers (via WithErrorWrapper) can delegate to
// the default duck-typing logic.
func ErrorToResponse(err error) *Response {
	if err == nil {
		return Success(nil)
	}

	// BizError — first-class support.
	if bizErr, ok := IsBizError(err); ok {
		return &Response{
			Code:    bizErr.Code,
			Message: bizErr.Message,
			Data:    bizErr.Data,
		}
	}

	// Duck-typed errors (e.g., validation.ValidationErrors, validation.BindError).
	code := ErrInternal.Code
	message := ErrInternal.Message
	var data any

	if bc, ok := err.(interface{ BizCode() int }); ok {
		code = bc.BizCode()
		message = err.Error()
	}
	if ed, ok := err.(interface{ ErrorData() any }); ok {
		data = ed.ErrorData()
		message = "validation failed" // override message when structured data is present
	}

	return &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
