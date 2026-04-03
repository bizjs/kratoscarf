package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator with kratoscarf conventions.
type Validator struct {
	validate *validator.Validate
}

// Option configures a Validator.
type Option func(*Validator)

// WithRule registers a simple custom validation rule.
// The function receives the field value and returns true if valid.
//
//	v := validation.New(
//	    validation.WithRule("even", func(v any) bool {
//	        n, ok := v.(int)
//	        return ok && n%2 == 0
//	    }),
//	)
func WithRule(tag string, fn func(any) bool) Option {
	return func(v *Validator) {
		_ = v.validate.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
			return fn(fl.Field().Interface())
		})
	}
}

// WithRuleFunc registers an advanced custom validation rule using the
// go-playground/validator FieldLevel API (for cross-field validation, etc.).
func WithRuleFunc(tag string, fn validator.Func) Option {
	return func(v *Validator) {
		_ = v.validate.RegisterValidation(tag, fn)
	}
}

// WithTagName overrides the struct tag name (default: "validate").
func WithTagName(name string) Option {
	return func(v *Validator) {
		v.validate.SetTagName(name)
	}
}

// New creates a Validator with sensible defaults and the given options.
func New(opts ...Option) *Validator {
	validate := validator.New()
	// Use JSON tag names for field names in error messages.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("json")
		if name == "" || name == "-" {
			return fld.Name
		}
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}
		return name
	})
	v := &Validator{validate: validate}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Struct validates a struct by its `validate` tags.
// Returns ValidationErrors on failure, nil on success.
func (v *Validator) Struct(s any) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}
	return Translate(err)
}

// Var validates a single variable against a tag string.
func (v *Validator) Var(field any, tag string) error {
	err := v.validate.Var(field, tag)
	if err == nil {
		return nil
	}
	return Translate(err)
}

// Validate validates a struct and returns ValidationErrors (422) on failure.
// Same as Struct() — the returned ValidationErrors implements HTTPStatus()
// and ErrorData() for automatic error encoding.
func (v *Validator) Validate(s any) error {
	return v.Struct(s)
}

// RegisterAlias registers a validation tag alias.
//
//	v.RegisterAlias("isColor", "hexcolor|rgb|rgba")
func (v *Validator) RegisterAlias(alias, tags string) {
	v.validate.RegisterAlias(alias, tags)
}
