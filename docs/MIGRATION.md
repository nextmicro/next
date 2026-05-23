# next v1 → v2 迁移指南

next v2 是一次**破坏性重构**。本文档列出 v1 与 v2 的概念映射，便于业务方平滑升级。

> **不提供自动迁移工具**。绝大多数业务代码只需小范围修改即可。

## 总体原则

| 维度 | v1 | v2 |
|-----|----|----|
| 定位 | 复制 kratos 代码再扩展 | 装饰 kratos，不复制 |
| 全局变量 | DefaultBroker / DefaultRegistry / DefaultLogger 等 | 通过 `container.Container` 注入 |
| 中间件注册 | `chain.Register("server.x", factory)` 全局表 | `middleware.NewRegistry().Register(...)` 可实例化 |
| Loader 编排 | `runtime.Runtime` + `Loader` 接口顺序串行 | `lifecycle.Lifecycle` + `Component` 拓扑分层并行 |
| Broker 类型安全 | 业务自行 marshal `[]byte` | `broker/typed` 提供 `Publisher[T]` / `Subscriber[T]` |

---

## 1. App 入口

### v1
```go
app, err := next.New(
    next.ID("svc"),
    next.Name("orders"),
    next.Server(httpSrv, grpcSrv),
    next.Loader(myLoader),
)
```

### v2
```go
app, err := next.New(
    next.ID("svc"),
    next.Name("orders"),
    next.Server(httpSrv, grpcSrv),
    next.WithComponents(myComponent),           // 新：lifecycle.Component
    next.WithContainer(func(c *container.Container) {
        container.Register(c, mybiz.NewService())  // 新：DI 注册
    }),
)
```

`next.Loader(...)` 仍可用（标记 Deprecated）；新代码应改用 `next.WithComponents(...)`。

类型 `*next.Next` 仍可用（类型别名 `type Next = App`）。

---

## 2. Middleware Factory 注册

### v1
```go
func init() {
    chain.Register("server.tracing", injectionServer)
    chain.Register("client.tracing", injectionClient)
}
```

### v2（保持兼容期）
```go
// 不变；chain.Register 兼容函数仍工作，内部委托到 middleware.DefaultRegistry
func init() {
    chain.Register("server.tracing", injectionServer)
}
```

### v2（推荐新写法）
```go
// 业务在 App 构造期注入到自己的 Registry
app, _ := next.New(
    next.WithContainer(func(c *container.Container) {
        reg := container.MustResolve[middleware.Registry](c)
        reg.Register("server.tracing", injectionServer)
    }),
)
```

---

## 3. 中间件自动装配

### v1
```go
ms, _ := chain.BuildMiddleware("http.server", cfg.GetMiddlewares())
// 内部会修改 cfg.GetMiddlewares()[i].Name，存在 in-place mutation
```

### v2
```go
ms, err := app.MiddlewareRegistry().Build("server", cfg.GetMiddlewares())
// 不修改入参；找不到的 middleware 默认返回错误
```

---

## 4. Broker 全局变量

### v1
```go
err := broker.DefaultBroker.Publish("topic", &broker.Message{Body: data})
```

### v2
```go
b, err := container.Resolve[broker.Broker](app.Container())
err = b.Publish(ctx, "topic", &broker.Message{Body: data})
```

### v2（类型安全推荐写法）
```go
var OrderCreated = typed.Topic[*pb.Order]{
    Name:  "orders.created",
    Codec: typed.ProtoCodec[*pb.Order]{},
}

pub := typed.NewPublisher(b, OrderCreated)
err := pub.Publish(ctx, &pb.Order{ID: "123"})  // 无需手写 marshal
```

---

## 5. Broker 选项 / Wrapper

| v1 路径 | v2 路径 |
|--------|--------|
| `next/adapter/broker/kafka` | `next/broker/kafka` |
| `next/adapter/broker/wrapper/logging` | `next/broker/wrappers/logging` |
| `next/adapter/broker/wrapper/metrics` | `next/broker/wrappers/metrics` |
| `broker.Init(opts...)` 接口 | 已移除；通过构造函数注入 |
| `b.String()` / `b.Address()` | 已移除；不再属于 Broker 接口 |
| `b.Connect()` 无参 | `b.Connect(ctx)` 带 ctx |

---

## 6. Middleware blank-import

### v1
```go
// next/naxt.go 自动 blank-import 全部 7 个 middleware
import _ "github.com/nextmicro/next/middleware/bbr"  // 隐式发生
```

### v2
```go
// 业务方自己显式：
import _ "github.com/nextmicro/next/middleware/all"  // 一行启用全部

// 或精细化：
import _ "github.com/nextmicro/next/middleware/tracing"
import _ "github.com/nextmicro/next/middleware/recovery"
// 跳过 bbr/circuitbreaker（避免某些平台 aegis 初始化问题）
```

---

## 7. Transport Server 与 Router DSL

### v1（deprecated 兼容层，v3.0.0 删除）
```go
import nexthttp "github.com/nextmicro/next/transport/http"

srv := nexthttp.NewServer(nexthttp.Address(":8080"))
route := srv.Route("/api", authMiddleware)
route.GET("/users/{id}", func(c nexthttp.Context) error {
    return c.JSON(200, map[string]string{"id": c.Vars().Get("id")})
})
```

### v2（推荐）
```go
import (
    khttp "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/nextmicro/next/transport"
    "github.com/nextmicro/next/transport/httpx"
)

srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{
    ServerOptions: append(httpx.ServerOptionsFromRouter(),
        khttp.Address(":8080"),
    ),
})

route := httpx.NewRouter(srv, "/api", authMiddleware)
route.GET("/users/{id}", func(c httpx.Context) error {
    return c.JSON(200, map[string]string{"id": c.Vars().Get("id")})
})
```

返回 `*khttp.Server`（kratos 原版）。Router DSL 迁移至 `transport/httpx`，保留 `Context` / `MiddlewareFunc` 签名。

### 批量替换 import 路径

```bash
# binding / status 子包 → kratos 等价物
find . -name '*.go' -exec sed -i '' \
  's|github.com/nextmicro/next/transport/http/binding|github.com/go-kratos/kratos/v2/transport/http/binding|g' {} +

find . -name '*.go' -exec sed -i '' \
  's|github.com/nextmicro/next/transport/http/status|github.com/go-kratos/kratos/v2/errors|g' {} +

# Router DSL → httpx
find . -name '*.go' -exec sed -i '' \
  's|github.com/nextmicro/next/transport/http|github.com/nextmicro/next/transport/httpx|g' {} +
```

> macOS 使用 `sed -i ''`；Linux 使用 `sed -i`。

---

## 7b. Middleware 适配器（v2.2.0）

Phase 4B 将 `bbr/metadata/circuitbreaker/recovery` 等 sub-package 重构为 kratos 适配器形态。

**业务方无需修改 yaml**。factory 名称（`server.tracing` 等）与配置字段保持不变。

| 变更 | 业务感知 |
|------|---------|
| 内部实现改为 kratos 适配器 | 无 |
| `metrics` 仍用 Prometheus（非 OTEL middleware） | 无（指标名不变） |
| `tracing` 基于 kratos Tracer + next span  enrichment | 无（span 名、trace 响应头不变） |
| `logging` 保持 next 自有实现 | 无 |

详见 [docs/middleware-adapter-mapping.md](middleware-adapter-mapping.md) 与 [docs/middleware-logging-fields.md](middleware-logging-fields.md)。

---

## 8. Config 类型化绑定

### v1
```go
var db struct{ Datasource string }
_ = config.DefaultConfig.Value("database").Scan(&db)
```

### v2
```go
type DB struct {
    Datasource string `json:"datasource"`
    Pool       int    `json:"pool"`
}
db, err := config.Bind[DB](config.DefaultConfig, "database")
// 一行：分配 *DB + Scan + 错误包装
```

热更新：
```go
stop, _ := config.BindWatch[DB](cfg, "database", func(newDB *DB) {
    pool.SetDSN(newDB.Datasource)
})
defer stop()
```

---

## 9. Loader → Component（自实现 Loader 的业务方）

### v1
```go
type MyLoader struct{ /* ... */ }
func (l *MyLoader) Init(opts ...loader.Option) error { ... }
func (l *MyLoader) Initialized() bool                { ... }
func (l *MyLoader) Start(ctx context.Context) error  { ... }
func (l *MyLoader) Stop(ctx context.Context) error   { ... }
func (l *MyLoader) Watch() error                     { ... }
func (l *MyLoader) String() string                   { return "MyLoader" }

next.New(next.Loader(&MyLoader{}))
```

### v2（推荐：直接实现 Component）
```go
type MyComponent struct{ /* ... */ }
func (c *MyComponent) Name() string                       { return "my" }
func (c *MyComponent) Start(ctx context.Context) error    { ... }
func (c *MyComponent) Stop(ctx context.Context) error     { ... }
// 可选：Init/Watch/Health/Deps

next.New(next.WithComponents(&MyComponent{}))
```

### v2（最小改动：保持 Loader）
```go
// next.Loader(...) 仍可用；内部自动包装为 Component
next.New(next.Loader(&MyLoader{}))
```

---

## 10. 依赖声明与启动顺序

### v1
依赖 Loader 切片注册顺序串行启动；无法并行。

### v2
```go
type CacheComponent struct{}
func (*CacheComponent) Name() string { return "cache" }
func (*CacheComponent) Deps() []reflect.Type {
    return []reflect.Type{
        container.TypeOf[*sql.DB](),  // 声明依赖 DB
    }
}
// Lifecycle 自动拓扑排序：DB → Cache → ...
// 同层组件并行启动；任一失败反向回滚
```

---

## 不变（无需迁移）

- kratos.App API（ID/Name/Version/Run/Stop 等）
- transport 协议（HTTP/gRPC 消息格式）
- proto 配置（`api/config/v1/config.proto`）
- 日志库 `nextmicro/logger`
- Nacos / Consul / etcd 集成
- middleware yaml 配置与 factory 名称（v2.2.0 适配器重构为零感知）

---

## 已知不兼容点（必须修改）

| 项 | 现象 | 修复 |
|---|------|------|
| `broker.DefaultBroker` | 不存在 | 改为 `container.Resolve[broker.Broker]` |
| `broker.Init(opts...)` | 接口已移除 | 通过构造函数传选项 |
| `broker.Broker.String()` | 方法移除 | 不再可用；用类型断言判断实现 |
| `broker.Broker.Address()` | 方法移除 | 业务方应在创建时记录 addr |
| `adapter/broker/kafka` | 包路径变更 | 改为 `broker/kafka` |
| `adapter/broker/wrapper/{logging,metrics}` | 包路径变更 | 改为 `broker/wrappers/{logging,metrics}` |

---

## 反馈

迁移过程遇到问题请提交 issue：<https://github.com/nextmicro/next/issues>
