package session

import (
	"context"
	"net/http"

	kratosmiddleware "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// MiddlewareOption configures the session middleware.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	skipPaths map[string]bool
}

// WithSkipPaths skips session loading for specific paths.
func WithSkipPaths(paths ...string) MiddlewareOption {
	return func(c *middlewareConfig) {
		for _, p := range paths {
			c.skipPaths[p] = true
		}
	}
}

func responseWriter(ctx context.Context) (http.ResponseWriter, bool) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil, false
	}
	ht, ok := tr.(*kratoshttp.Transport)
	if !ok {
		return nil, false
	}
	return ht.Response(), true
}

// Middleware returns a Kratos middleware that loads the session from
// the request cookie, injects it into context, and saves it after
// the handler if modified.
func Middleware(manager *Manager, opts ...MiddlewareOption) kratosmiddleware.Middleware {
	cfg := &middlewareConfig{
		skipPaths: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(handler kratosmiddleware.Handler) kratosmiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			httpReq, ok := kratoshttp.RequestFromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if cfg.skipPaths[httpReq.URL.Path] {
				return handler(ctx, req)
			}

			sess, err := manager.GetSession(ctx, httpReq)
			if err != nil {
				return nil, err
			}

			ctx = ContextWithSession(ctx, sess)
			resp, err := handler(ctx, req)

			if sess.Modified || sess.IsNew {
				if w, ok := responseWriter(ctx); ok {
					_ = manager.SaveSession(ctx, w, sess)
				}
			}

			return resp, err
		}
	}
}
