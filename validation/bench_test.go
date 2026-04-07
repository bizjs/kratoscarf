package validation

import (
	"testing"
)

type benchUser struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
	Age   int    `validate:"required,gte=0,lte=150"`
}

type benchAddress struct {
	Street string `validate:"required"`
	City   string `validate:"required"`
	Zip    string `validate:"required,len=5"`
}

type benchOrder struct {
	UserID  string       `validate:"required,uuid"`
	Items   []string     `validate:"required,min=1,max=100"`
	Address benchAddress `validate:"required"`
	Note    string       `validate:"max=500"`
}

var benchValidator = New()

func BenchmarkValidate_SimpleStruct_Valid(b *testing.B) {
	user := benchUser{Name: "Alice", Email: "alice@example.com", Age: 30}
	b.ReportAllocs()
	for b.Loop() {
		_ = benchValidator.Validate(user)
	}
}

func BenchmarkValidate_SimpleStruct_Invalid(b *testing.B) {
	user := benchUser{Name: "", Email: "not-email", Age: -1}
	b.ReportAllocs()
	for b.Loop() {
		_ = benchValidator.Validate(user)
	}
}

func BenchmarkValidate_NestedStruct_Valid(b *testing.B) {
	order := benchOrder{
		UserID:  "550e8400-e29b-41d4-a716-446655440000",
		Items:   []string{"item1", "item2", "item3"},
		Address: benchAddress{Street: "123 Main St", City: "Springfield", Zip: "62704"},
		Note:    "please deliver before noon",
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = benchValidator.Validate(order)
	}
}

func BenchmarkValidate_NestedStruct_Invalid(b *testing.B) {
	order := benchOrder{
		UserID:  "not-a-uuid",
		Items:   nil,
		Address: benchAddress{Street: "", City: "", Zip: "1"},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = benchValidator.Validate(order)
	}
}

func BenchmarkVar_Email(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchValidator.Var("alice@example.com", "required,email")
	}
}

func BenchmarkVar_UUID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchValidator.Var("550e8400-e29b-41d4-a716-446655440000", "required,uuid")
	}
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = New()
	}
}

func BenchmarkValidationErrors_Error(b *testing.B) {
	errs := ValidationErrors{
		{Field: "name", Rule: "required", Message: "name is required"},
		{Field: "email", Rule: "email", Message: "email must be a valid email"},
		{Field: "age", Rule: "gte", Message: "age must be 0 or greater"},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = errs.Error()
	}
}
