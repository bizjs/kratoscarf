# 简介

kratoscarf 是 [Kratos](https://go-kratos.dev/) 微服务框架的 Web 开发增强层。

## 它是什么

一组独立的 Go 包，为 Kratos HTTP Server 补充 Web 开发中最常用的功能：

- **路由** — Gin/Echo 风格的 `GET`/`POST`/`Group`/`Use` API
- **验证** — `Bind()` 自动验证请求体
- **响应** — `{code, message, data}` 信封格式 + 业务错误自动编码
- **认证** — JWT + Session，中间件一键集成
- **中间件** — CORS、Secure Headers、RequestID
- **健康检查** — Liveness/Readiness 分离
- **工具** — ID 生成、加密/哈希

## 它不是什么

- 不是一个新框架 — 不替代 Kratos，与 Kratos 原生 API 共存
- 不是全家桶 — 没有 ORM、没有配置中心、没有服务发现
- 不强制架构 — 不要求特定的目录结构或依赖注入方式

## 设计原则

1. **增强而非替代** — 用户同时 import `go-kratos/kratos` 和 `bizjs/kratoscarf`
2. **零包间耦合** — router/validation/response 互不 import，通过鸭子类型接口协作
3. **Contrib 隔离** — 重依赖（Redis）独立 go.mod，核心几乎零第三方依赖
4. **不为假设需求设计** — 只实现经过验证的、有实际业务价值的功能

## 包概览

### 核心包

| 包 | 说明 |
|---|---|
| [`router`](/router) | Web 风格路由 + Context |
| [`response`](/response) | 统一响应、BizError、分页 |
| [`validation`](/validation) | 请求验证 |
| [`auth/jwt`](/auth) | JWT 认证 |
| [`auth/session`](/auth) | Session 认证 |
| [`middleware`](/middleware) | CORS、Secure Headers、RequestID |
| [`health`](/health) | 健康检查 |
| [`util/id`](/util) | ID 生成（UUID/ULID/Short） |
| [`util/crypto`](/util) | 加密/哈希（AES-GCM/Bcrypt/HMAC） |

### Contrib 包

| 包 | 说明 | 依赖 |
|---|---|---|
| `contrib/auth/redis` | Redis Session Store | go-redis/v9 |
| `contrib/schedule` | 定时任务（Cron + Interval） | robfig/cron/v3 |
