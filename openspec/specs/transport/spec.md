## 需求

### 需求:直接复用 kratos transport 类型

next 必须直接使用 `kratos/transport/http.Server`、`kratos/transport/grpc.Server` 等原生类型，禁止重新定义同名类型或拷贝其内部实现。

#### 场景:返回 kratos 原生 Server 类型

- **当** 业务调用 `transport.NewHTTPServer(c, cfg)`
- **那么** 返回类型必须是 `*kratos/transport/http.Server`，业务可直接使用 kratos 文档中的所有方法

#### 场景:不存在 next/transport/http.Server 类型

- **当** 在 next 包结构中查找 transport.Server 自定义类型
- **那么** `next/transport/http` 与 `next/transport/grpc` 包**必须不导出**独立 `Server` 结构体实现（拷贝 kratos 源码）；仅允许 `deprecated.go` 中的薄包装（嵌入 `*khttp.Server` 以保留 `Route()` 过渡 API）
- **过渡条款**：`transport/http/deprecated.go` 必须在 `v3.0.0` 之前删除

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

### 需求:删除拷贝代码

next 仓库**必须**不再保留与 kratos 上游高度相似的拷贝代码，**禁止**新增此类拷贝。例外：`internal/{host,endpoint,matcher}/` 因 kratos 未公开公共 API 必须保留为自维护拷贝。

#### 场景:transport/http 拷贝代码清零

- **当** 检查 `next/transport/http/` 目录
- **那么** 必须不存在 `server.go`、`client.go`、`codec.go`、`context.go`、`binding/`、`status/` 等从 kratos 拷贝的文件或子包；仅允许保留 `transport/http/deprecated.go`

#### 场景:transport/grpc 拷贝代码清零

- **当** 检查 `next/transport/grpc/` 目录
- **那么** 必须不存在 `server.go`、`client.go`、`codec.go`、`balancer.go`、`interceptor.go`、`transport.go` 等从 kratos 拷贝的文件；`resolver/` 子包保留待独立评估

#### 场景:internal 包必须保留

- **当** 检查 `next/internal/{host,endpoint,matcher}/` 目录
- **那么** 这些包**必须保留**为 next 自维护的拷贝。原因：kratos 把它们放在 `kratos/v2/internal/` 不公开 API，外部包 import 会编译失败
- **后续工作**：评估能否上游推动 kratos 公开这些工具包，届时再考虑替换

### 需求:next 独有 Router DSL v2

next 必须提供独立子包 `transport/httpx` 实现 Router DSL，不持有自定义 Server 类型，而是以 `*khttp.Server` 作为底层依赖。

#### 场景:Router 构造与 kratos Server 解耦

- **当** 业务调用 `httpx.NewRouter(srv *khttp.Server, prefix string, m ...MiddlewareFunc)`
- **那么** 返回的 Router 必须通过 `srv.Route(prefix)` 在 kratos Server 上注册路由组，不依赖任何 next 自维护的 Server 结构体

#### 场景:Router 支持 GET/POST/PUT/DELETE/Group 等方法

- **当** 业务调用 `r.GET(path, handler, middleware...)` 等方法
- **那么** Router 必须正确将 handler 注册到底层 kratos Server，并保证与 v1 nexthttp.Router 等价的行为

#### 场景:Context 接口保留并解耦

- **当** 业务实现 `func(httpx.Context) error` 风格的 handler
- **那么** `httpx.Context` 必须提供 `Request() / Response() / Bind(v) / JSON(code, v) / Vars()` 等方法

#### 场景:Context 对象池化

- **当** 并发请求触发 Router handler
- **那么** httpx 必须使用 `sync.Pool` 复用 Context 实例

### 需求:用户感知一致性

业务代码看到的 Server 类型必须与 kratos 文档完全一致，禁止 next 引入额外的包装类型导致用户需要学习两套 API。

#### 场景:Server 方法可直接调用

- **当** 业务获得 `srv, _ := transport.NewHTTPServer(c, cfg)` 后
- **那么** 必须能直接调用 `srv.HandlePrefix(...)`、`srv.Use(...)`、`srv.WalkRoute(...)` 等 kratos `*http.Server` 的所有方法，无需类型断言或额外包装

#### 场景:Server 可直接传入 kratos.App

- **当** 业务调用 `kratos.New(kratos.Server(srv))`
- **那么** 必须成功通过编译并正常运行，禁止 next 引入与 kratos 不兼容的 Server 类型
