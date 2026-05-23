## 为什么

当前 next 框架本质是 go-kratos 的"打补丁副本"：约 3000 行代码（`transport/http`、`transport/grpc`、`internal/host`、`internal/endpoint`、`internal/matcher`、各类 middleware 实现）与 kratos 上游源码相似度达 80%-89%。这导致：

1. **kratos 升级成本高**：每次跟版需要手动 merge 拷贝代码
2. **拷贝代码失去官方维护**：kratos 的 bug fix 不会自动获益
3. **类型不安全**：Broker 接口使用 `*Message{Body []byte}` + 手动 marshal，编译期零保障
4. **全局变量泛滥**：`DefaultBroker`、`DefaultRegistry`、`DefaultLogger`、`nextConfig` 等多个单例，无法多实例、测试难、初始化顺序耦合
5. **Loader 接口失真**：`Initialized()` 是 mutable 状态、顺序硬编码、`Watch()` 多为空实现、无依赖声明
6. **Middleware 字符串魔法**：全局 `map<string,Factory>` + 入参 mutation（`_ms[i].Name = kind + "." + _ms[i].Name`）

是时候把 next 从"kratos 的副本"重新定位为"kratos 的装饰扩展层"。

## 变更内容

**BREAKING** 这是 v2 全新 API，不向后兼容、不保留 v1 分支。

### 删除（依赖 kratos 原生实现）
- **BREAKING** 删除 `transport/http/server.go`、`transport/http/client.go`、`transport/http/codec.go`、`transport/http/context.go`、`transport/http/resolver.go`、`transport/http/binding/*`（拷贝代码）
- **BREAKING** 删除 `transport/grpc/server.go`、`transport/grpc/client.go`、`transport/grpc/balancer.go`、`transport/grpc/codec.go`、`transport/grpc/interceptor.go`、`transport/grpc/transport.go`（拷贝代码）
- **BREAKING** 删除 `internal/host`、`internal/endpoint`、`internal/matcher`（kratos 已提供）
- **BREAKING** 删除 `middleware/{tracing,recovery,metrics,metadata,bbr,circuitbreaker}` 的内部实现，仅保留 Factory 适配器

### 新增（next 独有价值）
- 新增 `container` 包：类型安全的泛型 DI 容器，消除 `Default*` 全局变量
- 新增 `lifecycle` 包：基于拓扑排序的 Component 生命周期编排，替代 `runtime/loader`
- 新增 `broker/typed` 包：泛型类型安全消息层，编译期约束 topic 与 payload 类型

### 重构
- **BREAKING** 重写 `broker`：底层 `[]byte` 抽象 + 上层泛型 `Publisher[T] / Subscriber[T]`
- **BREAKING** 重写 `middleware/registry`：移除字符串 mutation，改为不可变 Builder
- **BREAKING** 重写 `transport`：仅提供 `NewHTTPServer(container, config) → *kratos/http.Server` 等构造工厂
- **BREAKING** 重写 `next.App`：从"包裹 kratos.App"改为"装饰 kratos.App"，持有 Container + Lifecycle

### 保留
- `adapter/{config/nacos, logger/{kratos,nacos,log}}`、`pkg/*`、`api/*` 保持稳定（小幅适配）

## 功能 (Capabilities)

### 新增功能

- `container`: 基于 Go 1.22 泛型的类型安全 DI 容器。提供 `Register[T] / Resolve[T] / RegisterNamed[T] / ResolveNamed[T]`，使用 `reflect.Type` 作为 key，启动期构建类型索引以避免热路径反射开销。
- `lifecycle`: Component 生命周期编排器。Component 通过接口组合声明能力（Initializer/Starter/Stopper/Watcher/HealthCheck），通过 `Deps() []reflect.Type` 声明依赖；启动时拓扑排序并分层并行启动，失败时反向回滚，停止时反向顺序。
- `broker`: 重写后的消息抽象。底层 `Broker` 处理 `[]byte` 保持插件化；上层 `typed.Topic[T]`、`typed.Publisher[T]`、`typed.Subscriber[T]`、`typed.Handler[T]`、`typed.Event[T]` 提供编译期类型安全；保留 `memory`（参考实现）与 `kafka`（生产）两个实现。
- `middleware`: 重写后的中间件装配。移除字符串 mutation 和全局 map 副作用，提供 `Registry.Register(name, factory)` 和 `Pipeline.Build(scope, configs)` 两个清晰接口；各 factory（tracing/recovery/metrics/metadata/cb/bbr）改为 kratos 原生中间件的配置适配器，零代码拷贝。
- `transport`: Server 构造工厂。提供 `NewHTTPServer(c *Container, cfg *configv1.Server) (*khttp.Server, error)` 和 `NewGRPCServer(...)`，把 next 配置翻译为 kratos `ServerOption`，不重写任何 kratos 类型。

### 修改功能

- 无（v2 不保留 v1 spec，全部为新增）

## 影响

### 代码量
- 当前 ~15,800 行（不含 test/proto），目标 ~5,000 行（净减约 68%）
- 拷贝代码 ~3,000 行清零

### 依赖
- 升级 `kratos` 到 `v2.7.3+`（已是当前版本，无需变更）
- 新增 Go 1.22 泛型使用（项目已 `go 1.22.7`）

### 用户影响
- next.bff.layout 等下游使用方需要完整迁移到 v2，本次不提供 v1→v2 自动迁移工具
- 不维护 v1 分支，v1 用户停留在最后一个 tag

### 测试
- 现有测试大量基于拷贝代码，需重写
- 新核心抽象（container/lifecycle/broker[T]）需达 ≥80% 单测覆盖

### 文档
- README 需要重写，从"基于 kratos 封装"明确为"kratos 的装饰扩展层"
- 新增 `examples/` 目录展示最佳实践
