# auth -- 认证模块

## 概述

`auth` 包为 kratoscarf 项目提供完整的认证能力,采用**接口与实现分离**的分层架构:

| 层级 | 包路径 | 职责 |
|------|--------|------|
| 核心接口层 | `auth/` | 定义 `Authenticator`、`TokenExtractor`、`TokenStore` 接口,提供中间件框架和 context 辅助函数 |
| JWT 实现 | `auth/jwt/` | 基于 `golang-jwt/jwt/v5` 的 JWT 认证实现 |
| Session 实现 | `auth/session/` | 服务端 Session 管理,内置内存 Store |
| Redis Session Store | `contrib/auth/redis/` | 基于 Redis 的生产级 Session 持久化 |

设计理念:业务代码只依赖 `auth/` 包的接口,具体实现按需引入。通过 Wire 的 `ProviderSet` 完成依赖注入,切换实现只需替换 Provider。

---

## 包结构

```
auth/
  auth.go          # 核心类型: Claims, TokenPair, Authenticator/TokenExtractor/TokenStore 接口
  context.go       # context 辅助: ClaimsFromContext, UserIDFromContext 等
  extractor.go     # Token 提取器: Bearer, Cookie, Query, Chain
  middleware.go     # Kratos 认证中间件 (JWT)
  provider.go      # Wire ProviderSet (空集, 作为占位)
  jwt/
    jwt.go         # JWT Authenticator 实现
    provider.go    # Wire ProviderSet (绑定 Authenticator 接口)
  session/
    session.go     # Session 结构体, Store 接口, Manager
    middleware.go   # Kratos Session 中间件 (自动加载/保存)
    context.go     # session context 辅助: FromContext
    memory.go      # 内存 Store 实现 (开发/测试用)
    provider.go    # Wire ProviderSet
contrib/auth/redis/
    session.go     # Redis SessionStore 实现
    provider.go    # Wire ProviderSet
```

---

## JWT 认证

### 配置

`config.JWTConfig` 定义了 JWT 相关的所有配置字段:

```go
type JWTConfig struct {
    Secret        string        `yaml:"secret"`        // 签名密钥 (必填)
    Issuer        string        `yaml:"issuer"`        // 签发者标识
    AccessExpiry  time.Duration `yaml:"accessExpiry"`  // Access Token 有效期, 默认 2h
    RefreshExpiry time.Duration `yaml:"refreshExpiry"` // Refresh Token 有效期, 默认 168h (7天)
    SigningMethod string        `yaml:"signingMethod"` // 签名算法, 默认 "HS256"
}
```

YAML 配置示例:

```yaml
auth:
  jwt:
    secret: "your-256-bit-secret-key-here"
    issuer: "myapp"
    accessExpiry: "2h"
    refreshExpiry: "168h"
    signingMethod: "HS256"
```

### 基本用法

#### 创建 Authenticator

```go
import (
    "github.com/bizjs/kratoscarf/auth/jwt"
    "github.com/bizjs/kratoscarf/config"
)

cfg := config.JWTConfig{
    Secret:        "your-secret-key",
    Issuer:        "myapp",
    AccessExpiry:  2 * time.Hour,
    RefreshExpiry: 168 * time.Hour,
}

authenticator := jwt.New(cfg)
```

可通过 `Option` 进一步配置:

```go
authenticator := jwt.New(cfg,
    jwt.WithSigningMethod(jwtlib.SigningMethodHS384),
    jwt.WithTokenStore(myTokenStore),  // 启用 Token 吊销
    jwt.WithClaimsFactory(func() auth.Claims { return auth.Claims{} }),
)
```

`jwt.Option` 完整列表:

| Option | 说明 |
|--------|------|
| `WithTokenStore(store auth.TokenStore)` | 设置 Token 存储, 启用吊销能力 |
| `WithSigningMethod(method jwt.SigningMethod)` | 覆盖签名算法 (默认 HS256) |
| `WithClaimsFactory(fn func() auth.Claims)` | 自定义 Claims 工厂函数 |

#### 生成 Token 对

```go
claims := auth.Claims{
    UserID:   "user-123",
    Username: "zhangsan",
    Roles:    []string{"admin", "editor"},
    Extra:    map[string]string{"department": "engineering"},
}

pair, err := authenticator.GenerateTokenPair(ctx, claims)
if err != nil {
    return err
}

// pair.AccessToken  -- 访问令牌
// pair.RefreshToken -- 刷新令牌
// pair.ExpiresAt    -- Access Token 过期时间
```

#### 验证 Token

```go
claims, err := authenticator.ValidateToken(ctx, tokenString)
if err != nil {
    // token 无效、过期、或已被吊销
    return err
}

fmt.Println(claims.UserID)   // "user-123"
fmt.Println(claims.Username) // "zhangsan"
fmt.Println(claims.Roles)    // ["admin", "editor"]
```

当配置了 `TokenStore` 时,`ValidateToken` 会额外检查 Token 是否已被吊销。

#### 刷新 Token

```go
newPair, err := authenticator.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
    return err
}
// 使用 newPair.AccessToken 和 newPair.RefreshToken
```

#### 吊销 Token

吊销需要先配置 `TokenStore`:

```go
err := authenticator.RevokeToken(ctx, tokenString)
if err != nil {
    return err
}
// Token 已加入黑名单, 后续 ValidateToken 将拒绝此 Token
```

如果未配置 `TokenStore`,`RevokeToken` 静默返回 `nil`。

### 中间件

`auth.Middleware` 返回一个 Kratos 中间件,自动从请求中提取 Token、验证并将 Claims 注入 context:

```go
import (
    "github.com/bizjs/kratoscarf/auth"
    "github.com/bizjs/kratoscarf/auth/jwt"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

authenticator := jwt.New(cfg)

httpSrv := kratoshttp.NewServer(
    kratoshttp.Middleware(
        auth.Middleware(authenticator),
    ),
)
```

#### 配置选项

```go
auth.Middleware(authenticator,
    // 自定义 Token 提取方式 (默认 BearerExtractor)
    auth.WithExtractor(auth.CookieExtractor("access_token")),

    // 跳过特定路径的认证
    auth.WithSkipPaths("/login", "/register", "/health"),

    // 自定义错误处理
    auth.WithErrorHandler(func(ctx context.Context, err error) error {
        return myCustomUnauthorizedError(err)
    }),
)
```

`MiddlewareOption` 完整列表:

| Option | 说明 |
|--------|------|
| `WithExtractor(e TokenExtractor)` | 覆盖默认的 BearerExtractor |
| `WithSkipPaths(paths ...string)` | 指定免认证路径 |
| `WithErrorHandler(fn func(ctx, err) error)` | 自定义认证失败时的错误返回 |

### Token 提取器

`auth` 包内置 4 种 `TokenExtractor` 实现:

#### BearerExtractor (默认)

从 `Authorization: Bearer <token>` 头提取:

```go
extractor := auth.BearerExtractor()
```

#### CookieExtractor

从指定名称的 Cookie 中提取:

```go
extractor := auth.CookieExtractor("access_token")
```

#### QueryExtractor

从 URL 查询参数中提取:

```go
extractor := auth.QueryExtractor("token")
// 匹配: GET /api/data?token=xxx
```

#### ChainExtractor

按顺序尝试多个提取器,返回第一个成功提取的结果:

```go
extractor := auth.ChainExtractor(
    auth.BearerExtractor(),           // 优先从 Header 提取
    auth.CookieExtractor("token"),    // 其次从 Cookie 提取
    auth.QueryExtractor("token"),     // 最后从 URL 参数提取
)
```

所有提取器在未找到 Token 时返回 `auth.ErrNoToken`,Authorization 头格式错误时返回 `auth.ErrInvalidAuthHeader`。

### Context 辅助函数

在通过中间件认证后的 handler 中,可使用以下函数获取用户信息:

```go
// 获取 Claims, 未认证时返回 nil
claims := auth.ClaimsFromContext(ctx)
if claims != nil {
    fmt.Println(claims.UserID, claims.Roles)
}

// 获取 Claims, 未认证时 panic (适合已确认认证通过的场景)
claims := auth.MustClaimsFromContext(ctx)

// 快捷获取 UserID, 未认证时返回空字符串
userID := auth.UserIDFromContext(ctx)
```

### Wire 集成

```go
import "github.com/bizjs/kratoscarf/auth/jwt"

// 在 wire.go 中:
var providerSet = wire.NewSet(
    jwt.ProviderSet,  // 提供 *jwt.Authenticator, 并绑定到 auth.Authenticator 接口
    // ...
)
```

`jwt.ProviderSet` 包含:
- `jwt.New` -- 构造函数 (需要 `config.JWTConfig` 作为输入)
- `wire.Bind(new(auth.Authenticator), new(*jwt.Authenticator))` -- 接口绑定

---

## Session 认证

### 配置

`config.SessionConfig` 定义了 Session 相关的所有配置字段:

```go
type SessionConfig struct {
    MaxAge     time.Duration `yaml:"maxAge"`     // Session 最大存活时间, 默认 24h
    CookieName string        `yaml:"cookieName"` // Cookie 名称, 默认 "session_id"
    CookiePath string        `yaml:"cookiePath"` // Cookie 路径, 默认 "/"
    Domain     string        `yaml:"domain"`     // Cookie 域名
    Secure     bool          `yaml:"secure"`     // 是否仅 HTTPS, 默认 false
    HTTPOnly   bool          `yaml:"httpOnly"`   // 是否 HTTPOnly, 默认 true
    SameSite   string        `yaml:"sameSite"`   // "lax" (默认) | "strict" | "none"
}
```

YAML 配置示例:

```yaml
auth:
  session:
    maxAge: "24h"
    cookieName: "session_id"
    cookiePath: "/"
    domain: "example.com"
    secure: true
    httpOnly: true
    sameSite: "lax"
```

### 基本用法

#### Session 结构体

```go
type Session struct {
    ID        string         // Session ID (自动生成)
    Values    map[string]any // 存储的键值对
    CreatedAt time.Time      // 创建时间
    ExpiresAt time.Time      // 过期时间
    IsNew     bool           // 是否为新创建的 Session
    Modified  bool           // 是否被修改过
}
```

#### 创建 Manager

```go
import (
    "github.com/bizjs/kratoscarf/auth/session"
    "github.com/bizjs/kratoscarf/config"
)

store := session.NewMemoryStore() // 开发/测试环境

cfg := config.SessionConfig{
    MaxAge:     24 * time.Hour,
    CookieName: "session_id",
    Secure:     true,
    HTTPOnly:   true,
    SameSite:   "lax",
}

manager := session.NewManager(store, cfg)
```

可通过 `Option` 进一步配置:

```go
manager := session.NewManager(store, cfg,
    session.WithMaxAge(48*time.Hour),
    session.WithCookieName("my_session"),
    session.WithCookiePath("/app"),
    session.WithCookieDomain("example.com"),
    session.WithCookieSecure(true),
    session.WithCookieHTTPOnly(true),
    session.WithCookieSameSite(http.SameSiteStrictMode),
    session.WithIDGenerator(func() string { return myCustomID() }),
)
```

`session.Option` 完整列表:

| Option | 说明 |
|--------|------|
| `WithMaxAge(d time.Duration)` | 覆盖 Session 最大存活时间 |
| `WithCookieName(name string)` | 覆盖 Cookie 名称 |
| `WithCookiePath(path string)` | 覆盖 Cookie 路径 |
| `WithCookieDomain(domain string)` | 覆盖 Cookie 域名 |
| `WithCookieSecure(secure bool)` | 覆盖 Secure 标志 |
| `WithCookieHTTPOnly(httpOnly bool)` | 覆盖 HTTPOnly 标志 |
| `WithCookieSameSite(s http.SameSite)` | 覆盖 SameSite 策略 |
| `WithIDGenerator(fn func() string)` | 自定义 Session ID 生成函数 |

#### 获取 Session

```go
sess, err := manager.GetSession(ctx, r)
if err != nil {
    return err
}
// 如果请求中有有效的 session cookie, 返回已有 Session
// 否则创建一个新 Session (sess.IsNew == true)
```

#### 读写 Session 值

```go
// 写入值 (自动标记 Modified = true)
sess.Set("userID", "user-123")
sess.Set("role", "admin")

// 读取值
userID, ok := sess.Get("userID")
if ok {
    fmt.Println(userID) // "user-123"
}

// 删除值 (自动标记 Modified = true)
sess.Delete("role")
```

#### 保存 Session

```go
err := manager.SaveSession(ctx, w, sess)
// 将 Session 持久化到 Store, 并向客户端设置 Cookie
```

#### 销毁 Session

```go
err := manager.DestroySession(ctx, w, r)
// 从 Store 中删除 Session, 并清除客户端 Cookie (MaxAge=-1)
```

### 中间件

`session.Middleware` 返回一个 Kratos 中间件,自动加载 Session 到 context,并在 handler 完成后按需保存:

```go
import (
    "github.com/bizjs/kratoscarf/auth/session"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

manager := session.NewManager(store, cfg)

httpSrv := kratoshttp.NewServer(
    kratoshttp.Middleware(
        session.Middleware(manager),
    ),
)
```

**自动保存行为**: 当 handler 执行完毕后,如果 `sess.Modified == true` 或 `sess.IsNew == true`,中间件会自动调用 `manager.SaveSession` 保存并设置 Cookie。不需要手动保存。

#### 配置选项

```go
session.Middleware(manager,
    session.WithSkipPaths("/health", "/metrics"),
)
```

### Context 辅助函数

```go
import "github.com/bizjs/kratoscarf/auth/session"

// 从 context 中获取 Session, 无 Session 时返回 nil
sess := session.FromContext(ctx)
if sess != nil {
    userID, _ := sess.Get("userID")
    fmt.Println(userID)
}
```

### 内存 Store (开发/测试)

`MemoryStore` 使用 `sync.RWMutex` 保护的 `map` 存储 Session,适用于开发和测试:

```go
store := session.NewMemoryStore()
```

特性:
- 自动后台清理过期 Session (每 5 分钟扫描一次)
- 进程重启后 Session 丢失
- 不支持多实例部署

### Redis Store (生产)

`contrib/auth/redis` 包提供基于 Redis 的 Session Store,适用于生产环境:

```go
import (
    goredis "github.com/redis/go-redis/v9"
    sessionredis "github.com/bizjs/kratoscarf/contrib/auth/redis"
)

client := goredis.NewClient(&goredis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})

store := sessionredis.NewSessionStore(client)
```

通过 `Option` 自定义 key 前缀:

```go
store := sessionredis.NewSessionStore(client,
    sessionredis.WithKeyPrefix("myapp:sess:"),  // 默认 "session:"
)
```

Redis Store 特性:
- Session 数据以 JSON 序列化存储
- 利用 Redis TTL 自动过期
- `Get` 时额外检查 `ExpiresAt`,过期则自动删除并返回 nil
- 支持多实例部署

### Wire 集成

使用内存 Store (开发/测试):

```go
import "github.com/bizjs/kratoscarf/auth/session"

var providerSet = wire.NewSet(
    session.ProviderSet,  // 提供 *Manager + *MemoryStore, 绑定 Store 接口
    // ...
)
```

`session.ProviderSet` 包含:
- `session.NewManager` -- 构造函数 (需要 `session.Store` 和 `config.SessionConfig`)
- `session.NewMemoryStore` -- 内存 Store 构造函数
- `wire.Bind(new(session.Store), new(*session.MemoryStore))` -- 接口绑定

使用 Redis Store (生产):

```go
import (
    "github.com/bizjs/kratoscarf/auth/session"
    sessionredis "github.com/bizjs/kratoscarf/contrib/auth/redis"
)

var providerSet = wire.NewSet(
    session.NewManager,         // Manager 构造函数
    sessionredis.ProviderSet,   // Redis Store + 接口绑定
    // ...
)
```

`sessionredis.ProviderSet` 包含:
- `sessionredis.NewSessionStore` -- 构造函数 (需要 `*goredis.Client`)
- `wire.Bind(new(session.Store), new(*sessionredis.SessionStore))` -- 接口绑定

---

## 完整示例

### JWT API 服务

一个完整的 JWT 认证 API 服务,包含登录、刷新、受保护路由:

```go
package main

import (
    "context"
    "time"

    "github.com/bizjs/kratoscarf/auth"
    "github.com/bizjs/kratoscarf/auth/jwt"
    "github.com/bizjs/kratoscarf/config"
    "github.com/go-kratos/kratos/v2"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    cfg := config.JWTConfig{
        Secret:        "my-secret-key-at-least-32-bytes!",
        Issuer:        "myapp",
        AccessExpiry:  2 * time.Hour,
        RefreshExpiry: 168 * time.Hour,
    }

    authenticator := jwt.New(cfg)

    httpSrv := kratoshttp.NewServer(
        kratoshttp.Address(":8080"),
        kratoshttp.Middleware(
            auth.Middleware(authenticator,
                auth.WithSkipPaths("/login", "/refresh"),
            ),
        ),
    )

    route := httpSrv.Route("/")

    // 登录 -- 返回 Token 对
    route.POST("/login", func(ctx kratoshttp.Context) error {
        // 省略: 验证用户名密码
        userID := "user-123"
        username := "zhangsan"

        pair, err := authenticator.GenerateTokenPair(ctx, auth.Claims{
            UserID:   userID,
            Username: username,
            Roles:    []string{"admin"},
        })
        if err != nil {
            return err
        }
        return ctx.JSON(200, pair)
    })

    // 刷新 Token
    route.POST("/refresh", func(ctx kratoshttp.Context) error {
        var req struct {
            RefreshToken string `json:"refreshToken"`
        }
        if err := ctx.Bind(&req); err != nil {
            return err
        }

        pair, err := authenticator.RefreshToken(ctx, req.RefreshToken)
        if err != nil {
            return err
        }
        return ctx.JSON(200, pair)
    })

    // 受保护路由 -- 需要有效 Token
    route.GET("/profile", func(ctx kratoshttp.Context) error {
        claims := auth.MustClaimsFromContext(ctx)
        return ctx.JSON(200, map[string]any{
            "userID":   claims.UserID,
            "username": claims.Username,
            "roles":    claims.Roles,
        })
    })

    app := kratos.New(kratos.Server(httpSrv))
    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

### Session Web 应用

一个完整的 Session 认证 Web 应用,包含登录、注销、用户信息:

```go
package main

import (
    "github.com/bizjs/kratoscarf/auth/session"
    "github.com/bizjs/kratoscarf/config"
    "github.com/go-kratos/kratos/v2"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    store := session.NewMemoryStore()
    cfg := config.SessionConfig{
        MaxAge:     24 * time.Hour,
        CookieName: "session_id",
        Secure:     false, // 开发环境
        HTTPOnly:   true,
        SameSite:   "lax",
    }
    manager := session.NewManager(store, cfg)

    httpSrv := kratoshttp.NewServer(
        kratoshttp.Address(":8080"),
        kratoshttp.Middleware(
            session.Middleware(manager,
                session.WithSkipPaths("/health"),
            ),
        ),
    )

    route := httpSrv.Route("/")

    // 登录 -- 将用户信息写入 Session
    route.POST("/login", func(ctx kratoshttp.Context) error {
        // 省略: 验证用户名密码
        sess := session.FromContext(ctx)
        sess.Set("userID", "user-123")
        sess.Set("username", "zhangsan")
        sess.Set("role", "admin")
        // 中间件检测到 Modified=true, 会自动保存并设置 Cookie
        return ctx.JSON(200, map[string]string{"message": "ok"})
    })

    // 注销 -- 销毁 Session
    route.POST("/logout", func(ctx kratoshttp.Context) error {
        httpReq := kratoshttp.RequestFromServerContext(ctx)
        tr, _ := transport.FromServerContext(ctx)
        ht := tr.(*kratoshttp.Transport)
        if err := manager.DestroySession(ctx, ht.Response(), ht.Request()); err != nil {
            return err
        }
        return ctx.JSON(200, map[string]string{"message": "logged out"})
    })

    // 查看用户信息
    route.GET("/profile", func(ctx kratoshttp.Context) error {
        sess := session.FromContext(ctx)
        if sess == nil {
            return ctx.JSON(401, map[string]string{"error": "not logged in"})
        }
        userID, _ := sess.Get("userID")
        username, _ := sess.Get("username")
        role, _ := sess.Get("role")
        return ctx.JSON(200, map[string]any{
            "userID":   userID,
            "username": username,
            "role":     role,
        })
    })

    app := kratos.New(kratos.Server(httpSrv))
    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

### 混合模式

同一服务中同时使用 JWT (API) 和 Session (Web 管理后台):

```go
package main

import (
    "time"

    "github.com/bizjs/kratoscarf/auth"
    "github.com/bizjs/kratoscarf/auth/jwt"
    "github.com/bizjs/kratoscarf/auth/session"
    "github.com/bizjs/kratoscarf/config"
    goredis "github.com/redis/go-redis/v9"
    sessionredis "github.com/bizjs/kratoscarf/contrib/auth/redis"
    "github.com/go-kratos/kratos/v2"
    kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    // --- JWT 配置 (API) ---
    jwtCfg := config.JWTConfig{
        Secret:        "my-jwt-secret-key-at-least-32-b",
        Issuer:        "myapp",
        AccessExpiry:  2 * time.Hour,
        RefreshExpiry: 168 * time.Hour,
    }
    jwtAuth := jwt.New(jwtCfg)

    // --- Session 配置 (Web 后台) ---
    redisClient := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
    sessStore := sessionredis.NewSessionStore(redisClient,
        sessionredis.WithKeyPrefix("myapp:sess:"),
    )
    sessCfg := config.SessionConfig{
        MaxAge:     8 * time.Hour,
        CookieName: "admin_session",
        Secure:     true,
        HTTPOnly:   true,
        SameSite:   "strict",
    }
    sessManager := session.NewManager(sessStore, sessCfg)

    // --- API 路由 (JWT 认证) ---
    apiSrv := kratoshttp.NewServer(
        kratoshttp.Address(":8080"),
        kratoshttp.Middleware(
            auth.Middleware(jwtAuth,
                auth.WithSkipPaths("/api/login"),
            ),
        ),
    )
    apiRoute := apiSrv.Route("/api")
    apiRoute.POST("/login", func(ctx kratoshttp.Context) error {
        pair, err := jwtAuth.GenerateTokenPair(ctx, auth.Claims{
            UserID: "user-123",
            Roles:  []string{"user"},
        })
        if err != nil {
            return err
        }
        return ctx.JSON(200, pair)
    })
    apiRoute.GET("/data", func(ctx kratoshttp.Context) error {
        claims := auth.MustClaimsFromContext(ctx)
        return ctx.JSON(200, map[string]string{"userID": claims.UserID})
    })

    // --- 管理后台路由 (Session 认证) ---
    adminSrv := kratoshttp.NewServer(
        kratoshttp.Address(":8081"),
        kratoshttp.Middleware(
            session.Middleware(sessManager,
                session.WithSkipPaths("/admin/login"),
            ),
        ),
    )
    adminRoute := adminSrv.Route("/admin")
    adminRoute.POST("/login", func(ctx kratoshttp.Context) error {
        sess := session.FromContext(ctx)
        sess.Set("adminID", "admin-001")
        sess.Set("role", "superadmin")
        return ctx.JSON(200, map[string]string{"message": "ok"})
    })
    adminRoute.GET("/dashboard", func(ctx kratoshttp.Context) error {
        sess := session.FromContext(ctx)
        adminID, _ := sess.Get("adminID")
        return ctx.JSON(200, map[string]any{"adminID": adminID})
    })

    app := kratos.New(kratos.Server(apiSrv, adminSrv))
    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

---

## 最佳实践

### Secret Key 管理

- **永远不要** 将 JWT Secret 硬编码在代码中或提交到版本控制
- 使用环境变量或密钥管理服务 (如 Vault, AWS Secrets Manager) 注入
- HS256 密钥长度至少 32 字节; 生产环境建议使用 RS256 或 ES256 (非对称签名)
- 定期轮换密钥, 轮换期间可通过 `ChainExtractor` 支持新旧密钥并存

```yaml
# 通过环境变量注入
auth:
  jwt:
    secret: "${JWT_SECRET}"
```

### Token 有效期建议

| Token 类型 | 推荐有效期 | 说明 |
|------------|-----------|------|
| Access Token | 15min - 2h | 越短越安全, API 密集调用场景可适当延长 |
| Refresh Token | 7d - 30d | 配合吊销机制使用 |
| Session | 8h - 24h | Web 应用场景, 配合滑动窗口 |

### Session 安全

生产环境 Session Cookie 应启用全部安全标志:

```yaml
auth:
  session:
    secure: true       # 仅 HTTPS 传输
    httpOnly: true     # 禁止 JavaScript 访问 (防 XSS)
    sameSite: "lax"    # 防 CSRF, 严格场景使用 "strict"
    domain: ".example.com"
```

- `Secure: true` -- 确保 Cookie 仅通过 HTTPS 发送
- `HTTPOnly: true` -- 防止 XSS 攻击窃取 Session ID
- `SameSite: "lax"` -- 对大多数场景提供 CSRF 保护, 跨站 POST 时使用 `"strict"`
- `SameSite: "none"` -- 仅在需要跨站请求时使用, 必须同时启用 `Secure: true`

### Token 吊销策略

JWT 本身是无状态的, 吊销需要配合 `TokenStore` 实现黑名单机制:

```go
// 方案一: 直接使用 Redis 作为 TokenStore (推荐)
// 实现 auth.TokenStore 接口即可:
type TokenStore interface {
    Store(ctx context.Context, token string, expiration time.Duration) error
    Exists(ctx context.Context, token string) (bool, error)
    Delete(ctx context.Context, token string) error
}

// 在创建 JWT Authenticator 时注入:
authenticator := jwt.New(cfg, jwt.WithTokenStore(redisTokenStore))
```

吊销策略选择:

| 策略 | 适用场景 | 说明 |
|------|---------|------|
| 短有效期 + 不吊销 | 低安全要求 API | 最简方案, Access Token 15min 过期即可 |
| Token 黑名单 | 需要即时吊销 | 每次 `ValidateToken` 检查黑名单, 有轻微性能开销 |
| Refresh Token 轮换 | 通用推荐 | 刷新时废弃旧 Refresh Token, 检测到旧 Token 使用则吊销全部 |
