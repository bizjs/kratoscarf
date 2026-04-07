# middleware — Web 中间件

Kratos HTTP Server 的 Web 中间件集合。提供 Kratos 没有内置的 Web 开发常用功能。

## 设计原则

1. **不重复造轮子** — Recovery、Logging 等 Kratos 已内置的不再实现
2. **两种类型** — HTTP Filter（`kratoshttp.FilterFunc`，路由前执行）和 Kratos Middleware（`middleware.Middleware`，路由后执行）
3. **优先用成熟库** — CORS 基于 `rs/cors`，不自己实现复杂 spec

## 包结构

```
middleware/
├── cors.go        # CORS — HTTP Filter (基于 rs/cors)
├── secure.go      # 安全响应头 — HTTP Filter
└── requestid.go   # X-Request-Id — Kratos Middleware
```

## 快速集成

```go
import (
    "github.com/bizjs/kratoscarf/middleware"
    "github.com/go-kratos/kratos/v2/middleware/logging"
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

httpSrv := kratoshttp.NewServer(
    kratoshttp.Address(":8080"),
    // HTTP 层 Filter — 先于路由执行
    kratoshttp.Filter(
        middleware.CORS(),                           // CORS
        middleware.Secure(middleware.SecureConfig{}), // 安全头
    ),
    // Kratos 层 Middleware — 路由后执行
    kratoshttp.Middleware(
        middleware.RequestID(), // X-Request-Id
        recovery.Recovery(),   // Kratos 内置
        logging.Server(logger), // Kratos 内置
    ),
)
```

### Filter vs Middleware

| 类型 | 签名 | 执行时机 | 适用场景 |
|------|------|---------|---------|
| HTTP Filter | `func(http.Handler) http.Handler` | 路由匹配**之前** | CORS 预检、安全头 |
| Kratos Middleware | `func(Handler) Handler` | 路由匹配**之后** | RequestID、认证、日志 |

CORS 必须是 Filter，因为 OPTIONS 预检请求需要在路由匹配前拦截并返回。

---

## CORS

基于 [rs/cors](https://github.com/rs/cors) 的薄封装，提供开箱即用的默认配置。

### 一键启用

```go
kratoshttp.Filter(middleware.CORS())
```

默认配置：
- 允许所有来源（`*`）
- 允许方法：GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- 允许头：Origin, Content-Type, Accept, Authorization
- 预检缓存：24 小时

### 自定义配置

```go
kratoshttp.Filter(middleware.CORS(
    middleware.WithAllowOrigins("https://example.com", "https://app.example.com"),
    middleware.WithAllowCredentials(),
    middleware.WithAllowHeaders("Origin", "Content-Type", "Accept", "Authorization", "X-Custom-Header"),
    middleware.WithExposeHeaders("X-Total-Count"),
    middleware.WithMaxAge(3600),
))
```

### 配置项

| Option | 默认值 | 说明 |
|--------|--------|------|
| `WithAllowOrigins(origins...)` | `["*"]` | 允许的来源 |
| `WithAllowMethods(methods...)` | GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS | 允许的 HTTP 方法 |
| `WithAllowHeaders(headers...)` | Origin, Content-Type, Accept, Authorization | 允许的请求头 |
| `WithExposeHeaders(headers...)` | 无 | 浏览器可读取的响应头 |
| `WithAllowCredentials()` | false | 允许携带 Cookie/Auth |
| `WithMaxAge(seconds)` | 86400 | 预检响应缓存时间 |

**注意：** `WithAllowCredentials()` 启用时，`WithAllowOrigins` 不能包含 `"*"`，必须指定具体域名。

### 行为

```
请求无 Origin 头 → 跳过，直接放行（非跨域请求）
Origin 不在白名单 → 不设 CORS 头（浏览器拒绝）
OPTIONS 预检    → 返回 204 + CORS 头，不进入路由
正常跨域请求    → 设 CORS 头，继续处理
```

---

## Secure Headers

设置安全相关的 HTTP 响应头，相当于 Node.js 生态的 [Helmet.js](https://helmetjs.github.io/)。

### 一键启用

```go
kratoshttp.Filter(middleware.Secure(middleware.SecureConfig{}))
```

默认设置的响应头：

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
```

### 生产环境配置

```go
kratoshttp.Filter(middleware.Secure(middleware.SecureConfig{
    HSTSMaxAge:            63072000, // 2 年
    HSTSIncludeSubDomains: true,
    ContentSecurityPolicy: "default-src 'self'",
}))
```

输出：

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=63072000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

### 配置项

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `XContentTypeOptions` | `"nosniff"` | 防止浏览器 MIME 嗅探 |
| `XFrameOptions` | `"DENY"` | 防止页面被嵌入 iframe（点击劫持） |
| `ReferrerPolicy` | `"strict-origin-when-cross-origin"` | 控制 Referer 头泄露 |
| `ContentSecurityPolicy` | 不设置 | 内容安全策略（按需启用） |
| `HSTSMaxAge` | 0（不设置） | HSTS 有效期（秒），仅 HTTPS |
| `HSTSIncludeSubDomains` | false | HSTS 包含子域名 |

**注意：** HSTS 仅在 HTTPS 下设置。开发环境不要启用 `HSTSMaxAge`，否则浏览器会强制 HTTPS 访问。

---

## RequestID

生成并传播 `X-Request-Id`，用于请求追踪和日志关联。

### 使用

```go
kratoshttp.Middleware(middleware.RequestID())
```

### 行为

1. 读取请求头 `X-Request-Id`
2. 如果不存在，生成新的 UUID v4
3. 写入 context（可通过 `RequestIDFromContext` 读取）
4. 设置响应头 `X-Request-Id`

### 在 handler 中读取

```go
func myHandler(ctx *router.Context) error {
    requestID := middleware.RequestIDFromContext(ctx.Context())
    // 用于日志、传递给下游服务等
    logger.Info("processing", "request_id", requestID)
    return ctx.Success(data)
}
```

### 传递给下游服务

```go
func callDownstream(ctx context.Context) {
    requestID := middleware.RequestIDFromContext(ctx)
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://svc-b/api", nil)
    req.Header.Set("X-Request-Id", requestID)
    // ...
}
```

### 与 Kratos 日志集成

```go
// 自定义 logging middleware 提取 request ID
kratoshttp.Middleware(
    middleware.RequestID(),
    logging.Server(log.With(logger,
        "request_id", log.Valuer(func(ctx context.Context) interface{} {
            return middleware.RequestIDFromContext(ctx)
        }),
    )),
)
```

---

## 与 Kratos 内置中间件配合

kratoscarf middleware 和 Kratos 内置中间件互不冲突，推荐组合：

| 功能 | 来源 | 说明 |
|------|------|------|
| CORS | `kratoscarf/middleware` | Kratos 未内置 |
| Secure Headers | `kratoscarf/middleware` | Kratos 未内置 |
| Request ID | `kratoscarf/middleware` | Kratos 未内置 |
| Recovery | `kratos/v2/middleware/recovery` | 内置，捕获 panic |
| Logging | `kratos/v2/middleware/logging` | 内置，请求日志 |
| Tracing | `kratos/v2/middleware/tracing` | 内置，OpenTelemetry |
| Metadata | `kratos/v2/middleware/metadata` | 内置，元数据传播 |
