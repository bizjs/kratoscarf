---
layout: home

hero:
  name: kratoscarf
  text: Kratos Web 开发增强层
  tagline: 不替代 Kratos，增强 Kratos。Gin/Echo 风格的路由 + 统一响应 + 自动验证 + 认证开箱即用。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/bizjs/kratoscarf

features:
  - title: Web 风格路由
    details: Gin/Echo 风格的 GET/POST/Group/Use API，无缝构建在 Kratos HTTP Server 之上。
  - title: 三层自动拦截
    details: 类 NestJS 的 ValidationPipe + TransformInterceptor + ExceptionFilter，一次配置自动生效。
  - title: 零包间耦合
    details: router/validation/response 互不 import，通过鸭子类型接口协作，按需组合。
  - title: 认证开箱即用
    details: JWT（生成/验证/刷新/撤销）+ Session（Cookie/Redis），中间件一键集成。
  - title: 安全默认值
    details: 算法校验、Secret 强度验证、Secure Headers、CORS、RequestID 内置。
  - title: Contrib 生态
    details: Redis Session Store、定时任务调度，独立模块按需引入，不污染核心依赖。
---
