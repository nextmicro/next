# next

> next 是基于 [go-kratos](https://github.com/go-kratos/kratos) 的**装饰扩展层**。

[![Go Reference](https://pkg.go.dev/badge/github.com/nextmicro/next.svg)](https://pkg.go.dev/github.com/nextmicro/next)

next 不是 kratos 的替代品，也不再复制 kratos 的源码。它在 kratos 之上提供：

- **类型安全的 DI 容器**（`container.Container`）：消除全局变量，编译期检查依赖
- **基于拓扑排序的 Component 生命周期编排**（`lifecycle.Lifecycle`）：自动并行启动、反向回滚、统一健康检查
- **类型化 Broker**（`broker/typed`）：基于 Go 泛型的 `Publisher[T]` / `Subscriber[T]`，业务无需手写 marshal
- **可实例化的 Middleware Registry**（`middleware.Registry`）：替代字符串魔法与全局注册表
- **类型化配置绑定**（`config.Bind[T]`）：泛型 API 一行绑定配置段

---

## 快速开始

```bash
go get github.com/nextmicro/next
```

```go
package main

import (
    "context"

    "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/nextmicro/next"
    "github.com/nextmicro/next/components"
    "github.com/nextmicro/next/container"
    "github.com/nextmicro/next/transport"

    _ "github.com/nextmicro/next/middleware/all" // 启用全部内置 middleware factory
)

type GreeterService struct{ name string }

func (s *GreeterService) Hello() string { return "hello " + s.name }

func main() {
    // 1. 创建 HTTP server（自动装配配置中的 middleware）
    srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{
        ServerOptions: []http.ServerOption{http.Address(":8080")},
    })
    if err != nil { panic(err) }

    // 2. 构造 App
    app, err := next.New(
        next.Name("greeter"),
        next.Version("v1.0.0"),
        next.Server(srv),
        next.WithComponents(
            components.DefaultLogger(),
            components.DefaultTracing(),
        ),
        next.WithContainer(func(c *container.Container) {
            _ = container.Register(c, &GreeterService{name: "next"})
        }),
    )
    if err != nil { panic(err) }

    if err := app.Run(); err != nil { panic(err) }
}
```

---

## 核心包

| 包 | 作用 |
|----|------|
| `next` | App 入口，组合 kratos.App + Container + Lifecycle |
| `container` | 类型安全的 DI 容器（基于 generics） |
| `lifecycle` | 基于拓扑排序的 Component 编排器 |
| `components` | 默认 Component 集合（logger / tracing / registry / broker） |
| `broker` + `broker/typed` | 异步消息抽象 + 类型安全包装 |
| `middleware` | 可实例化的 Middleware Registry + Builder |
| `transport` | HTTP / gRPC server 工厂（薄装饰 kratos 原版） |
| `transport/httpx` | next Router DSL（`Context` / `MiddlewareFunc` / `GET`/`POST`/`Group`） |
| `config` | 泛型类型化配置绑定 `Bind[T]` / `BindWatch[T]` |

---

## 设计亮点

### 1. 类型安全的 DI 容器

```go
c := container.New()
container.Register(c, db)                              // 按具体类型注册
container.RegisterAs[contracts.UserRepo](c, sqlrepo)   // 按接口注册
container.RegisterNamed(c, "primary", primaryDB)       // 按名称区分多实例

db, err := container.Resolve[*sql.DB](c)
repo := container.MustResolve[contracts.UserRepo](c)
```

### 2. Component 生命周期编排

```go
type CacheComponent struct{ db *sql.DB }
func (*CacheComponent) Name() string { return "cache" }
func (c *CacheComponent) Start(ctx context.Context) error { return c.warmup(ctx) }
func (c *CacheComponent) Deps() []reflect.Type {
    return []reflect.Type{container.TypeOf[*sql.DB]()}  // 声明依赖
}

// 编排器自动按拓扑分层并行启动，反向回滚
lc := lifecycle.New()
lc.Add(&DBComponent{}).Add(&CacheComponent{})
_ = lc.Start(ctx)
```

### 3. 类型安全 Broker

```go
var OrderCreated = typed.Topic[*pb.Order]{
    Name:  "orders.created",
    Codec: typed.ProtoCodec[*pb.Order]{},
}

pub := typed.NewPublisher(b, OrderCreated)
_ = pub.Publish(ctx, &pb.Order{ID: "123"})

sub := typed.NewSubscriber(b, OrderCreated)
_, _ = sub.Subscribe(ctx, func(ctx context.Context, e typed.Event[*pb.Order]) error {
    order := e.Payload()  // 编译期类型确定
    return processOrder(order)
})
```

### 4. 可实例化的 Middleware Registry

```go
reg := middleware.NewRegistry()                                  // 不再依赖全局单例
reg.Register("server.tracing", tracingFactory)
chain, _ := reg.Build("server", configFromYAML.Middlewares)     // 不修改入参
```

### 5. 类型化配置绑定

```go
type DB struct {
    Datasource string `json:"datasource"`
    Pool       int    `json:"pool"`
}
db, err := config.Bind[DB](cfg, "database")

stop, _ := config.BindWatch[DB](cfg, "database", func(newDB *DB) {
    pool.SetDSN(newDB.Datasource)
})
defer stop()
```

---

## 迁移自 v1

next v2 是 **break change**。v1 业务方升级路径见 [docs/MIGRATION.md](./docs/MIGRATION.md)。

主要改动：
- `next.New` 内部完全切换到 Container + Lifecycle（API 保持向后兼容）
- `*next.Next` → `*next.App`（保留类型别名 `type Next = App`）
- `broker.DefaultBroker` 全局变量 → 通过 `container.Resolve[broker.Broker]` 获取
- 7 个 middleware 子包不再自动 blank-import → 业务方按需 `_ "github.com/nextmicro/next/middleware/all"`

---

## 示例

- [`examples/quickstart/`](./examples/quickstart) — HTTP + gRPC 基础应用

---

## 许可证

MIT
