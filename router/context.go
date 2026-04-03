package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// Context wraps Kratos http.Context with convenience methods.
type Context struct {
	kratosCtx kratoshttp.Context
	request   *http.Request
	response  http.ResponseWriter
	validator StructValidator  // set by WithValidator
	wrapper   ResponseWrapper  // set by WithResponseWrapper
}

// --- Request helpers ---

// Param returns a URL path parameter by name.
func (c *Context) Param(key string) string {
	if c.kratosCtx != nil {
		return c.kratosCtx.Vars().Get(key)
	}
	return ""
}

// Query returns a query string parameter by name.
func (c *Context) Query(key string) string {
	if c.request != nil {
		return c.request.URL.Query().Get(key)
	}
	return ""
}

// QueryDefault returns a query string parameter with a fallback default.
func (c *Context) QueryDefault(key, defaultVal string) string {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// Header returns a request header value.
func (c *Context) Header(key string) string {
	if c.request != nil {
		return c.request.Header.Get(key)
	}
	return ""
}

// Bind decodes the request body into dst, then auto-validates if a
// validator was set via WithValidator (Gin-style).
func (c *Context) Bind(dst any) error {
	if c.kratosCtx == nil {
		return errors.New("router: no kratos context available for binding")
	}
	if err := c.kratosCtx.Bind(dst); err != nil {
		return err
	}
	if c.validator != nil {
		return c.validator.Validate(dst)
	}
	return nil
}

// BindQuery decodes query parameters into dst, then auto-validates if a
// validator was set via WithValidator.
func (c *Context) BindQuery(dst any) error {
	if c.kratosCtx == nil {
		return errors.New("router: no kratos context available for binding")
	}
	if err := c.kratosCtx.BindQuery(dst); err != nil {
		return err
	}
	if c.validator != nil {
		return c.validator.Validate(dst)
	}
	return nil
}

// --- Response helpers ---

// JSON sends a JSON response with the given status code.
func (c *Context) JSON(code int, data any) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	c.response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.response.WriteHeader(code)
	return json.NewEncoder(c.response).Encode(data)
}

// Success sends a 200 JSON response. If a ResponseWrapper is set on the
// router, data is automatically wrapped (e.g., {code: 0, message: "ok", data: ...}).
func (c *Context) Success(data any) error {
	if c.wrapper != nil {
		data = c.wrapper(data)
	}
	return c.JSON(http.StatusOK, data)
}

// Redirect sends an HTTP redirect.
func (c *Context) Redirect(code int, url string) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	http.Redirect(c.response, c.request, url, code)
	return nil
}

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) {
	if c.response != nil {
		c.response.Header().Set(key, value)
	}
}

// Cookie returns the named cookie from the request.
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

// SetCookie sets a cookie on the response.
func (c *Context) SetCookie(cookie *http.Cookie) {
	if c.response != nil {
		http.SetCookie(c.response, cookie)
	}
}

// NoContent sends a 204 response.
func (c *Context) NoContent() error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	c.response.WriteHeader(http.StatusNoContent)
	return nil
}

// Stream sends raw bytes with a given content type.
func (c *Context) Stream(contentType string, reader io.Reader) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	c.response.Header().Set("Content-Type", contentType)
	c.response.WriteHeader(http.StatusOK)
	_, err := io.Copy(c.response, reader)
	return err
}

// --- Context passthrough ---

// Request returns the raw *http.Request.
func (c *Context) Request() *http.Request {
	return c.request
}

// Response returns the raw http.ResponseWriter.
func (c *Context) Response() http.ResponseWriter {
	return c.response
}

// Context returns the Go context.Context (carries Kratos transport metadata).
func (c *Context) Context() context.Context {
	if c.request != nil {
		return c.request.Context()
	}
	return context.Background()
}

// SetValue sets a value on the request context (for passing data between middleware and handler).
func (c *Context) SetValue(key, val any) {
	if c.request != nil {
		ctx := context.WithValue(c.request.Context(), key, val)
		c.request = c.request.WithContext(ctx)
	}
}

// GetValue retrieves a value from the request context.
func (c *Context) GetValue(key any) any {
	if c.request != nil {
		return c.request.Context().Value(key)
	}
	return nil
}
