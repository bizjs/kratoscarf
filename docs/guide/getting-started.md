# 快速开始

## 安装

```bash
go get github.com/bizjs/kratoscarf
```

## 最小示例

```go
package main

import (
    "github.com/bizjs/kratoscarf/router"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    httpSrv := kratoshttp.NewServer(kratoshttp.Address(":8080"))

    r := router.NewRouter(httpSrv)
    r.GET("/hello", func(ctx *router.Context) error {
        return ctx.JSON(200, map[string]string{"message": "hello"})
    })

    httpSrv.Start(nil)
}
```

## 推荐配置

实际项目中，建议启用验证 + 响应包装 + 错误编码 + 中间件：

```go
package main

import (
    "github.com/bizjs/kratoscarf/middleware"
    "github.com/bizjs/kratoscarf/response"
    "github.com/bizjs/kratoscarf/router"
    "github.com/bizjs/kratoscarf/validation"
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    httpSrv := kratoshttp.NewServer(
        kratoshttp.Address(":8080"),
        // 统一错误编码
        kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()),
        // HTTP 层 Filter（先于路由）
        kratoshttp.Filter(
            middleware.CORS(),
            middleware.Secure(middleware.SecureConfig{}),
        ),
        // Kratos 层 Middleware（路由后）
        kratoshttp.Middleware(
            middleware.RequestID(),
            recovery.Recovery(),
        ),
    )

    // 验证 + 响应包装 一键启用
    r := router.NewRouter(httpSrv,
        router.WithValidator(validation.New()),
        router.WithResponseWrapper(response.Wrap),
    )

    // 路由分组
    api := r.Group("/api/v1")
    api.GET("/users", listUsers)
    api.POST("/users", createUser)

    httpSrv.Start(nil)
}

func listUsers(ctx *router.Context) error {
    users := []map[string]any{
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
    }
    return ctx.Success(users)
    // → {"code": 0, "message": "ok", "data": [{"id": 1, ...}, ...]}
}

func createUser(ctx *router.Context) error {
    var req struct {
        Name  string `json:"name" validate:"required,min=2"`
        Email string `json:"email" validate:"required,email"`
    }
    if err := ctx.Bind(&req); err != nil {
        return err // 验证失败 → 422
    }
    // 业务逻辑...
    return ctx.Success(map[string]any{"id": 1, "name": req.Name})
}
```

## 三层自动拦截

上面的配置启用了类 NestJS 的请求处理管道：

```
请求进入
  ↓
ctx.Bind(&req)     → WithValidator 自动验证 → 失败返回 422
  ↓
业务逻辑           → return err → ErrorEncoder 自动格式化错误
  ↓
ctx.Success(data)  → WithResponseWrapper 自动包装 → {code, message, data}
```

Handler 只需关注业务逻辑，验证、包装、错误处理全部自动。

## 下一步

- [路由](/router) — Context API、分组、中间件
- [验证](/validation) — 自定义规则、错误格式
- [响应](/response) — BizError、分页
- [认证](/auth) — JWT + Session
