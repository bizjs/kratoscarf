package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// Context wraps Kratos http.Context with convenience methods.
type Context struct {
	kratosCtx kratoshttp.Context
	request   *http.Request
	response  http.ResponseWriter
	validator StructValidator // set by WithValidator
	wrapper   ResponseWrapper // set by WithResponseWrapper
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

// QueryArray returns all values for a query parameter key.
// Useful for repeated keys like ?ids=1&ids=2&ids=3.
func (c *Context) QueryArray(key string) []string {
	if c.request != nil {
		return c.request.URL.Query()[key]
	}
	return nil
}

// ClientIP returns the client's real IP address, checking
// X-Forwarded-For and X-Real-Ip headers before falling back to RemoteAddr.
func (c *Context) ClientIP() string {
	if c.request == nil {
		return ""
	}
	if xff := c.request.Header.Get("X-Forwarded-For"); xff != "" {
		ip, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(ip)
	}
	if xri := c.request.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr is "ip:port" or "[ipv6]:port"
	addr := c.request.RemoteAddr
	if host, _, ok := strings.Cut(addr, "]:"); ok {
		return strings.TrimPrefix(host, "[")
	}
	if host, _, ok := strings.Cut(addr, ":"); ok {
		return host
	}
	return addr
}

// FormFile returns the first file for the given form key.
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	if c.request == nil {
		return nil, errors.New("router: no request available")
	}
	_, fh, err := c.request.FormFile(name)
	return fh, err
}

// MultipartForm parses the request as multipart form data with the given max memory.
// Remaining file data is stored in temporary files. The caller does not need to call
// r.Body.Close(), but should eventually remove temp files via r.MultipartForm.RemoveAll().
func (c *Context) MultipartForm(maxMemory int64) (*multipart.Form, error) {
	if c.request == nil {
		return nil, errors.New("router: no request available")
	}
	if err := c.request.ParseMultipartForm(maxMemory); err != nil {
		return nil, err
	}
	return c.request.MultipartForm, nil
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

// String sends a plain text response with optional formatting.
func (c *Context) String(code int, format string, values ...any) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	c.response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.response.WriteHeader(code)
	if len(values) > 0 {
		_, err := fmt.Fprintf(c.response, format, values...)
		return err
	}
	_, err := io.WriteString(c.response, format)
	return err
}

// Data sends raw bytes with the given content type and status code.
func (c *Context) Data(code int, contentType string, data []byte) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	c.response.Header().Set("Content-Type", contentType)
	c.response.WriteHeader(code)
	_, err := c.response.Write(data)
	return err
}

// File sends a file as the response. The Content-Type is inferred from the filename.
func (c *Context) File(filePath string) error {
	if c.request == nil || c.response == nil {
		return errors.New("router: no request/response available")
	}
	http.ServeFile(c.response, c.request, filePath)
	return nil
}

// Attachment sends a file as a download with the given filename.
func (c *Context) Attachment(filePath, filename string) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	if filename == "" {
		filename = filepath.Base(filePath)
	}
	c.response.Header().Set("Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	return c.File(filePath)
}

// Inline sends a file for inline display with the given filename.
func (c *Context) Inline(filePath, filename string) error {
	if c.response == nil {
		return errors.New("router: no response writer available")
	}
	if filename == "" {
		filename = filepath.Base(filePath)
	}
	c.response.Header().Set("Content-Disposition",
		mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	return c.File(filePath)
}

// ContentType returns the request's Content-Type (without parameters).
func (c *Context) ContentType() string {
	if c.request == nil {
		return ""
	}
	ct := c.request.Header.Get("Content-Type")
	mt, _, _ := mime.ParseMediaType(ct)
	return mt
}

// QueryString returns the raw query string without '?'.
func (c *Context) QueryString() string {
	if c.request != nil {
		return c.request.URL.RawQuery
	}
	return ""
}

// QueryValues returns all query parameters as url.Values.
func (c *Context) QueryValues() url.Values {
	if c.request != nil {
		return c.request.URL.Query()
	}
	return nil
}

// FormValue returns a single form value by key (from POST body or query string).
func (c *Context) FormValue(key string) string {
	if c.request != nil {
		return c.request.FormValue(key)
	}
	return ""
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
