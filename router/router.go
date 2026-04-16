package router

import (
	"context"

	kratosmiddleware "github.com/go-kratos/kratos/v2/middleware"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// HandlerFunc is the kratoscarf handler signature.
// Returns error for unified error handling.
type HandlerFunc func(ctx *Context) error

// StructValidator validates a struct. Implemented by validation.Validator.
type StructValidator interface {
	Validate(any) error
}

// ResponseWrapper wraps response data before writing.
// Typically a function that adds {code, message, data} envelope.
type ResponseWrapper = func(data any) any

// Router wraps Kratos HTTP Server routing with a friendlier API.
type Router struct {
	server     *kratoshttp.Server
	route      *kratoshttp.Router
	prefix     string
	middleware []kratosmiddleware.Middleware
	validator  StructValidator
	wrapper    ResponseWrapper
}

// NewRouter creates a new Router bound to a Kratos HTTP Server.
func NewRouter(srv *kratoshttp.Server, opts ...Option) *Router {
	r := &Router{
		server: srv,
		route:  srv.Route("/"),
		prefix: "",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Option configures a Router.
type Option func(*Router)

// WithValidator sets a struct validator. When set, Bind() automatically
// validates after binding (Gin-style). Pass validation.New() here.
func WithValidator(v StructValidator) Option {
	return func(r *Router) { r.validator = v }
}

// WithResponseWrapper sets a response wrapper for ctx.Success().
// Pass response.Success to auto-wrap all responses in {code, message, data}.
func WithResponseWrapper(w ResponseWrapper) Option {
	return func(r *Router) { r.wrapper = w }
}

// Group creates a sub-router with a path prefix and optional middleware.
func (r *Router) Group(prefix string, mws ...kratosmiddleware.Middleware) *Router {
	combined := make([]kratosmiddleware.Middleware, 0, len(r.middleware)+len(mws))
	combined = append(combined, r.middleware...)
	combined = append(combined, mws...)
	return &Router{
		server:     r.server,
		route:      r.route,
		prefix:     r.prefix + prefix,
		middleware: combined,
		validator:  r.validator,
		wrapper:    r.wrapper,
	}
}

// GET registers a handler for GET requests.
func (r *Router) GET(path string, h HandlerFunc) {
	r.Handle("GET", path, h)
}

// POST registers a handler for POST requests.
func (r *Router) POST(path string, h HandlerFunc) {
	r.Handle("POST", path, h)
}

// PUT registers a handler for PUT requests.
func (r *Router) PUT(path string, h HandlerFunc) {
	r.Handle("PUT", path, h)
}

// DELETE registers a handler for DELETE requests.
func (r *Router) DELETE(path string, h HandlerFunc) {
	r.Handle("DELETE", path, h)
}

// PATCH registers a handler for PATCH requests.
func (r *Router) PATCH(path string, h HandlerFunc) {
	r.Handle("PATCH", path, h)
}

// HEAD registers a handler for HEAD requests.
func (r *Router) HEAD(path string, h HandlerFunc) {
	r.Handle("HEAD", path, h)
}

// OPTIONS registers a handler for OPTIONS requests.
func (r *Router) OPTIONS(path string, h HandlerFunc) {
	r.Handle("OPTIONS", path, h)
}

// Handle registers a handler for the given method and path.
func (r *Router) Handle(method, path string, h HandlerFunc) {
	fullPath := r.prefix + path

	v := r.validator // capture for closure
	w := r.wrapper

	r.route.Handle(method, fullPath, func(ctx kratoshttp.Context) error {
		var inner kratosmiddleware.Handler = func(mwCtx context.Context, _ any) (any, error) {
			rc := &Context{
				kratosCtx: ctx,
				request:   ctx.Request().WithContext(mwCtx),
				response:  ctx.Response(),
				validator: v,
				wrapper:   w,
			}
			return nil, h(rc)
		}

		// Route-level middleware (JWT, RBAC, etc.)
		if len(r.middleware) > 0 {
			inner = kratosmiddleware.Chain(r.middleware...)(inner)
		}

		// Server-level middleware (recovery, logging, tracing, metrics)
		// via ctx.Middleware() — matches how protobuf-generated handlers work.
		_, err := ctx.Middleware(inner)(ctx.Request().Context(), nil)
		return err
	})
}

// Use appends middleware to this router scope.
func (r *Router) Use(mws ...kratosmiddleware.Middleware) {
	r.middleware = append(r.middleware, mws...)
}
