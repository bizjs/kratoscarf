# router — Web 风格路由

对 Kratos HTTP Server 的薄封装，提供 Gin/Echo 风格的路由 API。不替代 Kratos，与 Kratos 原生路由和 proto 生成的 handler 共存。

## 设计原则

1. **薄封装** — 底层全部委托给 Kratos `http.Server` 和 gorilla/mux，不重新实现路由匹配
2. **零内部依赖** — 不 import response、validation 等包，通过接口（`StructValidator`）可选对接
3. **Kratos 中间件兼容** — `Use()` 和 `Group()` 接受标准 `kratosmiddleware.Middleware`
4. **handler 返回 error** — 错误由 Kratos 的 `ErrorEncoder` 统一处理

## 包结构

```
router/
├── router.go    # Router, NewRouter, Group, Use, Handle, HTTP 方法快捷方式
└── context.go   # Context, 请求/响应辅助方法, Bind (auto-validate)
```

## 基本用法

```go
import (
    "github.com/bizjs/kratoscarf/router"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

httpSrv := kratoshttp.NewServer(
    kratoshttp.Address(":8080"),
    kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()),
)

r := router.NewRouter(httpSrv)

r.GET("/hello", func(ctx *router.Context) error {
    return ctx.JSON(200, map[string]string{"message": "hello"})
})
```

## Router API

### 创建

```go
// 基本
r := router.NewRouter(srv)

// 全功能 — 自动验证 + 自动响应包装 + 自动错误处理
r := router.NewRouter(srv,
    router.WithValidator(validation.New()),    // Bind() 自动验证
    router.WithResponseWrapper(response.Wrap), // Success() 自动包装 {code, message, data}
)
```

### 路由注册

```go
r.GET(path, handler)
r.POST(path, handler)
r.PUT(path, handler)
r.DELETE(path, handler)
r.PATCH(path, handler)
r.HEAD(path, handler)
r.OPTIONS(path, handler)

// 任意方法
r.Handle("GET", path, handler)
```

路径参数使用 gorilla/mux 语法：

```go
r.GET("/users/{id}", func(ctx *router.Context) error {
    id := ctx.Param("id")
    // ...
})

r.GET("/files/{path:.*}", func(ctx *router.Context) error {
    path := ctx.Param("path") // 匹配 "a/b/c"
    // ...
})
```

### 路由分组

```go
api := r.Group("/api/v1")
api.GET("/users", listUsers)
api.POST("/users", createUser)

// 带中间件的分组
admin := r.Group("/admin", authMiddleware, roleMiddleware)
admin.GET("/dashboard", dashboard)
```

Group 继承父路由的：
- 路径前缀（拼接）
- 中间件（追加）
- Validator（继承）
- ResponseWrapper（继承）

### 中间件

```go
// 全局中间件
r.Use(recovery, accessLog)

// 分组中间件
api := r.Group("/api", jwtMiddleware)

// 中间件签名 — 标准 Kratos 中间件
type Middleware func(Handler) Handler
type Handler func(ctx context.Context, req any) (any, error)
```

**重要：`Use()` 只影响之后注册的路由。已注册的路由不受影响。**

```go
r.GET("/public", handler)  // 无中间件
r.Use(authMiddleware)
r.GET("/private", handler) // 有 authMiddleware
```

## Context API

### 请求读取

```go
// 路径参数
id := ctx.Param("id")

// 查询参数
page := ctx.Query("page")
sort := ctx.QueryDefault("sort", "created_at")

// 请求头
token := ctx.Header("Authorization")

// Cookie
cookie, err := ctx.Cookie("session_id")

// 原始对象
req := ctx.Request()       // *http.Request
goCtx := ctx.Context()     // context.Context
```

### 请求绑定

```go
var req CreateUserRequest
if err := ctx.Bind(&req); err != nil {
    return err
}
```

`Bind` 行为：
1. 根据 Content-Type 自动解码（JSON、Form、Protobuf）— 委托给 Kratos
2. 如果 Router 设置了 Validator，自动验证 struct tag 规则
3. 绑定失败返回错误，验证失败返回 `ValidationErrors`（422）

```go
// BindQuery — 从查询参数绑定
var filter FilterRequest
if err := ctx.BindQuery(&filter); err != nil {
    return err
}
```

### 响应写入

```go
// Success — 如果设置了 ResponseWrapper，自动包装
ctx.Success(data)
// 无 wrapper → {"id": 1, "title": "..."}
// 有 wrapper → {"code": 0, "message": "ok", "data": {"id": 1, "title": "..."}}

// JSON — 直接写，不走 wrapper（自定义状态码时使用）
ctx.JSON(200, data)
ctx.JSON(201, response.Success(todo))

// 其他
ctx.NoContent()            // 204

// 重定向
ctx.Redirect(301, "/new-url")
ctx.Redirect(http.StatusFound, "/login")

// 流式响应
ctx.Stream("text/csv", csvReader)

// 响应头
ctx.SetHeader("X-Request-Id", id)

// Cookie
ctx.SetCookie(&http.Cookie{
    Name:  "token",
    Value: "abc",
    Path:  "/",
})
```

### Context 值传递

```go
// 中间件中设置
ctx.SetValue(userKey{}, user)

// Handler 中读取
user := ctx.GetValue(userKey{})
```

## 中间件与 Context 传播

Router 的中间件链通过 `context.Context` 传播值。中间件修改的 context 在 handler 中可通过 `ctx.Context()` 访问：

```go
// 中间件注入值
func myMiddleware() kratosmiddleware.Middleware {
    return func(handler kratosmiddleware.Handler) kratosmiddleware.Handler {
        return func(ctx context.Context, req any) (any, error) {
            ctx = context.WithValue(ctx, myKey{}, "value")
            return handler(ctx, req)
        }
    }
}

// Handler 读取
func myHandler(ctx *router.Context) error {
    val := ctx.Context().Value(myKey{})  // "value"
    // ...
}
```

### 已知限制

**Kratos server-level 中间件不会自动传播到 route handler。** 这是 Kratos 的设计 — transport context 和 request context 是分离的。

解决方案：
- 使用 `r.Group("/path", middleware)` 通过 router 挂载中间件（推荐）
- 使用 `r.Use(middleware)` 全局挂载
- 如果使用 proto 生成的 handler，用 Kratos 原生 `http.Middleware()` 注册

## 与 Kratos 原生路由共存

kratoscarf router 和 Kratos 原生路由注册在同一个 `http.Server` 上，互不冲突：

```go
httpSrv := kratoshttp.NewServer(...)

// kratoscarf router — 手写路由
r := router.NewRouter(httpSrv)
r.GET("/health", healthHandler)
r.POST("/api/todos", createTodo)

// Kratos 原生 — proto 生成的路由
v1.RegisterUserServiceHTTPServer(httpSrv, userService)
```

## 与 response 包配合

Router 不 import response，通过 `WithResponseWrapper` 可选集成。

### 推荐：设置 ResponseWrapper，handler 零包装代码

```go
// 初始化
r := router.NewRouter(srv,
    router.WithResponseWrapper(response.Wrap),
    router.WithValidator(validation.New()),
)

// Handler — 直接返回数据，自动包装
func listTodos(ctx *router.Context) error {
    return ctx.Success(todos)
    // → {"code": 0, "message": "ok", "data": [...]}
}

func getTodo(ctx *router.Context) error {
    // 返回业务错误 — ErrorEncoder 自动处理
    return response.ErrNotFound.WithMessage("todo not found")
    // → HTTP 404 {"code": 40400, "message": "todo not found"}
}

func createTodo(ctx *router.Context) error {
    // 自定义状态码 — 用 JSON() 绕过 wrapper
    return ctx.JSON(201, response.Success(todo))
}
```

### 不设置 ResponseWrapper

`ctx.Success(data)` 直接输出 data 的 JSON，不包装：

```go
r := router.NewRouter(srv) // 无 wrapper

func handler(ctx *router.Context) error {
    return ctx.Success(map[string]string{"name": "alice"})
    // → {"name": "alice"}
}
```

## 与 validation 包配合

两种集成方式，见 [docs/validation.md](validation.md)：

```go
// 方式一：Gin 风格 — Bind 自动验证
r := router.NewRouter(srv, router.WithValidator(validation.New()))
ctx.Bind(&req) // 自动验证

// 方式二：NestJS 风格 — Handle 泛型包装
r.POST("/todos", validation.Handle(v, s.Create))
```

## 错误处理流程

```
Handler 返回 error
    ↓
Kratos HTTP Server 捕获
    ↓
ErrorEncoder 处理 (response.NewHTTPErrorEncoder)
    ↓
检查错误类型:
  BizError       → 使用 HTTPCode/Code/Message
  HTTPStatus()   → 使用鸭子类型的状态码 (如 ValidationErrors → 422)
  其他 error     → 500 "internal server error"
```

Handler 不需要自己写 JSON 错误响应，直接 `return err` 即可。

## 类 NestJS 全自动拦截

三个选项组合实现 NestJS 式的请求/响应自动处理：

```go
r := router.NewRouter(srv,
    router.WithValidator(validation.New()),    // ≈ NestJS ValidationPipe
    router.WithResponseWrapper(response.Wrap), // ≈ NestJS TransformInterceptor
)
// + kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()) ≈ NestJS ExceptionFilter
```

| NestJS | kratoscarf | 效果 |
|--------|-----------|------|
| `ValidationPipe` | `WithValidator` | `ctx.Bind()` 自动验证，失败返回 422 |
| `TransformInterceptor` | `WithResponseWrapper` | `ctx.Success()` 自动包装 `{code, message, data}` |
| `HttpExceptionFilter` | `ErrorEncoder` | `return err` 自动格式化错误响应 |

Handler 只需关注业务逻辑：

```go
func (s *Svc) Create(ctx *router.Context) error {
    var req CreateRequest
    if err := ctx.Bind(&req); err != nil {
        return err              // 验证失败 → 422（自动）
    }
    todo, err := s.uc.Create(ctx.Context(), req.Title)
    if err != nil {
        return response.ErrInternal.WithCause(err) // 业务错误 → 500（自动）
    }
    return ctx.Success(todo)    // 成功 → 200 {code:0, data:...}（自动）
}
```
