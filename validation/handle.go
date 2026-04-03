package validation

// Binder is any object that has a Bind method (e.g., *router.Context).
type Binder interface {
	Bind(any) error
}

// Handle wraps a typed handler with automatic bind + validate.
// If bind or validation fails, the handler is never called.
//
// Usage:
//
//	r.POST("/todos", validation.Handle(v, s.Create))
//
//	func (s *Svc) Create(ctx *router.Context, req *CreateRequest) error {
//	    // req is already bound and validated
//	}
func Handle[Ctx Binder, T any](v *Validator, fn func(Ctx, *T) error) func(Ctx) error {
	return func(ctx Ctx) error {
		var req T
		if err := BindAndValidate(ctx.Bind, &req, v); err != nil {
			return err
		}
		return fn(ctx, &req)
	}
}

// BindAndValidate binds data, then validates.
// Returns BindError (400) on bind failure, ValidationErrors (422) on validation failure.
func BindAndValidate(bind func(any) error, dst any, v *Validator) error {
	if err := bind(dst); err != nil {
		return &BindError{Err: err}
	}
	return v.Validate(dst)
}
