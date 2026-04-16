# kratoscarf — 项目上下文

> Web-oriented enhancement layer for Kratos. Module: `github.com/bizjs/kratoscarf`

## 项目定位

企业级 Kratos 微服务框架的 Web 开发增强层。不替代 Kratos，与 Kratos 原生 API 共存。用户同时 import `go-kratos/kratos` 和 `bizjs/kratoscarf`。

## 核心设计原则

1. **每个包定义自己的 Config** — 无中心化 config 包，各包自治
2. **零包间耦合** — router/validation/response 互不 import，通过鸭子类型接口协作
3. **contrib 模式** — 重依赖（casbin/gorm/redis）独立 go.mod 在 contrib/ 下
4. **无空壳 provider** — 如果 Wire ProviderSet 是空的或只有一行，直接删掉
5. **不为假设需求设计** — 不做过度抽象（如已删除的 crypto/idx 包的接口层）

## 当前目录结构

```
kratoscarf/
├── go.mod                      # 核心模块

# ===== 已实现并经过 review 的核心包 =====
├── router/                     # Web 风格路由（Gin/Echo API 风格）
│   ├── router.go               # Router, Group, Use, Handle, WithValidator, WithResponseWrapper
│   └── context.go              # Context: Param, Query, Bind(auto-validate), JSON, Success, Redirect, Cookie...
├── response/                   # 统一响应与错误协议
│   ├── response.go             # Response, Success, Wrap, Error
│   ├── errors.go               # BizError, 预定义错误(ErrNotFound等), IsBizError, FromKratosError
│   ├── encoder.go              # NewHTTPResponseEncoder(WithSuccessWrapper), NewHTTPErrorEncoder(WithErrorWrapper), ErrorToResponse
│   └── pagination.go           # PageRequest/Response[T], CursorRequest/Response[T]
├── validation/                 # 请求验证（go-playground/validator 封装）
│   ├── validation.go           # Validator, New, Struct, Var, Validate, WithRule, WithRuleFunc, WithTagName
│   ├── errors.go               # FieldError, ValidationErrors (实现 HTTPStatus/BizCode/ErrorData 鸭子接口)
│   └── handle.go               # Binder 接口, Handle[Ctx,T] 泛型包装, BindAndValidate

# ===== 已实现并经过 review 的认证包 =====
├── auth/
│   ├── jwt/                    # JWT 认证（全部自包含，不依赖 auth/ 父包）
│   │   ├── types.go            # Claims, TokenPair, TokenExtractor, TokenStore
│   │   ├── jwt.go              # Authenticator, New(Config), GenerateTokenPair/ValidateToken/RefreshToken
│   │   ├── middleware.go       # Middleware(*Authenticator, ...WithSkipPaths/WithExtractor)
│   │   ├── extractor.go        # BearerExtractor, CookieExtractor, QueryExtractor, ChainExtractor
│   │   ├── context.go          # ClaimsFromContext, ContextWithClaims
│   │   └── provider.go         # Wire ProviderSet
│   └── session/                # Session 认证（全部自包含）
│       ├── session.go          # Session, Store 接口, Manager, Config, NewManager
│       ├── middleware.go        # Middleware(*Manager, ...WithSkipPaths)
│       ├── memory.go           # MemoryStore (开发/测试用)
│       ├── context.go          # FromContext, ContextWithSession
│       └── provider.go

# ===== 已实现并测试的工具包 =====
├── util/
│   ├── id/                     # ID 生成: ULID, UUID, UUIDv7, Short, ShortN
│   └── crypto/                 # 密码学工具: AESGCMEncrypt/Decrypt, BcryptHash/Verify

# ===== 脚手架已搭建，待深度实现 =====
├── middleware/                  # Web 中间件（CORS, Secure Headers, RequestID）
├── health/                     # 健康检查（/healthz, /readyz, Liveness/Readiness 分离）

# ===== contrib — 独立 go.mod，按需引入 =====
├── contrib/
│   ├── auth/redis/             # Redis session store
│   └── schedule/               # 定时任务调度（Cron + Interval，robfig/cron 封装）

# ===== 示例应用 =====
├── examples/                   # 独立 go.mod (module: github.com/bizjs/kratoscarf/examples)
│   ├── app/auth-app/           # Kratos layout, proto 生成 handler, JWT+Session 集成
│   └── app/router-validation-demo/  # kratoscarf router + validation + session, TODO CRUD API

# ===== 文档 =====
├── docs/
│   ├── auth.md                 # auth/jwt + auth/session 设计与用法
│   ├── router.md               # router 设计与用法（含 NestJS 对比）
│   ├── validation.md           # validation 三种用法 + 自定义规则 + 鸭子类型协议
│   ├── response.md             # 统一响应 + BizError + 编码器 + 分页
│   ├── health.md               # 健康检查设计与用法
│   ├── util.md                 # util/id + util/crypto 工具函数
│   └── middleware.md           # CORS, Secure Headers, RequestID
└── spec.md                     # 原始技术设计文档（部分已过时，以代码为准）
```

## 关键架构决策

### 包间协作方式

```
router     → (零内部依赖)
  StructValidator 接口 → validation.Validator 实现
  ResponseWrapper 函数引用 → response.Wrap

validation → (零内部依赖)
  ValidationErrors 实现 HTTPStatus()/BizCode()/ErrorData() 鸭子接口
  BindError 实现 HTTPStatus()/BizCode()

response   → (零内部依赖)
  ErrorEncoder 通过鸭子类型识别任何实现 HTTPStatus()/BizCode()/ErrorData() 的 error

auth/jwt   → (不依赖 auth/ 父包)
auth/session → (不依赖 auth/ 父包)
```

### Router 三层自动拦截（类 NestJS）

```go
r := router.NewRouter(srv,
    router.WithValidator(validation.New()),    // ≈ ValidationPipe: Bind 自动验证
    router.WithResponseWrapper(response.Wrap), // ≈ TransformInterceptor: Success 自动包装
)
// + kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()) ≈ ExceptionFilter
```

### Kratos 集成注意事项

- `srv.Route("/").GET()` 注册的 handler **不走** Kratos server-level middleware（Kratos 原生行为）
- kratoscarf router 的 `Handle()` 通过 `ctx.Middleware()` 自动接入 server-level middleware，与 proto 生成的 handler 行为一致
- kratoscarf router 支持两层中间件：server-level（recovery/logging/tracing）+ route-level（`Use()`/`Group()`注册的 JWT/RBAC 等）
- Session middleware 的 auto-save 在 kratoscarf router 中不生效（transport context 分离），login 需要显式 `SaveSession`

### 错误处理规范

三种错误模式，各有适用场景：

```
1. BizError（response 包）— 面向 HTTP 客户端的业务错误
   用于：handler 返回的业务错误
   特点：携带 HTTPCode + BizCode + Message，ErrorEncoder 自动格式化
   示例：return response.ErrNotFound.WithMessage("user not found")

2. Sentinel Error + 鸭子接口 — 可用 errors.Is 判断的固定错误
   用于：包级别可预期的错误（如缺少 token、header 格式错误）
   特点：实现 HTTPStatus()/BizCode() 接口，与 ErrorEncoder 兼容
   示例：var ErrNoToken = &authError{msg: "...", httpCode: 401, bizCode: 40100}

3. fmt.Errorf + %w — 内部包装错误，不直接暴露给客户端
   用于：包内部的错误传播
   特点：ErrorEncoder 回退到 500 + "internal server error"
   示例：return fmt.Errorf("schedule: failed to add job %q: %w", name, err)
```

规则：
- 所有面向客户端的错误必须实现 `HTTPStatus() int` 或使用 `BizError`
- 包内部错误用 `fmt.Errorf("包名: 上下文: %w", err)` 包装
- 永远不要在客户端响应中暴露 `err.Error()` 的原始内容（可能泄露内部信息）

### 命名规则

- util 子包函数名：`算法 + 动作` (如 `BcryptHash`, `AESGCMEncrypt`, `NewULID`)
- 不占用泛化名称，为未来扩展留空间
- 包名参与语义：`hash.BcryptHash` 不是 `hash.Hash`

## 待完成工作

### Phase 2 — 需要深度实现
- middleware/ — 已实现 CORS(rs/cors), Secure Headers, RequestID

### 需要清理
- 各包中空的 `provider.go` — 如果 ProviderSet 为空应删除
- `spec.md` — 部分内容已过时（config 包已删除、pagination 已合并到 response 等）

### 文档
- docs/ 中的 auth.md 需要更新（反映 auth/ 父包已清空的变化）
- 各 contrib 包暂无文档
