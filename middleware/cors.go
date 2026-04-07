package middleware

import (
	"net/http"

	"github.com/rs/cors"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// CORSOption configures the CORS middleware.
type CORSOption func(*cors.Options)

// WithAllowOrigins sets the allowed origins. Default: ["*"].
func WithAllowOrigins(origins ...string) CORSOption {
	return func(o *cors.Options) { o.AllowedOrigins = origins }
}

// WithAllowMethods sets the allowed HTTP methods.
func WithAllowMethods(methods ...string) CORSOption {
	return func(o *cors.Options) { o.AllowedMethods = methods }
}

// WithAllowHeaders sets the allowed request headers.
func WithAllowHeaders(headers ...string) CORSOption {
	return func(o *cors.Options) { o.AllowedHeaders = headers }
}

// WithExposeHeaders sets the headers the browser can read.
func WithExposeHeaders(headers ...string) CORSOption {
	return func(o *cors.Options) { o.ExposedHeaders = headers }
}

// WithAllowCredentials enables credentials (cookies, auth headers).
// When true, AllowOrigins must not contain "*".
func WithAllowCredentials() CORSOption {
	return func(o *cors.Options) { o.AllowCredentials = true }
}

// WithMaxAge sets how long (in seconds) preflight responses can be cached.
func WithMaxAge(seconds int) CORSOption {
	return func(o *cors.Options) { o.MaxAge = seconds }
}

// CORS returns an HTTP filter for Cross-Origin Resource Sharing.
// With no options, it allows all origins with sensible defaults.
//
//	kratoshttp.Filter(middleware.CORS())
//	kratoshttp.Filter(middleware.CORS(middleware.WithAllowOrigins("https://example.com"), middleware.WithAllowCredentials()))
func CORS(opts ...CORSOption) kratoshttp.FilterFunc {
	o := cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:         86400,
	}
	for _, opt := range opts {
		opt(&o)
	}
	c := cors.New(o)
	return func(next http.Handler) http.Handler {
		return c.Handler(next)
	}
}
