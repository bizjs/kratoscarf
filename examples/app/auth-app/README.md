# auth-app

演示 `auth/jwt` 和 `auth/session` 如何集成到标准 Kratos 应用中。

## 运行

```bash
cd examples
go run ./app/auth-app/cmd/auth-app/ -conf ./app/auth-app/configs
```

## 预置用户

| 用户名 | 密码 |
|--------|------|
| alice  | 123456 |
| bob    | 654321 |

## API 端点

### JWT 认证 (无状态，Bearer Token)

```bash
# 登录 — 返回 accessToken + refreshToken
curl -s -X POST http://localhost:8080/jwt/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"123456"}'

# 刷新 Token
curl -s -X POST http://localhost:8080/jwt/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refreshToken":"<REFRESH_TOKEN>"}'

# 获取个人信息 (需要 Bearer Token)
curl -s http://localhost:8080/jwt/profile \
  -H 'Authorization: Bearer <ACCESS_TOKEN>'
```

### Session 认证 (有状态，Cookie)

```bash
# 登录 — 服务端创建 Session，返回 Set-Cookie
curl -s -X POST http://localhost:8080/session/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"bob","password":"654321"}' -c cookies.txt

# 获取个人信息 (需要 Session Cookie)
curl -s http://localhost:8080/session/profile -b cookies.txt

# 登出 — 销毁 Session
curl -s -X POST http://localhost:8080/session/logout \
  -H 'Content-Type: application/json' -d '{}' \
  -b cookies.txt -c cookies.txt
```

## 集成要点

### 1. Proto 定义 API

在 `api/auth/v1/auth.proto` 中定义 RPC + HTTP binding，protoc 生成 handler wrapper：

```proto
service AuthService {
  rpc JWTLogin(LoginRequest) returns (TokenPair) {
    option (google.api.http) = { post: "/jwt/login" body: "*" };
  }
  rpc JWTProfile(EmptyRequest) returns (ProfileReply) {
    option (google.api.http) = { get: "/jwt/profile" };
  }
}
```

### 2. 注册 Middleware + 路由

```go
srv := http.NewServer(
    http.Middleware(
        authjwt.Middleware(jwtAuth,
            authjwt.WithSkipPaths("/jwt/login", "/jwt/refresh", "/session/login"),
        ),
        session.Middleware(sessMgr,
            session.WithSkipPaths("/jwt/login", "/jwt/refresh", "/jwt/profile"),
        ),
    ),
)
// proto 生成的代码内部调用 ctx.Middleware()，自动触发上面注册的 middleware chain
v1.RegisterAuthServiceHTTPServer(srv, authSvc)
```

### 3. Service 实现

Service 方法收到的 `context.Context` 已经被 middleware 增强，直接读取即可：

```go
func (s *AuthService) JWTProfile(ctx context.Context, _ *v1.EmptyRequest) (*v1.ProfileReply, error) {
    // JWT middleware 已验证 token 并注入 Claims
    claims := authjwt.ClaimsFromContext(ctx)
    // ...
}

func (s *AuthService) SessionProfile(ctx context.Context, _ *v1.EmptyRequest) (*v1.ProfileReply, error) {
    // Session middleware 已从 cookie 加载 Session
    sess := session.FromContext(ctx)
    // ...
}
```

## 配置说明

```yaml
auth:
  jwt:
    secret: "your-secret-key"      # JWT 签名密钥
    issuer: "your-app"             # JWT 签发者
    accessExpiry: "15m"            # Access Token 有效期
    refreshExpiry: "168h"          # Refresh Token 有效期 (7天)
  session:
    maxAge: "24h"                  # Session 有效期
    cookieName: "auth_app_sid"     # Cookie 名称
```
