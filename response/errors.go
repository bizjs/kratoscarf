package response

import (
	"errors"
	"fmt"
	"net/http"

	kratosErrors "github.com/go-kratos/kratos/v2/errors"
)

// BizError is the standard business error type.
type BizError struct {
	HTTPCode int    // HTTP status code to return
	Code     int    // business error code
	Message  string // default message
	Data     any    // optional structured error data (e.g. validation field errors)
	cause    error  // wrapped error
}

// Error implements the error interface.
func (e *BizError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error.
func (e *BizError) Unwrap() error {
	return e.cause
}

// NewBizError creates a new BizError.
func NewBizError(httpCode, bizCode int, message string) *BizError {
	return &BizError{
		HTTPCode: httpCode,
		Code:     bizCode,
		Message:  message,
	}
}

// WithCause returns a copy of this BizError wrapping an underlying error.
func (e *BizError) WithCause(err error) *BizError {
	clone := *e
	clone.cause = err
	return &clone
}

// WithMessage returns a copy of this BizError with an overridden message.
func (e *BizError) WithMessage(msg string) *BizError {
	clone := *e
	clone.Message = msg
	return &clone
}

// IsBizError checks if an error is a BizError and returns it.
func IsBizError(err error) (*BizError, bool) {
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr, true
	}
	return nil, false
}

// FromKratosError converts a Kratos error to a BizError.
func FromKratosError(err error) *BizError {
	if err == nil {
		return nil
	}
	var se *kratosErrors.Error
	if errors.As(err, &se) {
		return &BizError{
			HTTPCode: int(se.Code),
			Code:     int(se.Code) * 100,
			Message:  se.Message,
			cause:    err,
		}
	}
	return &BizError{
		HTTPCode: http.StatusInternalServerError,
		Code:     50000,
		Message:  err.Error(),
		cause:    err,
	}
}

// Pre-defined common errors.
var (
	ErrBadRequest         = NewBizError(400, 40000, "bad request")
	ErrUnauthorized       = NewBizError(401, 40100, "unauthorized")
	ErrForbidden          = NewBizError(403, 40300, "forbidden")
	ErrNotFound           = NewBizError(404, 40400, "resource not found")
	ErrConflict           = NewBizError(409, 40900, "conflict")
	ErrValidation         = NewBizError(422, 42200, "validation failed")
	ErrTooManyRequests    = NewBizError(429, 42900, "too many requests")
	ErrInternal           = NewBizError(500, 50000, "internal server error")
	ErrServiceUnavailable = NewBizError(503, 50300, "service unavailable")
)
