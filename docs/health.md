# health — 健康检查

Kubernetes 风格的健康检查，区分 Liveness（进程是否存活）和 Readiness（是否可接收流量）。

## 设计原则

1. **Liveness / Readiness 分离** — Checker 注册时声明类型，两端点独立检查
2. **并发执行** — 所有匹配的 Checker 并发运行，单个 Checker 不阻塞其他
3. **per-check 超时** — 每个 Checker 独立 context timeout，默认 5s
4. **零包依赖** — Handler 输出标准 `net/http.Handler`，不依赖 router 包

## 包结构

```
health/
├── health.go    # Registry, Checker, CheckType, CheckResult, HealthReport
└── handler.go   # NewLivenessHandler, NewReadinessHandler (net/http.Handler)
```

## 基本用法

```go
import "github.com/bizjs/kratoscarf/health"

// 创建 Registry
registry := health.NewRegistry()

// 注册 Readiness checker — DB、Redis 等外部依赖
registry.RegisterFunc("postgres", health.Readiness, func(ctx context.Context) error {
    return db.PingContext(ctx)
})

registry.RegisterFunc("redis", health.Readiness, func(ctx context.Context) error {
    return rdb.Ping(ctx).Err()
})

// 注册 Liveness checker — 进程内部状态
registry.RegisterFunc("goroutine", health.Liveness, func(ctx context.Context) error {
    if runtime.NumGoroutine() > 10000 {
        return fmt.Errorf("goroutine leak: %d", runtime.NumGoroutine())
    }
    return nil
})

// 同时参与两个端点
registry.RegisterFunc("disk", health.Readiness|health.Liveness, func(ctx context.Context) error {
    // 检查磁盘空间
    return checkDiskSpace()
})
```

## 注册 HTTP Handler

Handler 返回标准 `net/http.Handler`，可以接入任何 HTTP 框架。

### 接入 Kratos HTTP Server

```go
import kratoshttp "github.com/go-kratos/kratos/v2/transport/http"

httpSrv := kratoshttp.NewServer(kratoshttp.Address(":8080"))

httpSrv.HandlePrefix("/healthz", health.NewLivenessHandler(registry))
httpSrv.HandlePrefix("/readyz", health.NewReadinessHandler(registry))
```

### 接入 kratoscarf router

```go
r := router.NewRouter(httpSrv)

r.GET("/healthz", func(ctx *router.Context) error {
    health.NewLivenessHandler(registry).ServeHTTP(ctx.Response(), ctx.Request())
    return nil
})
```

### 接入标准 net/http

```go
mux := http.NewServeMux()
mux.Handle("/healthz", health.NewLivenessHandler(registry))
mux.Handle("/readyz", health.NewReadinessHandler(registry))
```

## 响应格式

### 全部健康 → HTTP 200

```json
{
  "status": "up",
  "checks": {
    "postgres": { "status": "up" },
    "redis": { "status": "up" }
  }
}
```

### 任一不健康 → HTTP 503

```json
{
  "status": "down",
  "checks": {
    "postgres": { "status": "up" },
    "redis": { "status": "down", "message": "connection refused" }
  }
}
```

### 无 Checker 注册 → HTTP 200

```json
{
  "status": "up",
  "checks": {}
}
```

Liveness 端点通常不注册任何 checker（进程在就是活的），返回 `{"status": "up"}`。

## CheckType 语义

| 类型 | 端点 | 语义 | 典型 Checker |
|------|------|------|-------------|
| `Readiness` | `/readyz` | 能否接收流量 | DB、Redis、消息队列 |
| `Liveness` | `/healthz` | 进程是否卡死 | goroutine 泄漏、死锁检测 |
| `Readiness\|Liveness` | 两者 | 同时参与 | 磁盘空间、关键文件 |

Kubernetes 行为：
- Liveness 失败 → 重启 Pod
- Readiness 失败 → 从 Service 摘除，不再接收流量

## 配置选项

```go
// 自定义 per-check 超时（默认 5s）
registry := health.NewRegistry(
    health.WithTimeout(3 * time.Second),
)
```

## Checker 接口

简单场景用 `RegisterFunc`，复杂场景实现 `Checker` 接口：

```go
type Checker interface {
    Name() string
    Check(ctx context.Context) CheckResult
}
```

```go
type postgresChecker struct {
    db *sql.DB
}

func (c *postgresChecker) Name() string { return "postgres" }

func (c *postgresChecker) Check(ctx context.Context) health.CheckResult {
    if err := c.db.PingContext(ctx); err != nil {
        return health.CheckResult{
            Status:  health.StatusDown,
            Message: err.Error(),
        }
    }
    return health.CheckResult{Status: health.StatusUp}
}

registry.Register(&postgresChecker{db: db}, health.Readiness)
```

`Check` 方法收到的 `ctx` 已携带 per-check timeout，checker 应使用此 ctx 进行 I/O 操作。

## 执行模型

```
CheckReadiness(ctx)
    ↓
过滤 Readiness 类型的 checker
    ↓
并发启动 goroutine，每个 checker 独立 context.WithTimeout
    ↓
等待全部完成，聚合结果
    ↓
任一 StatusDown → 整体 StatusDown → HTTP 503
全部 StatusUp   → 整体 StatusUp   → HTTP 200
```
