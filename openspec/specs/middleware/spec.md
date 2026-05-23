## 需求

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

### 需求:kratos 中间件适配器形态

`tracing/recovery/metrics/metadata/circuitbreaker/bbr` 等基础中间件的 factory **必须**作为 kratos 中间件（或 kratos 生态等价物）的配置翻译适配器，**禁止**重复实现 kratos 已有逻辑。next 业务特定能力**必须**通过 Option 注入。

#### 场景:tracing factory 复用 kratos Tracer

- **当** 实现 `server.tracing` 或 `client.tracing` factory
- **那么** 必须使用 `kratos/middleware/tracing.NewTracer` 处理 propagation/inject/extract；next 特有 span  enrichment（httpconv 属性、业务 span 名、响应 trace 头）保留在 `span.go` / `adapter.go`
- **禁止** 在 factory 中手写 W3C/B3 propagator 逻辑

#### 场景:recovery factory 复用 kratos 实现

- **当** 实现 `server.recovery` factory
- **那么** factory 内部必须调用 `kratos/middleware/recovery.Recovery(opts...)`；next panic 日志与 stack 配置通过 `recovery.WithHandler` Option 注入

#### 场景:metrics factory 使用 next Prometheus 命名

- **当** 实现 `server.metrics` 或 `client.metrics` factory
- **那么** 必须使用 `kratos/contrib/metrics/prometheus` 作为 metrics 后端，并通过 Option 注入 next 业务指标名（见 `pkg/metrics`）
- **说明**：kratos OTEL `middleware/metrics` 与 next Prometheus 指标 schema 不兼容，metrics 子包允许保留 next 采集逻辑，但须采用 adapter/config/factory 结构

#### 场景:metadata / circuitbreaker / bbr factory 复用 kratos 实现

- **当** 实现对应 factory
- **那么** 必须分别调用 `kratos/middleware/metadata`、`circuitbreaker`、`ratelimit` 的 Server/Client API；yaml 参数通过 `optionsFromConfig` 翻译为 Option

#### 场景:适配器代码量约束

- **当** 任意适配器 `adapter.go` 完成
- **那么** 非测试代码**应当**不超过 100 行；tracing 等因 span 状态记录可超出，须在 PR 说明

#### 场景:业务 yaml 配置零修改

- **当** 业务方已有 yaml middleware 配置
- **那么** 适配器重写后**必须**对相同 yaml 产生等价运行行为（span 名、metrics 标签、recovery 行为等）

### 需求:Option 翻译层标准化

每个适配器 sub-package（除 `logging`）**必须**提供 `optionsFromConfig(cfg *configv1.Middleware) ([]Option, error)`（或 scope 变体），factory 内部调用完成 yaml → Option 映射。

#### 场景:翻译层错误可定位

- **当** `optionsFromConfig` 返回错误
- **那么** 错误必须包含 sub-package 名称（如 `"tracing: ..."`）

### 需求:适配器 sub-package 目录结构

#### 场景:文件职责分离

- **当** 查看任意适配器 sub-package（除 `logging` 外）
- **那么** 必须包含 `adapter.go`、`config.go`、`factory.go`；`option.go` 按需

#### 场景:logging 例外

- **当** 查看 `middleware/logging`
- **那么** **允许**保留独立实现（next 业务 access log 字段约定）

### 需求:next 自有中间件的独立性

next 自有的中间件（`middleware/logging`）必须独立维护，不通过 kratos 适配器实现。

#### 场景:next/logging 不依赖 kratos/logging

- **当** 使用 `next/middleware/logging`
- **那么** 其实现必须直接使用 `nextmicro/logger` 与 next 自有日志字段约定

### 需求:中间件链的 Server/Client 分离

Registry 必须在 scope 层面区分服务端中间件（`server.*`）与客户端中间件（`client.*`）。

#### 场景:同名 middleware 服务端客户端分离

- **当** 注册 tracing 的服务端与客户端版本
- **那么** 必须分别为 `server.tracing` 与 `client.tracing` 注册不同 factory

#### 场景:Build 按 scope 过滤

- **当** 调用 `Build("server", configs)`
- **那么** 只能装配 `server.*` 前缀的 factory
