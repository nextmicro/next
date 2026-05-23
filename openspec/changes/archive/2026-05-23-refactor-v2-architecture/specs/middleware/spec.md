## 新增需求

### 需求:Registry 实例化、消除全局单例

中间件 Registry 必须为可实例化类型，由 Container 持有并注入，禁止使用包级 `globalRegistry` 单例。

#### 场景:多实例 Registry 互不影响

- **当** 测试代码创建两个独立的 `Registry` 实例并各自注册不同 factory
- **那么** 两个 Registry 必须互不影响，禁止通过包级单例共享状态

#### 场景:通过 Container 解析 Registry

- **当** 业务需要注册自定义 middleware factory
- **那么** 必须通过 `container.Resolve[middleware.Registry](c).Register(name, factory)`，禁止使用 `chain.Register(name, factory)` 形式的包级函数访问全局状态

### 需求:Factory 注册命名规范

Factory 注册名称必须使用形如 `<scope>.<name>` 的全限定名（例如 `server.tracing`、`client.logging`），禁止运行时通过字符串拼接修改输入配置的 Name 字段。

#### 场景:Factory 用全限定名注册

- **当** 注册一个用于服务端的 tracing factory
- **那么** 必须调用 `registry.Register("server.tracing", factory)`，禁止使用 `Register("tracing", ...)` 后在 Build 时拼接

#### 场景:Build 时不修改输入

- **当** 调用 `registry.Build("server", configs)` 构建中间件链
- **那么** 传入的 `configs` 切片元素的 `Name` 字段必须保持原样，禁止 in-place 修改

### 需求:配置驱动的中间件链装配

Registry 必须提供 `Build(scope string, configs []*configv1.Middleware) ([]middleware.Middleware, error)` 方法，按配置顺序构建 kratos 中间件切片。

#### 场景:按配置顺序构建链

- **当** 配置文件按 `[tracing, logging, recovery]` 顺序声明
- **那么** `Build` 返回的中间件切片顺序必须与配置一致，禁止重新排序

#### 场景:未注册的 middleware 名称

- **当** 配置中引用了未注册的 middleware 名称
- **那么** `Build` 必须返回包含未知名称的错误，禁止静默忽略；同时提供可选的"warn-only"模式记录日志并跳过

#### 场景:Options 反序列化失败

- **当** 配置中某个 middleware 的 `Options`（`anypb.Any`）无法反序列化到 factory 期望的类型
- **那么** `Build` 必须返回包含 middleware 名称和反序列化错误的错误

### 需求:kratos 中间件适配器形态（渐进式）

`tracing/recovery/metrics/metadata/circuitbreaker/bbr` 等基础中间件的 factory **目标**是仅作为 kratos 中间件的配置翻译适配器，禁止重复实现 kratos 已有逻辑。

**v2.0.0 阶段说明**：本次重构已完成 Registry / Builder 解耦（消除字符串 mutation 与全局单例），但 7 个 sub-package 内部仍保留 next 自有实现（含 next 业务定制：field 过滤、b3 propagator、业务字段名规范等）。改写为 kratos 纯适配器形态，需逐个 middleware 评估保留哪些定制能力，由后续独立变更承担。

#### 场景:tracing factory 复用 kratos 实现（Phase 4B 目标）

- **当** 实现 `server.tracing` factory
- **那么 (v2.0.0 阶段)** 现有 next 自有实现继续工作，不强制重写
- **那么 (Phase 4B 完成后)** factory 内部必须调用 `kratos/middleware/tracing.Server(opts...)` 并仅负责将 `configv1.Middleware.Options` 翻译为 `tracing.Option`

#### 场景:适配器代码量约束（Phase 4B 完成后生效）

- **当** 任意 kratos 中间件的适配器实现完成（Phase 4B 后）
- **那么** 该适配器的非测试代码应不超过 100 行（指引性指标，超出需在 PR 中说明原因）

### 需求:next 自有中间件的独立性

next 自有的中间件（当前为 `middleware/logging`）必须独立维护，不通过 kratos 适配器实现，以保证 next 业务日志格式的统一约定。

#### 场景:next/logging 不依赖 kratos/logging

- **当** 使用 `next/middleware/logging` 中间件
- **那么** 其实现必须直接使用 `nextmicro/logger` 与 next 自有的日志字段约定，禁止依赖 `kratos/middleware/logging` 的内部实现

#### 场景:next/logging 作为 factory 注册

- **当** 业务在配置中使用 `next.logging` 或类似名称
- **那么** 框架必须通过相同的 Registry 机制注册并装配此中间件，统一装配流程

### 需求:中间件链的 Server/Client 分离

Registry 必须在 scope 层面区分服务端中间件（`server.*`）与客户端中间件（`client.*`），同一逻辑中间件的服务端版本与客户端版本由不同 factory 提供。

#### 场景:同名 middleware 服务端客户端分离

- **当** 注册 tracing 的服务端与客户端版本
- **那么** 必须分别为 `server.tracing` 与 `client.tracing` 注册不同 factory，禁止单一 factory 通过运行时判断角色

#### 场景:Build 按 scope 过滤

- **当** 调用 `Build("server", configs)`
- **那么** 只能装配 `server.*` 前缀的 factory；调用 `Build("client", configs)` 只能装配 `client.*` 前缀的 factory
