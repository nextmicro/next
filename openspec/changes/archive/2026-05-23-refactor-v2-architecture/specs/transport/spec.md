## 新增需求

### 需求:直接复用 kratos transport 类型

next 必须直接使用 `kratos/transport/http.Server`、`kratos/transport/grpc.Server` 等原生类型，禁止重新定义同名类型或拷贝其内部实现。

#### 场景:返回 kratos 原生 Server 类型

- **当** 业务调用 `transport.NewHTTPServer(c, cfg)`
- **那么** 返回类型必须是 `*kratos/transport/http.Server`，业务可直接使用 kratos 文档中的所有方法

#### 场景:不存在 next/transport/http.Server 类型（渐进式：本次 v2 v2.0.0 已新建顶层 `transport.NewHTTPServer/NewGRPCServer` 返回 kratos 原生类型作为新入口；旧 `transport/http/Server` / `transport/grpc/Server` 自定义类型暂时并行存在，于独立后续变更 `refactor-v2-transport-cleanup` 中删除）

- **当** 在 next 包结构中查找 transport.Server 自定义类型
- **那么 (v2.0.0 阶段)** `transport/http` 与 `transport/grpc` 包**可以暂时保留** `Server` 结构体类型，标注为 Deprecated；新业务必须通过 `transport.NewHTTPServer / NewGRPCServer` 创建实例
- **那么 (Phase 3B 完成后)** `next/transport/http` 与 `next/transport/grpc` 包**必须不导出** `Server` 结构体类型，只导出 next 独有的扩展（如 Router DSL 重设计后的接口）

### 需求:配置驱动的 ServerBuilder

next 必须为每种 transport 提供 `NewXxxServer(c *Container, cfg *configv1.Server) (*kxxx.Server, error)` 形式的构造函数，将业务配置翻译为 kratos `ServerOption`。

#### 场景:从配置构造 HTTP Server

- **当** 业务调用 `transport.NewHTTPServer(c, cfg)` 且配置包含 `addr` 与 `timeout`
- **那么** 返回的 Server 必须已经应用 `http.Address(cfg.Addr)` 与 `http.Timeout(cfg.Timeout)` 等 kratos Option

#### 场景:从配置构造 gRPC Server

- **当** 业务调用 `transport.NewGRPCServer(c, cfg)`
- **那么** 返回的 Server 必须已根据配置应用相应的 `grpc.Address` / `grpc.Timeout` / `grpc.TLSConfig` 等 Option

#### 场景:无配置时使用默认值

- **当** 调用方传入 nil 或不完整的配置
- **那么** 构造函数必须使用合理默认值（如 `:0` 监听地址、1s 超时）成功创建 Server，且不返回错误

### 需求:中间件自动装配

ServerBuilder 必须从 Container 中解析 `middleware.Registry`，按配置自动构建并应用中间件链。

#### 场景:自动装配 server.* 中间件

- **当** 配置中声明 `http.middlewares: [tracing, logging, recovery]`
- **那么** ServerBuilder 必须调用 `registry.Build("server", configs)` 构建中间件切片并通过 `http.Middleware(...)` 注入到 Server

#### 场景:中间件构建失败的处理

- **当** Registry.Build 返回错误（例如未知 middleware 名称）
- **那么** ServerBuilder 必须返回包装错误，禁止静默丢弃中间件继续构造 Server

### 需求:删除拷贝代码（渐进式）

**最终目标**：仓库中不再存在与 kratos 上游高度相似的拷贝代码。

**v2.0.0 阶段说明**：本次重构已通过新增 `transport.NewHTTPServer / NewGRPCServer` 顶层工厂提供 v2 业务入口，旧 `transport/http` 与 `transport/grpc` 子包并行存在以兼容 v1 业务方（其中 `transport/http/router.go` 包含 next 独有的 mux 路由 DSL，删除前需重新设计基于 kratos.HTTPServer 的替代方案）。子包删除工作由独立变更 `refactor-v2-transport-cleanup` 承担。

#### 场景:transport/http 拷贝代码清零（Phase 3B 目标）

- **当** 检查 `next/transport/http/` 目录（Phase 3B 完成后）
- **那么** 必须不存在 `server.go`、`client.go`、`codec.go`、`context.go`、`resolver.go`、`binding/` 等从 kratos 拷贝的文件；仅保留 next 独有的扩展点（如自定义 Router 接口的新形态）

#### 场景:transport/grpc 拷贝代码清零（Phase 3B 目标）

- **当** 检查 `next/transport/grpc/` 目录（Phase 3B 完成后）
- **那么** 必须不存在 `server.go`、`client.go`、`codec.go`、`balancer.go`、`interceptor.go`、`transport.go` 等从 kratos 拷贝的文件

#### 场景:internal 包必须保留（spec 修订）

- **当** 检查 `next/internal/{host,endpoint,matcher}/` 目录
- **那么** 这些包**必须保留**为 next 自维护的拷贝。原因：kratos 把它们放在 `kratos/v2/internal/` 不公开 API，外部包 import 会编译失败。Spec 原版"清零"目标在事实层面不可行。
- **后续工作**：评估能否上游推动 kratos 公开这些工具包（如 `kratos/v2/pkg/host`），届时再考虑替换。

### 需求:用户感知一致性

业务代码看到的 Server 类型必须与 kratos 文档完全一致，禁止 next 引入额外的包装类型导致用户需要学习两套 API。

#### 场景:Server 方法可直接调用

- **当** 业务获得 `srv, _ := transport.NewHTTPServer(c, cfg)` 后
- **那么** 必须能直接调用 `srv.HandlePrefix(...)`、`srv.Use(...)`、`srv.WalkRoute(...)` 等 kratos `*http.Server` 的所有方法，无需类型断言或额外包装

#### 场景:Server 可直接传入 kratos.App

- **当** 业务调用 `kratos.New(kratos.Server(srv))`
- **那么** 必须成功通过编译并正常运行，禁止 next 引入与 kratos 不兼容的 Server 类型
