## 上下文

next 框架现状是一个"fork 式"的 kratos 包装层：

```
当前依赖关系（问题）：
  next/transport/http/server.go  ≈ 89% 拷贝自 kratos/transport/http/server.go
  next/transport/grpc/server.go  ≈ 拷贝自 kratos/transport/grpc/server.go
  next/middleware/{tracing,recovery,metrics,...}  ≈ 拷贝自 kratos 中间件
  next/internal/{host,endpoint,matcher}  ≈ 完全拷贝自 kratos/internal/*
```

总计约 3000 行拷贝代码，约占代码量 20%。这种模式导致 kratos 升级、bug fix 都无法自动受益，并产生独立维护负担。

同时 next 提供了 kratos 没有的能力：
- 消息中间件抽象（Broker / Kafka 集成）
- 配置驱动的中间件装配（yaml → middleware chain）
- 多源配置（local + env + nacos）
- 完整生命周期（logger/tracing/registry/broker 统一管理）
- 协程工具、信号处理、指标转换等 pkg 工具

**约束**：
- 项目已使用 Go 1.22.7 + kratos v2.7.3
- 现有用户主要是 next.bff.layout 等内部业务项目
- 已接受 v2 完全 break，不需要 v1 兼容、不需要迁移工具

**利益相关者**：
- 框架维护者：减少 kratos 同步成本，聚焦差异化价值
- 业务开发者：编译期类型安全、统一的依赖注入、明确的生命周期

## 目标 / 非目标

**目标：**

1. **彻底删除拷贝代码**：transport/http、transport/grpc、internal/host、internal/endpoint、internal/matcher 改为依赖 kratos 原生
2. **建立三大核心抽象**：Container（DI）、Lifecycle（编排）、Broker[T]（类型安全消息）
3. **消除全局变量**：DefaultBroker、DefaultRegistry、DefaultLogger、nextConfig 全部下沉到 Container
4. **中间件去字符串魔法**：移除全局 map、移除入参 mutation，改为不可变 Builder
5. **代码瘦身**：~15,800 行 → ~5,000 行（净减约 68%）
6. **保持 next 独有价值**：Broker 抽象、配置驱动装配、多源配置、pkg 工具集

**非目标：**

- 不替换 kratos，next 不试图成为 kratos 的竞争者
- 不引入新的传输协议（仅保留 HTTP/gRPC，通过 kratos）
- 不扩展更多 broker 实现（仅 memory + kafka）
- 不维护 v1 向后兼容
- 不为 next.bff.layout 等下游提供自动迁移工具
- 不引入新的 DI 容器框架（如 fx/wire/dig），自实现轻量级 Container
- 不重写 kratos 的服务发现，registry 仅作配置→Registrar 工厂

## 决策

### D1. next 重新定位为 kratos 的"装饰扩展层"

**选择**：next 直接 import 并使用 kratos 的所有原生类型（`*kratos.App`、`*khttp.Server`、`*kgrpc.Server`、`middleware.Middleware`），不再拷贝任何 kratos 内部代码。

**替代方案**：
- A. 继续 fork 模式（现状）→ 维护成本高，已确认放弃
- B. 完全脱离 kratos → 工作量巨大，不必要
- C. 仅做 facade（不增加新抽象）→ 失去 next 的差异化价值

**理由**：Go 生态对"小而美的扩展层"接受度高（参考 echo/gin 与 net/http 的关系），kratos 的设计本身留足了扩展空间（Option 模式、interface 解耦），不需要分叉。

### D2. 三大核心抽象的接口设计

#### D2.1 Container：泛型 DI 容器

```go
// container/container.go (示意)
type Container struct {
    types  map[reflect.Type]any
    named  map[reflect.Type]map[string]any
    mu     sync.RWMutex
}

func Register[T any](c *Container, v T) error
func Resolve[T any](c *Container) (T, error)
func MustResolve[T any](c *Container) T
func RegisterNamed[T any](c *Container, name string, v T) error
func ResolveNamed[T any](c *Container, name string) (T, error)
```

**关键决策**：
- 使用 `reflect.Type` 而非字符串作为 key → 编译期类型安全
- 使用泛型函数（package-level）而非方法 → Go 不支持泛型方法
- 启动期注册一次性完成，热路径只读，使用 `sync.RWMutex` 优化读

**替代方案**：
- A. 使用 fx/dig/wire → 引入大型框架，违背"轻量装饰层"定位
- B. 字符串 key → 失去类型安全
- C. 仅按 interface 注册 → 无法支持具体类型

#### D2.2 Lifecycle：拓扑排序的 Component 编排

```go
// lifecycle/component.go (示意)
type Component interface {
    Name() string
}

type Initializer interface { Init(ctx context.Context, c *Container) error }
type Starter      interface { Start(ctx context.Context) error }
type Stopper      interface { Stop(ctx context.Context) error }
type Watcher      interface { Watch(ctx context.Context) error }
type HealthCheck  interface { Health(ctx context.Context) error }
type DependsOn    interface { Deps() []reflect.Type }
```

**关键决策**：
- 能力通过**接口组合**，Component 按需实现（避免胖基类）
- 依赖用 `reflect.Type`（与 Container 一致）
- 启动顺序：拓扑排序 → 分层 → 同层并行
- 失败回滚：启动失败时反向 Stop 已启动组件
- 停止顺序：完全反向，超时控制，多错误聚合

**替代方案**：
- A. 现有 Loader 接口（保留 Initialized/Init/Start/Watch/Stop）→ 状态隐式、无依赖声明
- B. 单一胖接口 Component{Init+Start+Stop+Watch+Health}（kratos 风格）→ 强制实现不需要的方法
- C. 字符串依赖名 → 失去类型安全

#### D2.3 Broker[T]：底层 []byte + 上层泛型类型安全

```go
// broker/broker.go (底层)
type Broker interface {
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Publish(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error
    Subscribe(ctx context.Context, topic string, h Handler, opts ...SubscribeOption) (Subscription, error)
}

// broker/typed/typed.go (上层泛型)
type Topic[T any] struct {
    Name  string
    Codec Codec[T]
}

type Publisher[T any]  interface { Publish(ctx, T, ...PublishOption) error }
type Subscriber[T any] interface { Subscribe(ctx, Handler[T], ...SubscribeOption) (Subscription, error) }

type Handler[T any] func(context.Context, Event[T]) error
type Event[T any]   interface {
    Topic() string
    Payload() T
    Header() Header
    Ack() error
    Nack(err error) error
}

func NewPublisher[T any](b Broker, t Topic[T]) Publisher[T]
func NewSubscriber[T any](b Broker, t Topic[T]) Subscriber[T]
```

**关键决策**：
- 底层 `Broker` 仍处理 `[]byte` → 保持插件化能力（memory/kafka 实现）
- 上层 `Publisher[T]/Subscriber[T]` 在 `Broker` 之上做编解码包装
- `Topic[T]` 是一等公民 → 业务集中定义、不同 topic 可用不同 codec
- 拆分 Publisher/Subscriber → 单一职责，按需注入
- 移除 `Init/Address/String` 等冗余方法

**替代方案**：
- A. 直接做泛型 `Broker[T]` → 不同 topic 需要不同 Broker 实例，违反单一 Broker 多 topic 现实
- B. 保留 `*Message{[]byte}` 不做泛型 → 编译期无类型安全
- C. 完全运行时反射 → 性能差、错误推迟到运行期

### D3. Middleware 去字符串魔法

**选择**：
- 移除全局 `globalRegistry` 单例，改为 `Registry` 类型，由 Container 持有
- 移除 `BuildMiddleware` 中对 `_ms[i].Name` 的 in-place 修改（拷贝输入或不可变 Visit）
- `Factory` 仍以名称查找，但作用域更清晰（`server.tracing` 全限定名，无 hardcode 拼接）
- 各 factory（tracing/recovery/metrics 等）改为 kratos 中间件的配置适配器，零代码拷贝

```go
// middleware/registry.go (示意)
type Registry interface {
    Register(name string, factory Factory)
    Build(scope string, configs []*configv1.Middleware) ([]middleware.Middleware, error)
}

// middleware/factories/tracing/tracing.go (示意)
import ktracing "github.com/go-kratos/kratos/v2/middleware/tracing"

func init() { middleware.Register("server.tracing", serverFactory) }
func serverFactory(c *configv1.Middleware) (middleware.Middleware, error) {
    opts := translateConfig(c)
    return ktracing.Server(opts...), nil  // 直接用 kratos
}
```

**关键决策**：
- 各 middleware 包从 ~300 行 → ~50 行（仅配置翻译）
- next 自有约定的 middleware（如 next/middleware/logging 自定义日志字段格式）保留独立实现

### D4. Transport 改为 ServerBuilder 模式

**选择**：

```go
// transport/http.go (新)
import khttp "github.com/go-kratos/kratos/v2/transport/http"

func NewHTTPServer(c *container.Container, cfg *configv1.Server) (*khttp.Server, error) {
    opts := []khttp.ServerOption{}
    // 翻译配置...
    ms := buildMiddleware(c, cfg.GetHttp().GetMiddlewares())
    opts = append(opts, khttp.Middleware(ms...))
    return khttp.NewServer(opts...), nil
}
```

- 不再有 `next/transport/http.Server` 类型
- 直接返回 `*khttp.Server`（kratos 原生）
- 用户感知：`next.New(transport.HTTP(cfg))` 拿到的就是 kratos Server

### D5. App 装饰器模式

**选择**：

```go
// app.go (新)
type App struct {
    *kratos.App                  // 嵌入 kratos.App
    container *container.Container
    lifecycle *lifecycle.Lifecycle
    opt       Options
}

func New(opts ...Option) (*App, error) {
    app := &App{ container: container.New() }
    // 1. 应用 Options
    // 2. 注册基础设施 Component (logger/tracing/registry/broker)
    // 3. lifecycle.Start() 启动 Component
    // 4. 构造 kratos Servers
    // 5. app.App = kratos.New(...)
    return app, nil
}

func (app *App) Run() error {
    if err := app.lifecycle.Start(ctx); err != nil { ... }
    return app.App.Run()
}
```

### D6. 全局变量消除策略

| 旧全局变量 | 新位置 |
|----|----|
| `broker.DefaultBroker` | `Container.Resolve[broker.Broker]()` |
| `registry.DefaultRegistry` | `Container.Resolve[registry.Registry]()` |
| `logger.DefaultLogger` | `Container.Resolve[logger.Logger]()` |
| `nextConfig` | `Container.Resolve[*configv1.Next]()` |
| `middleware.globalRegistry` | `Container.Resolve[middleware.Registry]()` |

迁移期允许保留 `Default*` 作为 `init()` 时从 Container 获取的便捷别名，但内部逻辑一律通过 Container。

### D7. 包结构

```
next/
├── app.go              # next.App 装饰 kratos.App
├── options.go          # next.Option
├── container/          # 泛型 DI 容器
├── lifecycle/          # Component 编排
├── broker/
│   ├── broker.go       # 底层 Broker
│   ├── memory.go       # 参考实现
│   ├── kafka/          # kafka 实现（移自 adapter）
│   └── typed/          # 泛型层
├── middleware/
│   ├── registry.go     # Factory 注册
│   ├── builder.go      # 从 config 构建
│   ├── logging/        # next 自有日志
│   └── factories/      # 各 kratos 中间件适配
├── transport/
│   ├── http.go         # ServerBuilder
│   └── grpc.go         # ServerBuilder
├── registry/factory.go # Registrar 工厂
├── config/             # 多源配置（保留 + 类型化）
├── adapter/            # config/nacos, logger/* (保留)
├── pkg/                # 工具库（保留）
└── api/                # proto 定义（保留）
```

## 风险 / 权衡

**[R1] 完全 break 导致下游迁移成本高**
- → 缓解：本次明确不维护 v1，下游按自己节奏迁移；v2 文档需提供"对照表"
- → 缓解：v2 API 设计更稳定，未来不会再有这种规模的 break

**[R2] 泛型重度使用导致编译时间增加 / 错误信息复杂**
- → 缓解：Go 泛型经过 1.21+ 多版本优化，性能与可读性可接受
- → 缓解：Container 等关键接口提供非泛型 helper（如 `MustResolve`）减少调用方负担

**[R3] Container 启动期注册顺序错误导致 Resolve 失败**
- → 缓解：Lifecycle 拓扑排序保证依赖先就绪
- → 缓解：Container 提供 `Validate()` 在启动前检查所有依赖可解析

**[R4] reflect.Type 作为 key 在接口类型场景下需谨慎**
- → 缓解：明确区分"按具体类型注册"和"按接口类型注册"
- → 缓解：提供 `RegisterAs[Interface](c, impl)` helper 显式声明

**[R5] kratos middleware 适配器若 kratos 改动接口仍需跟进**
- → 缓解：相比 fork 整个实现，仅适配器层跟进成本可控（~50 行 × 6 个 middleware）
- → 缓解：固定 kratos minor version，重要升级单独测试

**[R6] Broker[T] 多 topic 共享底层连接的资源管理**
- → 缓解：底层 Broker 持有连接，typed 层无独立资源
- → 缓解：Subscription 关闭只关闭订阅、不关闭底层连接

**[R7] 一次性大规模重构风险高**
- → 缓解：分 6 个 Phase 渐进推进，每 Phase 独立可验证
- → 缓解：Phase 1 (Kernel) 与 Phase 2 (Broker) 完成后即可在分支并行验证

**[R8] adapter 层的 kafka 实现可能依赖被删除的 broker 旧接口**
- → 缓解：Phase 2 中 kafka 实现一并迁移到新 `broker.Broker` 接口
- → 缓解：保留 `adapter/broker/wrapper` 改造为新接口的 Wrapper

## 迁移计划（v1 不维护，仅供下游参考）

由于本次不保留 v1 兼容，迁移策略简化为：

1. v2 完成后打 tag `v2.0.0`，v1 留在最后一个 tag（如 `v1.7.x`）
2. 下游项目升级时：
   - 替换 `next.New(...)` 的 Options（语义大致兼容，但部分 Option 重命名）
   - 业务消息从 `broker.Publish/Subscribe` 改为 `typed.Publisher[T]/Subscriber[T]`
   - 自定义 middleware 注册从 `chain.Register` 改为 `container.Resolve[Registry].Register`
3. 文档提供"v1 概念 → v2 概念"对照表（README 章节）

## 待解决问题

- [Q1] Container 是否需要支持 lazy initialization（首次 Resolve 时构造）？倾向于否，启动期一次性注册更简单
- [Q2] Lifecycle 同层并行启动的并发度是否需要可配置？默认无限制即可，启动期不是热路径
- [Q3] Broker typed 层的 Codec 是否要内置 JSON、ProtoBuf、MsgPack？建议先内置 Proto，其他按需添加
- [Q4] `next/middleware/logging` 的自有约定是否保留？暂保留，作为 next 业务日志格式的统一入口
