package jwt

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// middlewareConfig holds middleware configuration.
type middlewareConfig struct {
	extractor    TokenExtractor
	skipPaths    map[string]bool
	errorHandler func(ctx context.Context, err error) error
}

// MiddlewareOption is a functional option for configuring the JWT middleware.
type MiddlewareOption func(*middlewareConfig)

// WithExtractor overrides the default BearerExtractor.
func WithExtractor(e TokenExtractor) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.extractor = e
	}
}

// WithSkipPaths skips authentication for specific paths (e.g. /login, /health).
func WithSkipPaths(paths ...string) MiddlewareOption {
	return func(c *middlewareConfig) {
		for _, p := range paths {
			c.skipPaths[p] = true
		}
	}
}

// WithErrorHandler overrides the default 401 error behavior.
func WithErrorHandler(fn func(ctx context.Context, err error) error) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.errorHandler = fn
	}
}

// Middleware returns a Kratos middleware that validates tokens and injects Claims into context.
func Middleware(authenticator *Authenticator, opts ...MiddlewareOption) middleware.Middleware {
	cfg := &middlewareConfig{
		extractor: BearerExtractor(),
		skipPaths: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Extract path from transport info for skip-path check.
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*kratoshttp.Transport); ok {
					path := ht.Request().URL.Path
					if cfg.skipPaths[path] {
						return handler(ctx, req)
					}

					token, err := cfg.extractor.Extract(ht.Request())
					if err != nil {
						if cfg.errorHandler != nil {
							return nil, cfg.errorHandler(ctx, err)
						}
						return nil, err
					}

					claims, err := authenticator.ValidateToken(ctx, token)
					if err != nil {
						if cfg.errorHandler != nil {
							return nil, cfg.errorHandler(ctx, err)
						}
						return nil, err
					}

					ctx = ContextWithClaims(ctx, claims)
				}
			}
			return handler(ctx, req)
		}
	}
}
