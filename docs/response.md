# response — 统一响应与错误协议

定义 API 响应格式、业务错误体系、分页数据结构，以及 Kratos HTTP 编码器。是整个应用的"协议层"——所有包通过它（或它的鸭子类型接口）达成一致的 HTTP 输出格式。

## 包结构

```
response/
├── response.go    # Response, Success, Wrap, Error
├── errors.go      # BizError, 预定义错误, IsBizError, FromKratosError
├── encoder.go     # NewHTTPResponseEncoder, NewHTTPErrorEncoder
└── pagination.go  # PageRequest/Response, CursorRequest/Response
```

## 统一响应格式

所有 API 响应遵循同一结构：

```json
{
    "code": 0,
    "message": "ok",
    "data": { ... }
}
```

| 字段 | 说明 |
|------|------|
| `code` | 业务码，`0` = 成功，非零 = 错误 |
| `message` | 人类可读消息 |
| `data` | 响应载荷，错误时可携带结构化错误详情 |

### 构造响应

```go
// 成功
response.Success(data)
// → {code: 0, message: "ok", data: ...}

response.SuccessWithMessage(data, "created")
// → {code: 0, message: "created", data: ...}

// 从错误构造
response.Error(err)
// BizError → {code: 40400, message: "not found"}
// 其他 error → {code: 50000, message: "internal server error"}
```

### Wrap — 用于 router 自动包装

```go
// router 初始化时设置一次
r := router.NewRouter(srv, router.WithResponseWrapper(response.Wrap))

// handler 中 ctx.Success(data) 自动调用 Wrap
func handler(ctx *router.Context) error {
    return ctx.Success(todos) // → {code:0, message:"ok", data:[...]}
}
```

`Wrap` 和 `Success` 功能相同，区别仅在返回类型（`any` vs `*Response`），`Wrap` 专为 `WithResponseWrapper` 设计。

## 业务错误

### BizError

```go
type BizError struct {
    HTTPCode int    // HTTP 状态码
    Code     int    // 业务错误码
    Message  string // 错误消息
    Data     any    // 可选结构化数据（如验证字段错误）
}
```

### 预定义错误

| 变量 | HTTP | 业务码 | 消息 |
|------|------|--------|------|
| `ErrBadRequest` | 400 | 40000 | bad request |
| `ErrUnauthorized` | 401 | 40100 | unauthorized |
| `ErrForbidden` | 403 | 40300 | forbidden |
| `ErrNotFound` | 404 | 40400 | resource not found |
| `ErrConflict` | 409 | 40900 | conflict |
| `ErrValidation` | 422 | 42200 | validation failed |
| `ErrTooManyRequests` | 429 | 42900 | too many requests |
| `ErrInternal` | 500 | 50000 | internal server error |
| `ErrServiceUnavailable` | 503 | 50300 | service unavailable |

### 用法

```go
// 直接返回 — ErrorEncoder 自动设置 HTTP 状态码和 JSON 响应
return response.ErrNotFound
// → HTTP 404 {"code": 40400, "message": "resource not found"}

// 自定义消息
return response.ErrNotFound.WithMessage("user 123 not found")
// → HTTP 404 {"code": 40400, "message": "user 123 not found"}

// 包装内部错误（不会暴露给客户端）
return response.ErrInternal.WithCause(dbErr)
// → HTTP 500 {"code": 50000, "message": "internal server error"}
// 日志中可通过 errors.Unwrap 获取原始错误
```

### 自定义业务错误

```go
var ErrDuplicate = response.NewBizError(409, 40901, "duplicate entry")
var ErrQuotaExceeded = response.NewBizError(403, 40301, "quota exceeded")

// 使用
return ErrDuplicate.WithMessage("username already taken")
```

### 安全设计

`WithCause(err)` 包装的内部错误**不会**出现在 HTTP 响应中。客户端只看到 `Message`，`cause` 仅用于服务端日志和 `errors.Unwrap` 链。

```go
return response.ErrInternal.WithCause(fmt.Errorf("db: connection refused"))
// 客户端看到: {"code": 50000, "message": "internal server error"}
// 服务端日志: "internal server error: db: connection refused"
```

### 错误判断

```go
bizErr, ok := response.IsBizError(err)
if ok {
    log.Printf("biz code: %d, http: %d", bizErr.Code, bizErr.HTTPCode)
}

// 从 Kratos 错误转换
bizErr := response.FromKratosError(kratosErr)
```

## Kratos 编码器

### ResponseEncoder — 成功响应

用于 proto 生成的 Kratos handler，自动将返回值包装为统一格式：

```go
httpSrv := kratoshttp.NewServer(
    kratoshttp.ResponseEncoder(response.NewHTTPResponseEncoder()),
)
```

proto handler 返回的 reply 会被包装：

```go
// proto service 实现
func (s *Svc) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.User, error) {
    return &v1.User{Name: "alice"}, nil
}
// → {"code": 0, "message": "ok", "data": {"name": "alice"}}
```

#### 自定义成功响应结构

通过 `WithSuccessWrapper` 替换默认的 `{code, message, data}` 信封。签名与 `router.WithResponseWrapper` 一致（`func(data any) any`），可传同一个函数：

```go
myWrapper := func(data any) any {
    return map[string]any{"success": true, "result": data}
}

httpSrv := kratoshttp.NewServer(
    kratoshttp.ResponseEncoder(response.NewHTTPResponseEncoder(
        response.WithSuccessWrapper(myWrapper),
    )),
)
// proto handler 返回 → {"success": true, "result": {"name": "alice"}}

r := router.NewRouter(httpSrv,
    router.WithResponseWrapper(myWrapper), // 同一个函数，两处复用
)
// ctx.Success(data) → 同样的信封结构
```

不传 `WithSuccessWrapper` 时行为与之前完全一致（默认 `response.Success`）。

### ErrorEncoder — 错误响应

将 handler 返回的 error 格式化为统一响应，支持三种错误识别方式：

```go
httpSrv := kratoshttp.NewServer(
    kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()),
)
```

识别优先级：

```
1. BizError        → 使用 HTTPCode / Code / Message / Data
2. HTTPStatus() int → 鸭子类型 HTTP 状态码（如 ValidationErrors → 422）
3. BizCode() int    → 鸭子类型业务码
4. ErrorData() any  → 鸭子类型结构化数据
5. 其他 error       → 500 "internal server error"（不泄露内部信息）
```

#### 自定义错误响应结构

通过 `WithErrorWrapper` 替换默认的错误信封。可在自定义函数内调用 `ErrorToResponse(err)` 复用默认的鸭子类型解析逻辑：

```go
httpSrv := kratoshttp.NewServer(
    kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder(
        response.WithErrorWrapper(func(err error) any {
            resp := response.ErrorToResponse(err) // 复用鸭子类型逻辑
            return map[string]any{
                "success": false,
                "error":   resp.Message,
                "code":    resp.Code,
            }
        }),
    )),
)
// BizError → {"success": false, "error": "not found", "code": 40400}
```

HTTP 状态码始终由鸭子类型决定（`BizError.HTTPCode` / `HTTPStatus()` 接口），不受 `WithErrorWrapper` 影响。

不传 `WithErrorWrapper` 时行为与之前完全一致。

### ErrorToResponse — 导出的错误转换

`ErrorToResponse(err error) *Response` 将 error 通过鸭子类型转换为 `*Response`。主要用途：

- 在 `WithErrorWrapper` 自定义函数中复用默认解析逻辑
- 在需要手动构造错误响应的场景中使用

```go
resp := response.ErrorToResponse(err)
// resp.Code    — 业务码（BizError.Code 或 BizCode() 鸭子类型）
// resp.Message — 消息
// resp.Data    — 结构化数据（ErrorData() 鸭子类型）
```

### 鸭子类型协议

任何 error 实现以下方法即可被 ErrorEncoder 识别，**不需要 import response**：

```go
// 你的自定义错误类型
type MyError struct { ... }

func (e *MyError) Error() string     { return "..." }
func (e *MyError) HTTPStatus() int   { return 429 }   // → HTTP 429
func (e *MyError) BizCode() int      { return 42900 } // → code: 42900
func (e *MyError) ErrorData() any    { return details } // → data: details
```

`validation.ValidationErrors` 就是这样接入的——它不 import response，但 ErrorEncoder 能正确处理。

## 分页

### Offset 分页（传统）

```go
// 请求
req := response.PageRequest{Page: 1, PageSize: 20, Sort: "created_at desc"}
req.Normalize(20, 100) // 填充默认值，限制上界
offset := req.Offset() // → 0

// 数据层
items, total := queryFromDB(req.Offset(), req.PageSize)

// 响应
result := response.NewPageResponse(items, total, req)
return ctx.Success(result)
```

输出：
```json
{
    "code": 0, "message": "ok",
    "data": {
        "items": [...],
        "total": 100,
        "page": 1,
        "pageSize": 20
    }
}
```

`PageRequest.Normalize(defaultSize, maxSize)` 行为：
- `Page < 1` → 设为 1
- `PageSize <= 0` → 设为 defaultSize
- `PageSize > maxSize` → 设为 maxSize

### Cursor 分页（适合大数据量/实时流）

```go
// 请求
req := response.CursorRequest{Cursor: "abc123", Limit: 20}

// 数据层
items, nextCursor, hasMore := queryWithCursor(req.Cursor, req.Limit)

// 响应
result := response.NewCursorResponse(items, nextCursor, hasMore)
return ctx.Success(result)
```

输出：
```json
{
    "code": 0, "message": "ok",
    "data": {
        "items": [...],
        "nextCursor": "def456",
        "hasMore": true
    }
}
```

## 依赖关系

```
response  →  Kratos (仅 encoder 用到 kratoshttp 类型)
```

response 不 import 项目内任何其他包。其他包通过以下方式与 response 协作：

| 包 | 协作方式 |
|----|---------|
| router | `WithResponseWrapper(response.Wrap)` — 函数引用传递 |
| validation | 鸭子类型 `HTTPStatus()` / `BizCode()` / `ErrorData()` |
| handler 代码 | 直接 import，使用 `BizError` 和 `PageResponse` |
