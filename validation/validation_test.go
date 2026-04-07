package validation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
)

// ---------------------------------------------------------------------------
// Test structs
// ---------------------------------------------------------------------------

type CreateUserReq struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=0,lte=150"`
}

type CustomTagReq struct {
	Name string `json:"name" binding:"required"`
}

type EvenFieldReq struct {
	Count int `json:"count" validate:"even"`
}

type PasswordReq struct {
	Password        string `json:"password"         validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password"  validate:"required,eqfield=Password"`
}

type AliasReq struct {
	Color string `json:"color" validate:"required,isColor"`
}

type NoJSONTagReq struct {
	Username string `validate:"required"`
}

// ---------------------------------------------------------------------------
// 1. New() creates validator; default tag name is "validate"
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}
	if v.validate == nil {
		t.Fatal("internal validator is nil")
	}
}

func TestNew_DefaultTagIsValidate(t *testing.T) {
	v := New()
	// A struct using the "validate" tag should be validated.
	req := CreateUserReq{Name: "", Email: "bad"}
	err := v.Struct(req)
	if err == nil {
		t.Fatal("expected validation error for empty required fields, got nil")
	}
}

// ---------------------------------------------------------------------------
// 2. Validate / Struct — valid passes, invalid returns ValidationErrors
// ---------------------------------------------------------------------------

func TestStruct_ValidPasses(t *testing.T) {
	v := New()
	req := CreateUserReq{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := v.Struct(req); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestStruct_InvalidReturnsValidationErrors(t *testing.T) {
	v := New()
	req := CreateUserReq{Name: "", Email: "not-an-email", Age: -1}
	err := v.Struct(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(ve) == 0 {
		t.Fatal("expected at least one field error")
	}
	// Should contain errors for name (required), email (email), age (gte)
	fields := make(map[string]bool)
	for _, fe := range ve {
		fields[fe.Field] = true
	}
	for _, expected := range []string{"name", "email", "age"} {
		if !fields[expected] {
			t.Errorf("expected error for field %q, got fields: %v", expected, fields)
		}
	}
}

func TestValidate_IsSameAsStruct(t *testing.T) {
	v := New()
	req := CreateUserReq{Name: "", Email: "x"}
	err1 := v.Struct(req)
	err2 := v.Validate(req)
	if err1 == nil || err2 == nil {
		t.Fatal("both should return errors")
	}
	var ve1, ve2 ValidationErrors
	errors.As(err1, &ve1)
	errors.As(err2, &ve2)
	if len(ve1) != len(ve2) {
		t.Fatalf("Struct and Validate should produce same number of errors: %d vs %d", len(ve1), len(ve2))
	}
}

// ---------------------------------------------------------------------------
// 3. Var — validate single variable
// ---------------------------------------------------------------------------

func TestVar_ValidEmail(t *testing.T) {
	v := New()
	if err := v.Var("alice@example.com", "required,email"); err != nil {
		t.Fatalf("expected valid email to pass, got %v", err)
	}
}

func TestVar_InvalidEmail(t *testing.T) {
	v := New()
	err := v.Var("not-email", "email")
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(ve) != 1 {
		t.Fatalf("expected 1 error, got %d", len(ve))
	}
	if ve[0].Rule != "email" {
		t.Errorf("expected rule 'email', got %q", ve[0].Rule)
	}
}

func TestVar_RequiredEmpty(t *testing.T) {
	v := New()
	err := v.Var("", "required")
	if err == nil {
		t.Fatal("expected error for empty required var")
	}
}

func TestVar_MinMax(t *testing.T) {
	v := New()
	// String length check
	if err := v.Var("ab", "min=2,max=5"); err != nil {
		t.Fatalf("expected 'ab' to pass min=2,max=5, got %v", err)
	}
	if err := v.Var("a", "min=2"); err == nil {
		t.Fatal("expected error for 'a' with min=2")
	}
}

// ---------------------------------------------------------------------------
// 4. WithRule — register custom validation rule via struct tag
// ---------------------------------------------------------------------------

func TestWithRule(t *testing.T) {
	v := New(
		WithRule("even", func(val any) bool {
			n, ok := val.(int)
			return ok && n%2 == 0
		}),
	)

	// Valid: even number
	req := EvenFieldReq{Count: 4}
	if err := v.Struct(req); err != nil {
		t.Fatalf("expected even=4 to pass, got %v", err)
	}

	// Invalid: odd number
	req2 := EvenFieldReq{Count: 3}
	err := v.Struct(req2)
	if err == nil {
		t.Fatal("expected error for odd number")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(ve) != 1 {
		t.Fatalf("expected 1 error, got %d", len(ve))
	}
	if ve[0].Rule != "even" {
		t.Errorf("expected rule 'even', got %q", ve[0].Rule)
	}
	if ve[0].Field != "count" {
		t.Errorf("expected field 'count' (json tag), got %q", ve[0].Field)
	}
}

// ---------------------------------------------------------------------------
// 5. WithRuleFunc — register custom rule with FieldLevel function
// ---------------------------------------------------------------------------

func TestWithRuleFunc(t *testing.T) {
	// Register a rule using the full validator.FieldLevel API.
	vPass := New(
		WithRuleFunc("always_pass", func(fl validator.FieldLevel) bool {
			return true
		}),
	)

	type AlwaysPassReq struct {
		Val string `json:"val" validate:"always_pass"`
	}
	if err := vPass.Struct(AlwaysPassReq{Val: "anything"}); err != nil {
		t.Fatalf("expected always_pass to succeed, got %v", err)
	}

	vFail := New(
		WithRuleFunc("always_fail", func(fl validator.FieldLevel) bool {
			return false
		}),
	)
	type AlwaysFailReq struct {
		Val string `json:"val" validate:"always_fail"`
	}
	err := vFail.Struct(AlwaysFailReq{Val: "anything"})
	if err == nil {
		t.Fatal("expected always_fail to return error")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if ve[0].Rule != "always_fail" {
		t.Errorf("expected rule 'always_fail', got %q", ve[0].Rule)
	}
}

func TestWithRuleFunc_CrossField(t *testing.T) {
	// Use FieldLevel to implement cross-field validation.
	v := New(
		WithRuleFunc("eqfield_custom", func(fl validator.FieldLevel) bool {
			// Compare current field to the param-named sibling field.
			otherName := fl.Param()
			other := fl.Parent().FieldByName(otherName)
			return fl.Field().String() == other.String()
		}),
	)
	type Req struct {
		A string `json:"a" validate:"required"`
		B string `json:"b" validate:"eqfield_custom=A"`
	}
	if err := v.Struct(Req{A: "same", B: "same"}); err != nil {
		t.Fatalf("expected match to pass, got %v", err)
	}
	err := v.Struct(Req{A: "one", B: "two"})
	if err == nil {
		t.Fatal("expected error when fields differ")
	}
}

// ---------------------------------------------------------------------------
// 6. WithTagName — changes the struct tag name
// ---------------------------------------------------------------------------

func TestWithTagName(t *testing.T) {
	v := New(WithTagName("binding"))
	req := CustomTagReq{Name: ""}
	err := v.Struct(req)
	if err == nil {
		t.Fatal("expected validation error with 'binding' tag")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(ve) != 1 {
		t.Fatalf("expected 1 error, got %d", len(ve))
	}
	if ve[0].Rule != "required" {
		t.Errorf("expected rule 'required', got %q", ve[0].Rule)
	}
}

func TestWithTagName_OriginalTagIgnored(t *testing.T) {
	v := New(WithTagName("binding"))
	// CreateUserReq uses "validate" tag which should now be ignored.
	req := CreateUserReq{Name: "", Email: "bad"}
	if err := v.Struct(req); err != nil {
		t.Fatal("expected no error because 'validate' tag should be ignored when tag is 'binding'")
	}
}

// ---------------------------------------------------------------------------
// 7. ValidationErrors — Error(), HTTPStatus(), BizCode(), ErrorData()
// ---------------------------------------------------------------------------

func TestValidationErrors_Error_Empty(t *testing.T) {
	var ve ValidationErrors
	if got := ve.Error(); got != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", got)
	}
}

func TestValidationErrors_Error_Single(t *testing.T) {
	ve := ValidationErrors{
		{Field: "email", Rule: "email", Message: "must be a valid email address"},
	}
	expected := "email: must be a valid email address"
	if got := ve.Error(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestValidationErrors_Error_Multiple(t *testing.T) {
	ve := ValidationErrors{
		{Field: "name", Rule: "required", Message: "is required"},
		{Field: "age", Rule: "gte", Message: "must be greater than or equal to 0"},
	}
	got := ve.Error()
	if got != "name: is required; age: must be greater than or equal to 0" {
		t.Errorf("unexpected Error() output: %q", got)
	}
}

func TestValidationErrors_HTTPStatus(t *testing.T) {
	ve := ValidationErrors{{Field: "x", Message: "bad"}}
	if got := ve.HTTPStatus(); got != 422 {
		t.Errorf("expected HTTPStatus()=422, got %d", got)
	}
}

func TestValidationErrors_BizCode(t *testing.T) {
	ve := ValidationErrors{{Field: "x", Message: "bad"}}
	if got := ve.BizCode(); got != 42200 {
		t.Errorf("expected BizCode()=42200, got %d", got)
	}
}

func TestValidationErrors_ErrorData(t *testing.T) {
	ve := ValidationErrors{
		{Field: "name", Rule: "required", Message: "is required"},
		{Field: "email", Rule: "email", Message: "must be a valid email address"},
	}
	data := ve.ErrorData()
	slice, ok := data.([]FieldError)
	if !ok {
		t.Fatalf("expected []FieldError, got %T", data)
	}
	if len(slice) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(slice))
	}
	if slice[0].Field != "name" || slice[1].Field != "email" {
		t.Errorf("unexpected field names: %v", slice)
	}
}

// ---------------------------------------------------------------------------
// 8. FieldError — Field, Tag/Rule, Message populated correctly
// ---------------------------------------------------------------------------

func TestFieldError_PopulatedFromValidation(t *testing.T) {
	v := New()
	type Req struct {
		Email string `json:"email" validate:"required,email"`
	}
	err := v.Struct(Req{Email: ""})
	if err == nil {
		t.Fatal("expected error")
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve) == 0 {
		t.Fatal("expected field errors")
	}
	fe := ve[0]
	if fe.Field != "email" {
		t.Errorf("expected Field='email', got %q", fe.Field)
	}
	if fe.Rule != "required" {
		t.Errorf("expected Rule='required', got %q", fe.Rule)
	}
	if fe.Message != "is required" {
		t.Errorf("expected Message='is required', got %q", fe.Message)
	}
}

func TestFieldError_JSONTagUsedAsFieldName(t *testing.T) {
	v := New()
	type Req struct {
		FirstName string `json:"first_name" validate:"required"`
	}
	err := v.Struct(Req{})
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve) == 0 {
		t.Fatal("expected errors")
	}
	if ve[0].Field != "first_name" {
		t.Errorf("expected field name from json tag 'first_name', got %q", ve[0].Field)
	}
}

func TestFieldError_NoJSONTag_UsesStructFieldName(t *testing.T) {
	v := New()
	err := v.Struct(NoJSONTagReq{Username: ""})
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve) == 0 {
		t.Fatal("expected errors")
	}
	if ve[0].Field != "Username" {
		t.Errorf("expected field name 'Username', got %q", ve[0].Field)
	}
}

func TestFieldError_ParamPopulated(t *testing.T) {
	v := New()
	type Req struct {
		Name string `json:"name" validate:"min=3"`
	}
	err := v.Struct(Req{Name: "ab"})
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve) == 0 {
		t.Fatal("expected errors")
	}
	if ve[0].Param != "3" {
		t.Errorf("expected Param='3', got %q", ve[0].Param)
	}
	if ve[0].Rule != "min" {
		t.Errorf("expected Rule='min', got %q", ve[0].Rule)
	}
}

// ---------------------------------------------------------------------------
// 9. BindError — implements HTTPStatus() returning 400, BizCode() returning 40000
// ---------------------------------------------------------------------------

func TestBindError_HTTPStatus(t *testing.T) {
	be := &BindError{Err: errors.New("bad json")}
	if got := be.HTTPStatus(); got != 400 {
		t.Errorf("expected HTTPStatus()=400, got %d", got)
	}
}

func TestBindError_BizCode(t *testing.T) {
	be := &BindError{Err: errors.New("bad json")}
	if got := be.BizCode(); got != 40000 {
		t.Errorf("expected BizCode()=40000, got %d", got)
	}
}

func TestBindError_Error(t *testing.T) {
	inner := errors.New("invalid JSON at offset 42")
	be := &BindError{Err: inner}
	if got := be.Error(); got != "invalid JSON at offset 42" {
		t.Errorf("expected %q, got %q", inner.Error(), got)
	}
}

func TestBindError_Unwrap(t *testing.T) {
	inner := errors.New("underlying")
	be := &BindError{Err: inner}
	if !errors.Is(be, inner) {
		t.Error("expected Unwrap to expose inner error")
	}
}

// ---------------------------------------------------------------------------
// 10. Translate — converts go-playground errors to ValidationErrors
// ---------------------------------------------------------------------------

func TestTranslate_Nil(t *testing.T) {
	if got := Translate(nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestTranslate_NonValidatorError(t *testing.T) {
	err := errors.New("some random error")
	ve := Translate(err)
	if len(ve) != 1 {
		t.Fatalf("expected 1 error entry, got %d", len(ve))
	}
	if ve[0].Message != "some random error" {
		t.Errorf("expected message 'some random error', got %q", ve[0].Message)
	}
	if ve[0].Field != "" {
		t.Errorf("expected empty field, got %q", ve[0].Field)
	}
}

// ---------------------------------------------------------------------------
// 11. buildMessage — human-readable messages for various rules
// ---------------------------------------------------------------------------

func TestBuildMessage_ViaStruct(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		val     any
		wantMsg string
	}{
		{
			name: "required",
			val: struct {
				F string `json:"f" validate:"required"`
			}{F: ""},
			wantMsg: "is required",
		},
		{
			name: "email",
			val: struct {
				F string `json:"f" validate:"required,email"`
			}{F: "bad"},
			wantMsg: "must be a valid email address",
		},
		{
			name: "min_string",
			val: struct {
				F string `json:"f" validate:"min=5"`
			}{F: "ab"},
			wantMsg: "must be at least 5 characters",
		},
		{
			name: "max_string",
			val: struct {
				F string `json:"f" validate:"max=2"`
			}{F: "abcd"},
			wantMsg: "must be at most 2 characters",
		},
		{
			name: "min_int",
			val: struct {
				F int `json:"f" validate:"min=10"`
			}{F: 3},
			wantMsg: "must be at least 10",
		},
		{
			name: "max_int",
			val: struct {
				F int `json:"f" validate:"max=5"`
			}{F: 10},
			wantMsg: "must be at most 5",
		},
		{
			name: "url",
			val: struct {
				F string `json:"f" validate:"url"`
			}{F: "not-a-url"},
			wantMsg: "must be a valid URL",
		},
		{
			name: "uuid",
			val: struct {
				F string `json:"f" validate:"uuid"`
			}{F: "not-uuid"},
			wantMsg: "must be a valid UUID",
		},
		{
			name: "alphanum",
			val: struct {
				F string `json:"f" validate:"alphanum"`
			}{F: "abc-123!"},
			wantMsg: "must contain only alphanumeric characters",
		},
		{
			name: "oneof",
			val: struct {
				F string `json:"f" validate:"oneof=red green blue"`
			}{F: "yellow"},
			wantMsg: "must be one of: red green blue",
		},
		{
			name: "gte",
			val: struct {
				F int `json:"f" validate:"gte=5"`
			}{F: 2},
			wantMsg: "must be greater than or equal to 5",
		},
		{
			name: "lte",
			val: struct {
				F int `json:"f" validate:"lte=5"`
			}{F: 10},
			wantMsg: "must be less than or equal to 5",
		},
		{
			name: "gt",
			val: struct {
				F int `json:"f" validate:"gt=5"`
			}{F: 5},
			wantMsg: "must be greater than 5",
		},
		{
			name: "lt",
			val: struct {
				F int `json:"f" validate:"lt=5"`
			}{F: 5},
			wantMsg: "must be less than 5",
		},
		{
			name: "len",
			val: struct {
				F string `json:"f" validate:"len=3"`
			}{F: "ab"},
			wantMsg: "must be exactly 3 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Struct(tc.val)
			if err == nil {
				t.Fatal("expected error")
			}
			var ve ValidationErrors
			if !errors.As(err, &ve) {
				t.Fatalf("expected ValidationErrors, got %T", err)
			}
			// Find the relevant error (may have multiple if required also fails).
			var found bool
			for _, fe := range ve {
				if fe.Message == tc.wantMsg {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected message %q in errors: %v", tc.wantMsg, ve)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 12. BindAndValidate
// ---------------------------------------------------------------------------

func TestBindAndValidate_BindFails(t *testing.T) {
	v := New()
	bindErr := errors.New("cannot decode JSON")
	bind := func(dst any) error { return bindErr }

	var req CreateUserReq
	err := BindAndValidate(bind, &req, v)
	if err == nil {
		t.Fatal("expected error")
	}
	var be *BindError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BindError, got %T", err)
	}
	if be.HTTPStatus() != 400 {
		t.Errorf("expected 400, got %d", be.HTTPStatus())
	}
	if !errors.Is(be, bindErr) {
		t.Error("expected BindError to wrap the original bind error")
	}
}

func TestBindAndValidate_ValidationFails(t *testing.T) {
	v := New()
	// Bind succeeds but leaves required fields empty.
	bind := func(dst any) error { return nil }

	var req CreateUserReq
	err := BindAndValidate(bind, &req, v)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if ve.HTTPStatus() != 422 {
		t.Errorf("expected 422, got %d", ve.HTTPStatus())
	}
}

func TestBindAndValidate_Success(t *testing.T) {
	v := New()
	bind := func(dst any) error {
		r := dst.(*CreateUserReq)
		r.Name = "Bob"
		r.Email = "bob@example.com"
		r.Age = 25
		return nil
	}

	var req CreateUserReq
	if err := BindAndValidate(bind, &req, v); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if req.Name != "Bob" {
		t.Errorf("expected name 'Bob', got %q", req.Name)
	}
}

// ---------------------------------------------------------------------------
// 13. Handle — generic wrapper (uses a mock Binder)
// ---------------------------------------------------------------------------

type mockBinder struct {
	bindFunc func(any) error
}

func (m *mockBinder) Bind(dst any) error { return m.bindFunc(dst) }

func TestHandle_Success(t *testing.T) {
	v := New()
	var received *CreateUserReq
	handler := Handle[*mockBinder, CreateUserReq](v, func(ctx *mockBinder, req *CreateUserReq) error {
		received = req
		return nil
	})

	ctx := &mockBinder{bindFunc: func(dst any) error {
		r := dst.(*CreateUserReq)
		r.Name = "Alice"
		r.Email = "alice@test.com"
		r.Age = 28
		return nil
	}}

	if err := handler(ctx); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if received == nil {
		t.Fatal("handler was not called")
	}
	if received.Name != "Alice" {
		t.Errorf("expected Name='Alice', got %q", received.Name)
	}
}

func TestHandle_BindError(t *testing.T) {
	v := New()
	handler := Handle[*mockBinder, CreateUserReq](v, func(ctx *mockBinder, req *CreateUserReq) error {
		t.Fatal("handler should not be called on bind error")
		return nil
	})

	ctx := &mockBinder{bindFunc: func(dst any) error {
		return errors.New("parse error")
	}}

	err := handler(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	var be *BindError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BindError, got %T", err)
	}
}

func TestHandle_ValidationError(t *testing.T) {
	v := New()
	handler := Handle[*mockBinder, CreateUserReq](v, func(ctx *mockBinder, req *CreateUserReq) error {
		t.Fatal("handler should not be called on validation error")
		return nil
	})

	ctx := &mockBinder{bindFunc: func(dst any) error {
		// Bind succeeds but leaves invalid data.
		return nil
	}}

	err := handler(ctx)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
}

// ---------------------------------------------------------------------------
// 14. RegisterAlias
// ---------------------------------------------------------------------------

func TestRegisterAlias(t *testing.T) {
	v := New()
	v.RegisterAlias("isColor", "hexcolor|rgb|rgba")

	type Req struct {
		Color string `json:"color" validate:"required,isColor"`
	}
	if err := v.Struct(Req{Color: "#ff0000"}); err != nil {
		t.Fatalf("expected valid hex color to pass, got %v", err)
	}
	err := v.Struct(Req{Color: "not-a-color"})
	if err == nil {
		t.Fatal("expected error for invalid color")
	}
}

// ---------------------------------------------------------------------------
// 15. Edge cases
// ---------------------------------------------------------------------------

func TestValidationErrors_ImplementsErrorInterface(t *testing.T) {
	var err error = ValidationErrors{{Field: "f", Message: "m"}}
	_ = err.Error() // must compile
}

func TestValidationErrors_EmptySlice(t *testing.T) {
	ve := ValidationErrors{}
	if got := ve.HTTPStatus(); got != 422 {
		t.Errorf("even empty slice returns 422, got %d", got)
	}
	if got := ve.BizCode(); got != 42200 {
		t.Errorf("even empty slice returns 42200, got %d", got)
	}
	data := ve.ErrorData()
	slice := data.([]FieldError)
	if len(slice) != 0 {
		t.Errorf("expected empty slice from ErrorData, got %v", slice)
	}
}

func TestMultipleOptions(t *testing.T) {
	v := New(
		WithRule("even", func(val any) bool {
			n, ok := val.(int)
			return ok && n%2 == 0
		}),
		WithRule("positive", func(val any) bool {
			n, ok := val.(int)
			return ok && n > 0
		}),
	)

	type Req struct {
		Val int `json:"val" validate:"even,positive"`
	}

	if err := v.Struct(Req{Val: 4}); err != nil {
		t.Fatalf("expected 4 to pass even+positive, got %v", err)
	}
	err := v.Struct(Req{Val: -2})
	if err == nil {
		t.Fatal("expected error for -2 (not positive)")
	}
	err = v.Struct(Req{Val: 3})
	if err == nil {
		t.Fatal("expected error for 3 (not even)")
	}
}

func TestJSONTagWithOmitempty(t *testing.T) {
	v := New()
	type Req struct {
		Name string `json:"name,omitempty" validate:"required"`
	}
	err := v.Struct(Req{Name: ""})
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve) == 0 {
		t.Fatal("expected error")
	}
	// Should use "name", not "name,omitempty"
	if ve[0].Field != "name" {
		t.Errorf("expected field 'name', got %q", ve[0].Field)
	}
}

func TestJSONTagDash_UsesStructName(t *testing.T) {
	v := New()
	type Req struct {
		Internal string `json:"-" validate:"required"`
	}
	err := v.Struct(Req{Internal: ""})
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve) == 0 {
		t.Fatal("expected error")
	}
	// json:"-" should fall back to struct field name
	if ve[0].Field != "Internal" {
		t.Errorf("expected field 'Internal', got %q", ve[0].Field)
	}
}

// ---------------------------------------------------------------------------
// 16. Duck-type interface compliance (compile-time checks)
// ---------------------------------------------------------------------------

// Ensure ValidationErrors satisfies the duck-type interfaces used by response encoder.
type httpStatuser interface{ HTTPStatus() int }
type bizCoder interface{ BizCode() int }
type errorDataProvider interface{ ErrorData() any }

var (
	_ error             = ValidationErrors{}
	_ httpStatuser      = ValidationErrors{}
	_ bizCoder          = ValidationErrors{}
	_ errorDataProvider = ValidationErrors{}

	_ error        = (*BindError)(nil)
	_ httpStatuser = (*BindError)(nil)
	_ bizCoder     = (*BindError)(nil)
)

// Ensure fmt import is used.
var _ = fmt.Sprintf
