# kratoscarf examples

基于 Kratos Layout 的示例应用，演示 kratoscarf 各模块与 Kratos 框架的集成方式。

## 目录结构

```
examples/
├── app/
│   └── auth-app/              # 认证示例 (JWT + Session)
└── third_party/               # Proto 依赖
```

## 运行示例

```bash
cd examples
go run ./app/<app-name>/cmd/<app-name>/ -conf ./app/<app-name>/configs
```

各示例的详细说明见对应 app 目录下的 README。
