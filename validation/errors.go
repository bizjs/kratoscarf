package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldError represents a single field validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

// ValidationErrors is returned when one or more fields fail validation.
type ValidationErrors []FieldError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	msgs := make([]string, 0, len(e))
	for _, fe := range e {
		msgs = append(msgs, fmt.Sprintf("%s: %s", fe.Field, fe.Message))
	}
	return strings.Join(msgs, "; ")
}

// HTTPStatus returns 422 so error encoders can set the correct HTTP status.
// Implements the httpStatuser interface checked by response.NewHTTPErrorEncoder.
func (e ValidationErrors) HTTPStatus() int { return 422 }

// BizCode returns the business error code for validation failures.
func (e ValidationErrors) BizCode() int { return 42200 }

// ErrorData returns structured field errors for inclusion in error responses.
func (e ValidationErrors) ErrorData() any { return []FieldError(e) }

// BindError wraps a binding failure with a 400 status code.
type BindError struct{ Err error }

func (e *BindError) Error() string   { return e.Err.Error() }
func (e *BindError) Unwrap() error   { return e.Err }
func (e *BindError) HTTPStatus() int { return 400 }
func (e *BindError) BizCode() int    { return 40000 }

// Translate converts go-playground validator.ValidationErrors
// into kratoscarf ValidationErrors with readable messages.
func Translate(err error) ValidationErrors {
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return ValidationErrors{{
			Field:   "",
			Rule:    "",
			Message: err.Error(),
		}}
	}
	result := make(ValidationErrors, 0, len(ve))
	for _, fe := range ve {
		result = append(result, FieldError{
			Field:   fe.Field(),
			Rule:    fe.Tag(),
			Param:   fe.Param(),
			Message: buildMessage(fe),
		})
	}
	return result
}

func buildMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		if isNumericKind(fe.Kind()) {
			return fmt.Sprintf("must be at least %s", fe.Param())
		}
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		if isNumericKind(fe.Kind()) {
			return fmt.Sprintf("must be at most %s", fe.Param())
		}
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", fe.Param())
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	default:
		return fmt.Sprintf("failed on '%s' validation", fe.Tag())
	}
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
