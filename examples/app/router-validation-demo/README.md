# router-validation-demo

演示 kratoscarf `router` + `validation` + `response` + `auth/session` 如何集成到 Kratos 应用中。

一个带 Session 认证的 TODO 列表 REST API。

## 核心集成点

```go
// 一行配置，三层自动拦截
r := router.NewRouter(srv,
    router.WithValidator(validation.New()),    // Bind 自动验证 (≈ NestJS ValidationPipe)
    router.WithResponseWrapper(response.Wrap), // Success 自动包装 (≈ NestJS Interceptor)
)
// + ErrorEncoder                              // 错误自动格式化 (≈ NestJS ExceptionFilter)
```

Handler 零模板代码：

```go
func (s *Svc) Create(ctx *router.Context) error {
    var req CreateRequest                          // struct tag 定义验证规则
    if err := ctx.Bind(&req); err != nil {         // 自动 bind + validate
        return err                                  // → 422 {code:42200, data:[field errors]}
    }
    todo := &Todo{Title: req.Title}
    s.repo.Create(ctx.Context(), todo)
    return ctx.Success(todo)                        // → 200 {code:0, data:{...}}
}
```

## 功能清单

| 功能 | 实现方式 |
|------|---------|
| 路由注册 | `r.Group()`, `api.GET()`, `api.POST()` |
| 路径参数 | `ctx.Param("id")` |
| 查询参数 | `ctx.Query("page")`, `ctx.QueryDefault("pageSize", "10")` |
| 分页 | `response.PageRequest` + `response.NewPageResponse` |
| 请求验证 | `ctx.Bind(&req)` 自动验证 (`validate` struct tag) |
| 成功响应 | `ctx.Success(data)` 自动包装 `{code, message, data}` |
| 自定义状态码 | `ctx.JSON(201, data)` |
| 204 无内容 | `ctx.NoContent()` |
| 业务错误 | `return response.ErrNotFound.WithMessage(...)` |
| Session 认证 | `session.Middleware` + `requireSession` guard |

## 运行

```bash
cd examples
go run ./app/router-validation-demo/cmd/router-validation-demo/ -conf ./app/router-validation-demo/configs
```

## 测试

```bash
# 登录
curl -s -X POST localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice"}' -c cookies.txt

# 列表 (分页)
curl -s 'localhost:8080/api/todos?page=1&pageSize=2' -b cookies.txt

# 详情
curl -s localhost:8080/api/todos/1 -b cookies.txt

# 创建
curl -s -X POST localhost:8080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"New task"}' -b cookies.txt

# 创建 (验证失败)
curl -s -X POST localhost:8080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{}' -b cookies.txt

# 更新
curl -s -X PUT localhost:8080/api/todos/1 \
  -H 'Content-Type: application/json' \
  -d '{"completed":true}' -b cookies.txt

# 删除
curl -s -X DELETE localhost:8080/api/todos/1 -b cookies.txt -w "\nHTTP %{http_code}\n"

# 登出
curl -s -X POST localhost:8080/logout -b cookies.txt -c cookies.txt

# 未登录访问 (expect 401)
curl -s localhost:8080/api/todos -b cookies.txt
```

## 响应示例

成功：
```json
{"code": 0, "message": "ok", "data": {"id": "1", "title": "...", "completed": false}}
```

分页：
```json
{"code": 0, "message": "ok", "data": {"items": [...], "total": 3, "page": 1, "pageSize": 2}}
```

验证失败 (422)：
```json
{"code": 42200, "message": "validation failed", "data": [{"field": "title", "rule": "required", "message": "is required"}]}
```

未找到 (404)：
```json
{"code": 40400, "message": "todo 999 not found"}
```

未登录 (401)：
```json
{"code": 40100, "message": "login required"}
```
