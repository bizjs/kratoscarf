# validation — 请求验证

基于 [go-playground/validator](https://github.com/go-playground/validator) 封装，提供 struct tag 驱动的验证能力，与 kratoscarf router 深度集成。

## 包结构

```
validation/
├── validation.go   # Validator, New, Struct, Var, Validate, 选项
├── errors.go       # FieldError, ValidationErrors, BindError, Translate
└── handle.go       # Binder 接口, Handle 泛型包装, BindAndValidate
```

## 三种使用方式

### 方式一：Gin 风格 — Bind 自动验证（推荐）

设置一次，所有 handler 零验证代码。

```go
import (
    "github.com/bizjs/kratoscarf/router"
    "github.com/bizjs/kratoscarf/validation"
)

// 创建 router 时设置 validator — 之后 ctx.Bind() 自动验证
r := router.NewRouter(srv, router.WithValidator(validation.New()))
```

Handler 只需 `ctx.Bind()`, 验证自动完成:

```go
type CreateRequest struct {
    Title string `json:"title" validate:"required,min=1,max=200"`
    Email string `json:"email" validate:"required,email"`
}

func (s *Svc) Create(ctx *router.Context) error {
    var req CreateRequest
    if err := ctx.Bind(&req); err != nil {
        return err // bind 失败 → 400, 验证失败 → 422
    }
    // req 已验证, 直接使用
}
```

### 方式二：NestJS 风格 — 泛型 Handler 包装

Handler 签名声明需要的请求类型, `validation.Handle` 自动 bind + validate, 验证失败 handler 不执行:

```go
import "github.com/bizjs/kratoscarf/validation"

v := validation.New()

r.POST("/todos", validation.Handle(v, s.Create))

func (s *Svc) Create(ctx *router.Context, req *CreateRequest) error {
    // req 已经 bind + validate 过了
    // 如果验证失败, 这个函数根本不会被调用
}
```

### 方式三：手动调用

不使用 router, 或需要更细粒度控制时:

```go
v := validation.New()

// 方式 A: 分步调用
if err := ctx.Bind(&req); err != nil { return err }
if err := v.Validate(&req); err != nil { return err }

// 方式 B: 一步完成
if err := validation.BindAndValidate(ctx.Bind, &req, v); err != nil {
    return err
}

// 方式 C: 只验证, 不绑定
if err := v.Struct(&req); err != nil { ... }

// 方式 D: 验证单个变量
if err := v.Var(email, "required,email"); err != nil { ... }
```

## 内置验证规则

所有 [go-playground/validator](https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Baked_In_Validators_and_Tags) 内置规则均可使用:

```go
type User struct {
    Username string  `json:"username" validate:"required,min=3,max=32,alphanum"`
    Email    string  `json:"email"    validate:"required,email"`
    Password string  `json:"password" validate:"required,min=8,max=128"`
    Age      int     `json:"age"      validate:"omitempty,gte=0,lte=150"`
    Role     string  `json:"role"     validate:"required,oneof=admin user guest"`
    Website  string  `json:"website"  validate:"omitempty,url"`
    Code     string  `json:"code"     validate:"omitempty,len=6"`
    ID       string  `json:"id"       validate:"omitempty,uuid"`
}
```

常用规则速查:

| 规则 | 含义 | 适用类型 |
|------|------|---------|
| `required` | 不能为零值 | 所有 |
| `omitempty` | 零值时跳过后续规则 | 所有 |
| `min=N` | 最小长度/值 | string/number/slice |
| `max=N` | 最大长度/值 | string/number/slice |
| `len=N` | 精确长度 | string/slice |
| `gte=N` / `lte=N` | 大于等于/小于等于 | number |
| `gt=N` / `lt=N` | 大于/小于 | number |
| `email` | 邮箱格式 | string |
| `url` | URL 格式 | string |
| `uuid` | UUID 格式 | string |
| `alphanum` | 字母+数字 | string |
| `oneof=a b c` | 枚举值 | string/number |

## 自定义验证规则

### 简单规则 — 只判断值

```go
v := validation.New(
    validation.WithRule("even", func(val any) bool {
        n, ok := val.(int)
        return ok && n%2 == 0
    }),
    validation.WithRule("not_empty_slice", func(val any) bool {
        rv := reflect.ValueOf(val)
        return rv.Kind() == reflect.Slice && rv.Len() > 0
    }),
)

type Req struct {
    Count int      `json:"count" validate:"required,even"`
    Tags  []string `json:"tags"  validate:"not_empty_slice"`
}
```

### 高级规则 — 跨字段验证

使用 `WithRuleFunc` 获取完整的 `validator.FieldLevel` API:

```go
import "github.com/go-playground/validator/v10"

v := validation.New(
    validation.WithRuleFunc("gtfield", func(fl validator.FieldLevel) bool {
        field := fl.Field().Int()
        other := fl.Parent().FieldByName(fl.Param()).Int()
        return field > other
    }),
)
```

### 规则别名

组合已有规则:

```go
v := validation.New()
v.RegisterAlias("isColor", "hexcolor|rgb|rgba")

type Theme struct {
    Primary string `json:"primary" validate:"required,isColor"`
}
```

### 自定义 tag 名

默认使用 `validate` tag, 可改为 `binding` (Gin 风格):

```go
v := validation.New(validation.WithTagName("binding"))

type Req struct {
    Name string `json:"name" binding:"required"`
}
```

## 错误响应格式

验证失败返回 HTTP 422, 带结构化字段错误:

```json
{
    "code": 42200,
    "message": "validation failed",
    "data": [
        {"field": "title", "rule": "required", "message": "is required"},
        {"field": "email", "rule": "email", "message": "must be a valid email address"}
    ]
}
```

绑定失败 (JSON 格式错误等) 返回 HTTP 400:

```json
{
    "code": 40000,
    "message": "invalid character '}' looking for beginning of value"
}
```

### 错误类型

| 类型 | HTTP 状态码 | 业务码 | 说明 |
|------|-----------|--------|------|
| `ValidationErrors` | 422 | 42200 | 字段验证失败, 携带 `[]FieldError` |
| `BindError` | 400 | 40000 | 请求体解析失败 |

### 错误处理协议

`ValidationErrors` 和 `BindError` 通过鸭子类型与 `response.NewHTTPErrorEncoder` 协作, validation 不 import response:

```go
// validation 包的错误实现这些方法
type ValidationErrors []FieldError
func (e ValidationErrors) HTTPStatus() int  { return 422 }
func (e ValidationErrors) BizCode() int     { return 42200 }
func (e ValidationErrors) ErrorData() any   { return []FieldError(e) }

// response encoder 通过接口检查
if hs, ok := err.(interface{ HTTPStatus() int }); ok { httpCode = hs.HTTPStatus() }
if bc, ok := err.(interface{ BizCode() int }); ok { code = bc.BizCode() }
if ed, ok := err.(interface{ ErrorData() any }); ok { data = ed.ErrorData() }
```

任何自定义错误类型实现这些方法, encoder 即可自动处理。

## 错误消息

默认提供中英文友好的错误消息。字段名使用 JSON tag (而非 Go struct field name):

```go
type Req struct {
    UserName string `json:"user_name" validate:"required"`
}
// 错误消息: {"field": "user_name", ...}  (不是 "UserName")
```

数字类型和字符串类型的 min/max 消息会自动区分:

```go
type Req struct {
    Name string `json:"name" validate:"min=3"`   // "must be at least 3 characters"
    Age  int    `json:"age"  validate:"min=18"`  // "must be at least 18"
}
```

## 与 Kratos proto 服务集成

对于使用 proto 生成的 Kratos HTTP handler, 在 service 实现中手动调用:

```go
func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.TokenPair, error) {
    // proto 生成的 handler 已经做了 Bind, 这里只做验证
    if err := s.validator.Validate(req); err != nil {
        return nil, err
    }
    // ...
}
```

## 依赖关系

```
router      →  (无内部依赖, 通过 StructValidator 接口对接)
validation  →  (无内部依赖, 通过鸭子类型对接 response encoder)
response    →  (无内部依赖)
```

三个包完全独立, 可单独使用。
