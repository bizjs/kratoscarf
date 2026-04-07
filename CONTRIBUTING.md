# 贡献指南

感谢你对 kratoscarf 的贡献兴趣。以下是参与项目开发的指南。

## 开发环境

- Go 1.25+
- golangci-lint（`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`）

```bash
git clone git@github.com:bizjs/kratoscarf.git
cd kratoscarf
make ci   # 验证环境：格式化 + 静态检查 + 全量测试
```

## 工作流程

1. Fork 仓库，创建功能分支
2. 开发、测试、确保 `make ci` 通过
3. 提交 PR 到 `main` 分支

## Commit 规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)，Changelog 从 commit message 自动生成。

```
<type>: <description>

feat:     新功能
fix:      Bug 修复
refactor: 重构（不改变行为）
perf:     性能优化
docs:     文档
test:     测试
chore:    构建/CI/依赖等杂项
```

示例：

```
feat: add API versioning support for router
fix: JWT middleware skips auth on non-HTTP transport
docs: update auth.md with session cookie security notes
```

## 代码规范

### 通用

- `make lint` 必须通过（golangci-lint 配置见 `.golangci.yml`）
- 所有导出的函数、类型、常量必须有 godoc 注释
- 不添加不必要的注释 — 代码本身应该清晰

### 包设计

- **零包间耦合** — 核心包（router/validation/response）之间不能互相 import
- **鸭子类型协作** — 包间通过接口协作（如 `HTTPStatus()`/`BizCode()`），不通过具体类型
- **Contrib 隔离** — 引入新的外部依赖时，放在 `contrib/` 下独立 go.mod
- **不为假设需求设计** — 不做"也许将来会用到"的抽象

### 错误处理

三种模式，按场景选择：

| 场景 | 模式 | 示例 |
|------|------|------|
| 面向 HTTP 客户端的业务错误 | `response.BizError` | `return response.ErrNotFound.WithMessage("user not found")` |
| 包级别可预期的固定错误 | Sentinel error + 鸭子接口 | `var ErrNoToken = &authError{...}` |
| 包内部错误传播 | `fmt.Errorf` + `%w` | `return fmt.Errorf("pkg: context: %w", err)` |

规则：
- 面向客户端的错误必须实现 `HTTPStatus() int` 或使用 `BizError`
- 永远不要在客户端响应中暴露 `err.Error()` 原始内容

### 命名

- 构造函数：`New*`（如 `NewRouter`、`NewScheduler`）
- 配置选项：`With*`（如 `WithValidator`、`WithTimeout`）
- util 子包函数：`算法 + 动作`（如 `BcryptHash`、`AESGCMEncrypt`）
- 包名参与语义：`id.ULID()` 而不是 `id.New()`

### Wire ProviderSet

- 如果 ProviderSet 为空或只有一行构造函数，不要创建 `provider.go`
- 只在构造函数有多个依赖需要 Wire 注入时才提供 ProviderSet

## 测试

- 所有新功能必须有测试
- 使用标准库 `testing`，不引入测试框架
- 安全相关代码（auth/crypto）需要覆盖边界条件和错误路径
- Benchmark 覆盖核心路径（router/response/validation）

```bash
make test       # 运行根模块测试
make test-all   # 运行所有模块测试（含 contrib）

# 运行特定包测试
go test ./auth/jwt/ -v

# 运行 benchmark
go test ./router/ -bench=. -benchmem
```

## 新增包的检查清单

添加新包时，确认以下事项：

- [ ] 包放在正确的位置（核心包 vs contrib）
- [ ] 有完整的测试文件
- [ ] 所有导出 API 有 godoc 注释
- [ ] `make lint` 通过
- [ ] 更新 `README.md` 包概览表
- [ ] 如果是 contrib 包，有独立 go.mod
- [ ] 如果涉及安全（认证/加密），通过安全审查

## 发布

发布通过 GitHub Actions 手动触发（见 `.github/workflows/release.yml`）。

版本号遵循 [Semantic Versioning](https://semver.org/)：
- 破坏性变更 → major（v2.0.0）
- 新功能 → minor（v0.2.0）
- Bug 修复 → patch（v0.1.1）

所有模块统一版本号。
