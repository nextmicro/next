## 修改需求

### 需求:kratos 中间件适配器形态

`tracing/recovery/metrics/metadata/circuitbreaker/bbr` 等基础中间件的 factory **必须**作为 kratos 中间件的配置翻译适配器，**禁止**重复实现 kratos 已有逻辑。next 业务特定能力（如 b3 propagator、Sentry 集成、Prometheus 命名约定）**必须**通过 Option 注入而非独立实现。

#### 场景:tracing factory 复用 kratos 实现

- **当** 实现 `server.tracing` 或 `client.tracing` factory
- **那么** factory 内部必须调用 `kratos/middleware/tracing.Server(opts...)` 或 `Client(opts...)`，仅负责将 `configv1.Middleware.Options` 翻译为 `tracing.Option`
- **禁止** factory 中重新实现 span 创建、context propagation、metadata 注入等 kratos 已提供的逻辑

#### 场景:recovery factory 复用 kratos 实现

- **当** 实现 `server.recovery` factory
- **那么** factory 内部必须调用 `kratos/middleware/recovery.Recovery(opts...)`；Sentry 集成与自定义 panic logger 必须通过 `recovery.WithHandler(HandlerFunc)` Option 注入
- **禁止** factory 中重新实现 panic recover 机制

#### 场景:metrics factory 复用 kratos 实现

- **当** 实现 `server.metrics` 或 `client.metrics` factory
- **那么** factory 内部必须调用 `kratos/middleware/metrics.Server(opts...)` 或 `Client(opts...)`；next 业务的 Prometheus 标签命名约定（如 `http_server_requests_total{method,path,code}`）必须通过 `metrics.WithRequests(counter)` 等 Option 注入
- **禁止** factory 中重新实现 metrics 收集逻辑

#### 场景:metadata factory 复用 kratos 实现

- **当** 实现 `server.metadata` 或 `client.metadata` factory
- **那么** factory 内部必须调用 `kratos/middleware/metadata.Server(opts...)` 或 `Client(opts...)`；next 业务的 prefix 规则（如 `x-md-*`）必须通过 `metadata.WithPrefix(prefix)` Option 注入

#### 场景:circuitbreaker factory 复用 kratos 实现

- **当** 实现 `client.circuitbreaker` factory
- **那么** factory 内部必须调用 `kratos/middleware/circuitbreaker.Client(opts...)`；aegis 参数（如 `success`、`request`、`bucket`）必须通过 Option 翻译注入

#### 场景:bbr factory 复用 kratos 实现

- **当** 实现 `server.bbr` 或 `server.ratelimit` factory
- **那么** factory 内部必须调用 `kratos/middleware/ratelimit.Server(opts...)`；ratelimit 已默认使用 bbr 限流器
- **禁止** 重复包装 aegis ratelimit 库

#### 场景:适配器代码量约束

- **当** 任意 kratos 中间件的适配器实现完成
- **那么** 该适配器的 `adapter.go` 主文件非测试代码**必须**不超过 100 行（强制约束）；如确实超出，PR 中必须说明理由并经 maintainer 同意

#### 场景:业务 yaml 配置零修改

- **当** 业务方已有 yaml 配置（如 `http.middlewares: [{name: server.tracing, options: {...}}]`）
- **那么** 适配器重写后**必须**对相同 yaml 产生与 v1 等价的运行行为（追踪 span 名、metrics 标签、recovery 行为等）
- **禁止** 在本变更中改变 yaml 字段名或语义

## 新增需求

### 需求:Option 翻译层标准化

每个适配器 sub-package **必须**提供独立的 Option 翻译层函数，将 `configv1.Middleware` 反序列化为 Option 列表。

#### 场景:统一翻译函数签名

- **当** 实现任何 kratos middleware 适配器 sub-package
- **那么** 该 sub-package 必须暴露 `optionsFromConfig(cfg *configv1.Middleware) ([]Option, error)` 函数（或同等签名），factory 内部调用此函数完成 yaml 配置到 Option 的映射

#### 场景:翻译层独立可测

- **当** 编写适配器单元测试
- **那么** 必须为 `optionsFromConfig` 编写独立测试用例，覆盖：合法配置、空配置、非法字段、`anypb.Any` 反序列化失败等场景

#### 场景:翻译层错误必须可定位

- **当** `optionsFromConfig` 返回错误
- **那么** 错误必须包含 sub-package 名称（如 `"tracing: ..."`），且必须包装原始错误以保留 stack trace

### 需求:适配器 sub-package 目录结构

每个适配器 sub-package **必须**采用统一的目录结构，便于业务方查阅扩展规则。

#### 场景:文件职责分离

- **当** 查看任意适配器 sub-package（除 `logging` 外）
- **那么** 必须包含以下文件：
  - `option.go`：定义 `Option = ktype.Option` 类型别名与 next 特有 Option 函数
  - `adapter.go`：实现 `Server(opts ...Option) middleware.Middleware` 与 `Client(opts ...Option) middleware.Middleware`，≤ 100 行
  - `config.go`：实现 `optionsFromConfig(cfg) ([]Option, error)`，≤ 50 行
  - `factory.go`：通过 `func init()` 或显式注册函数将 factory 注册到 `middleware.DefaultRegistry`
  - `*_test.go`：包含 v1 行为契约测试

#### 场景:logging 例外

- **当** 查看 `middleware/logging` sub-package
- **那么** **允许**保留独立目录结构（非适配器形态），因 spec 已明确 next 自有日志格式独立维护

### 需求:Option 类型代理 kratos

适配器 sub-package 的 `Option` 类型**必须**为对应 kratos middleware Option 的类型别名，让业务方可直接传递 kratos middleware Option。

#### 场景:Option 类型别名

- **当** 业务方使用适配器 sub-package（如 `tracing`）
- **那么** `tracing.Option` 必须为 `type Option = ktracing.Option` 类型别名，业务可混合使用 next 特有 Option 与 kratos 原生 Option

#### 场景:next 特有 Option 返回 kratos Option

- **当** 实现 next 特有 Option（如 `tracing.WithB3Propagator()`）
- **那么** 该函数必须返回 `Option`（即 `ktracing.Option`），通过组合 kratos Option 完成 next 业务定制
- **禁止** 引入独立的 next Option 类型导致与 kratos Option 不兼容
