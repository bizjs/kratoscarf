package response

import (
	"encoding/json"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPResponseEncoder returns a Kratos HTTP ResponseEncoder that wraps
// all responses in the unified Response format.
func NewHTTPResponseEncoder() kratoshttp.EncodeResponseFunc {
	return func(w http.ResponseWriter, r *http.Request, v any) error {
		resp := Success(v)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return json.NewEncoder(w).Encode(resp)
	}
}

// NewHTTPErrorEncoder returns a Kratos HTTP ErrorEncoder that converts
// errors into the unified Response format.
//
// It recognizes errors via duck typing:
//   - BizError: uses HTTPCode, Code, Message, Data
//   - Any error with HTTPStatus() int: uses that as HTTP status
//   - Any error with BizCode() int: uses that as business code
//   - Any error with ErrorData() any: includes that in response data
//   - Other errors: 500 with generic message
func NewHTTPErrorEncoder() kratoshttp.EncodeErrorFunc {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		resp := errorToResponse(err)
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

// errorToResponse converts an error to a Response, checking duck-typed interfaces.
func errorToResponse(err error) *Response {
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
