# kratoscarf

Kratos 微服务框架的 Web 开发增强层。不替代 Kratos，与 Kratos 原生 API 共存。

## 特性

- **Web 风格路由** — Gin/Echo 风格的 `GET`/`POST`/`Group`/`Use` API，基于 Kratos HTTP Server
- **请求验证** — 基于 go-playground/validator，`Bind()` 自动验证，支持自定义规则
- **统一响应** — `{code, message, data}` 信封格式，业务错误自动编码，分页开箱即用
- **JWT 认证** — Token 生成/验证/刷新/撤销，算法校验，中间件一键集成
- **Session 认证** — Cookie-based Session，内存/Redis Store，中间件自动加载保存
- **Web 中间件** — CORS（基于 rs/cors）、Secure Headers、RequestID
- **健康检查** — Liveness/Readiness 分离，并发检查，per-check 超时
- **工具包** — ID 生成（UUID/UUIDv7/ULID/Short）、加密（AES-GCM/Bcrypt/HMAC-SHA256）

## 安装

```bash
go get github.com/bizjs/kratoscarf
```

Contrib 模块按需安装：

```bash
go get github.com/bizjs/kratoscarf/contrib/auth/redis    # Redis Session Store
go get github.com/bizjs/kratoscarf/contrib/schedule       # 定时任务调度
```

## 快速开始

```go
package main

import (
    "github.com/bizjs/kratoscarf/middleware"
    "github.com/bizjs/kratoscarf/response"
    "github.com/bizjs/kratoscarf/router"
    "github.com/bizjs/kratoscarf/validation"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    httpSrv := kratoshttp.NewServer(
        kratoshttp.Address(":8080"),
        kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()),
        // HTTP 层 Filter
        kratoshttp.Filter(
            middleware.CORS(),
            middleware.Secure(middleware.SecureConfig{}),
        ),
        // Kratos 层 Middleware
        kratoshttp.Middleware(
            middleware.RequestID(),
        ),
    )

    // 创建路由（验证 + 响应包装 一键启用）
    r := router.NewRouter(httpSrv,
        router.WithValidator(validation.New()),
        router.WithResponseWrapper(response.Wrap),
    )

    r.GET("/hello", func(ctx *router.Context) error {
        return ctx.Success(map[string]string{"message": "hello"})
        // → {"code": 0, "message": "ok", "data": {"message": "hello"}}
    })

    r.POST("/users", func(ctx *router.Context) error {
        var req struct {
            Name  string `json:"name" validate:"required"`
            Email string `json:"email" validate:"required,email"`
        }
        if err := ctx.Bind(&req); err != nil {
            return err // 验证失败 → 422
        }
        // 业务逻辑...
        return ctx.Success(req)
    })

    // 启动
    httpSrv.Start(nil)
}
```

## 三层自动拦截

类似 NestJS 的 Pipe + Interceptor + ExceptionFilter，一次配置自动生效：

```go
r := router.NewRouter(srv,
    router.WithValidator(validation.New()),    // Bind() 自动验证 → 422
    router.WithResponseWrapper(response.Wrap), // Success() 自动包装 → {code, message, data}
)
// + ErrorEncoder → return err 自动格式化错误响应
```

| NestJS | kratoscarf | 效果 |
|--------|-----------|------|
| `ValidationPipe` | `WithValidator` | `ctx.Bind()` 自动验证 |
| `TransformInterceptor` | `WithResponseWrapper` | `ctx.Success()` 自动包装 |
| `HttpExceptionFilter` | `ErrorEncoder` | `return err` 自动格式化 |

Handler 只需关注业务逻辑：

```go
func createUser(ctx *router.Context) error {
    var req CreateRequest
    if err := ctx.Bind(&req); err != nil {
        return err                              // → 422 验证失败
    }
    user, err := svc.Create(ctx.Context(), req)
    if err != nil {
        return response.ErrInternal.WithCause(err) // → 500 业务错误
    }
    return ctx.Success(user)                    // → 200 {code:0, data:...}
}
```

## 包概览

### 核心包

| 包 | 说明 | 文档 |
|---|---|---|
| `router` | Web 风格路由 + Context | [docs/router.md](docs/router.md) |
| `response` | 统一响应、BizError、分页 | [docs/response.md](docs/response.md) |
| `validation` | 请求验证 | [docs/validation.md](docs/validation.md) |
| `auth/jwt` | JWT 认证 | [docs/auth.md](docs/auth.md) |
| `auth/session` | Session 认证 | [docs/auth.md](docs/auth.md) |
| `middleware` | CORS、Secure Headers、RequestID | [docs/middleware.md](docs/middleware.md) |
| `health` | 健康检查 | [docs/health.md](docs/health.md) |
| `util/id` | ID 生成 | [docs/util.md](docs/util.md) |
| `util/crypto` | 加密/哈希 | [docs/util.md](docs/util.md) |

### Contrib 包

| 包 | 说明 | 依赖 |
|---|---|---|
| `contrib/auth/redis` | Redis Session Store | go-redis/v9 |
| `contrib/schedule` | 定时任务（Cron + Interval） | robfig/cron/v3 |

## 设计原则

1. **增强而非替代** — 与 Kratos 原生 API 共存，不引入新的框架约束
2. **零包间耦合** — router/validation/response 互不 import，通过鸭子类型接口协作
3. **Contrib 隔离** — 重依赖（Redis/Casbin）独立 go.mod，核心零第三方依赖（除 Kratos 和 rs/cors）
4. **不为假设需求设计** — 只实现经过验证的、有实际业务价值的功能

## 开发

```bash
make build      # 编译
make test       # 运行测试
make test-all   # 运行所有模块测试（含 contrib）
make lint       # golangci-lint
make ci         # fmt + vet + lint + test-all
```

## 贡献

请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。

## License

MIT
