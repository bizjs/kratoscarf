package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
)

type requestIDKey struct{}

const headerXRequestID = "X-Request-Id"

// RequestIDFromContext returns the request ID from context, or empty string if not set.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// RequestID returns a Kratos middleware that generates and propagates X-Request-Id.
// If the incoming request already has an X-Request-Id header, it is reused.
func RequestID() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			ht, ok := tr.(*kratoshttp.Transport)
			if !ok {
				return handler(ctx, req)
			}

			id := ht.Request().Header.Get(headerXRequestID)
			if id == "" {
				id = uuid.New().String()
			}

			ctx = context.WithValue(ctx, requestIDKey{}, id)
			ht.ReplyHeader().Set(headerXRequestID, id)

			return handler(ctx, req)
		}
	}
}
