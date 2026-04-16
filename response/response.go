package response

// Response is the standard API response wrapper.
type Response struct {
	Code    int    `json:"code"`           // business code, 0 = success
	Message string `json:"message"`        // human-readable message
	Data    any    `json:"data,omitempty"` // response payload
}

// Success creates a success response.
func Success(data any) *Response {
	return &Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	}
}

// SuccessWithMessage creates a success response with a custom message.
func SuccessWithMessage(data any, msg string) *Response {
	return &Response{
		Code:    0,
		Message: msg,
		Data:    data,
	}
}

// Wrap is an alias for Success that returns any.
// Designed for use with router.WithResponseWrapper(response.Wrap).
func Wrap(data any) any {
	return Success(data)
}

// Error creates an error response from an error.
// Delegates to ErrorToResponse for duck-typed interface support.
func Error(err error) *Response {
	return ErrorToResponse(err)
}
