## 1. Phase 1 — Kernel 内核基础（container + lifecycle）

- [x] 1.1 创建 `container/` 包，定义 `Container` 结构体（`types map[reflect.Type]any`、`named map[reflect.Type]map[string]any`、`sync.RWMutex`）
- [x] 1.2 实现 `Register[T any](c *Container, v T) error`、`RegisterAs[I any](c *Container, v I) error`、`RegisterNamed[T any](c *Container, name string, v T) error`
- [x] 1.3 实现 `Resolve[T any](c *Container) (T, error)`、`MustResolve[T any](c *Container) T`、`ResolveNamed[T any](c *Container, name string) (T, error)`
- [x] 1.4 实现 `Container.Seal()` 与 `Container.Sealed()`，启动后写入返回错误
- [x] 1.5 实现 `Container.Validate(deps map[string][]reflect.Type) error`，校验所有声明依赖可解析
- [x] 1.6 为 `container` 包补齐单测，覆盖：重复注册、命名隔离、并发解析、未注册错误、Seal 后写入、Validate
- [x] 1.7 创建 `lifecycle/` 包，定义 `Component`、`Initializer`、`Starter`、`Stopper`、`Watcher`、`HealthCheck`、`DependsOn` 接口
- [x] 1.8 实现 `lifecycle/graph.go`：DAG 构建、循环依赖检测、拓扑分层
- [x] 1.9 实现 `Lifecycle` 主结构与 `Add(c Component)`、`Start(ctx)`、`Stop(ctx)`、`Health(ctx)`、`Watch(ctx)` 方法
- [x] 1.10 实现同层并行启动（`golang.org/x/sync/errgroup`）
- [x] 1.11 实现启动失败时的反向回滚与多错误聚合（`errors.Join`）
- [x] 1.12 实现反向停止顺序与超时控制（每个 Stop 使用 ctx 派生的 timeout）
- [x] 1.13 实现 `Health(ctx)` 聚合，返回 OK / DEGRADED + 详情
- [x] 1.14 为 `lifecycle` 包补齐单测，覆盖：循环依赖、同层并行、回滚、停止超时、Health 聚合
- [x] 1.15 撰写 `container` 与 `lifecycle` 包的 Go doc 与使用示例

## 2. Phase 2 — Broker 重构（核心差异化能力）

- [x] 2.1 重写 `broker/broker.go`：定义最小化的 `Broker` 接口（`Connect/Disconnect/Publish/Subscribe`）
- [x] 2.2 重写 `broker/message.go`、`broker/event.go`、`broker/subscription.go`：定义 `Message`、`Header`、`Event`、`Subscription` 类型
- [x] 2.3 重写 `broker/options.go`：定义 `PublishOption`、`SubscribeOption` 及内置选项（Header、ContentType 等）
- [x] 2.4 保留并简化 `broker/wrapper.go`：保留 `Wrapper` 装饰器机制
- [x] 2.5 创建 `broker/typed/` 子包，定义 `Topic[T]`、`Codec[T]`、`Publisher[T]`、`Subscriber[T]`、`Handler[T]`、`Event[T]`
- [x] 2.6 实现 `typed.NewPublisher[T]` 与 `typed.NewSubscriber[T]`，封装底层 Broker 的字节流与 Codec
- [x] 2.7 实现 `typed.ProtoCodec[T]` 与 `typed.JSONCodec[T]` 两种内置 Codec
- [x] 2.8 重写 `broker/memory.go`：适配新的 `Broker` 接口，作为参考实现
- [x] 2.9 将 `adapter/broker/kafka/` 迁移到 `broker/kafka/` 并适配新接口
- [x] 2.10 将 `adapter/broker/wrapper/{logging,metrics}` 适配为新 `Wrapper` 形式
- [x] 2.11 为 `broker`（含 typed/memory/kafka）补齐单测，覆盖：发布订阅、类型安全、Codec、Wrapper、解码失败、Subscription 关闭
- [x] 2.12 删除旧 `broker/options.go` 中 `Init/Address/String` 等已废弃接口的相关代码

## 3. Phase 3 — 拷贝代码清零

- [ ] 3.1 删除 `transport/http/server.go`、`transport/http/client.go`、`transport/http/codec.go`、`transport/http/context.go`、`transport/http/resolver.go`、`transport/http/calloption.go` _(推迟到 Phase 3B：旧 Server 类型仍是 v1 业务方主入口；先与新 `transport.NewHTTPServer` 并行存在，下一轮再硬删)_
- [ ] 3.2 删除 `transport/http/binding/`、`transport/http/status/`、`transport/http/redirect.go`、`transport/http/response.go`、`transport/http/filter.go`、`transport/http/codec_go1.20.go` _(推迟到 Phase 3B)_
- [ ] 3.3 评估 `transport/http/router.go` 与 `transport/http/transport.go`：保留 router.go（next 独有的 mux 路由 DSL 与 Context 抽象），transport.go 推迟评估
- [ ] 3.4 删除 `transport/grpc/server.go`、`transport/grpc/client.go`、`transport/grpc/codec.go`、`transport/grpc/balancer.go`、`transport/grpc/interceptor.go`、`transport/grpc/transport.go` _(推迟到 Phase 3B)_
- [ ] 3.5 评估 `transport/grpc/resolver/`：与 kratos resolver 对比，若重复则删除并依赖 kratos _(推迟到 Phase 3B)_
- [ ] 3.6 删除 `internal/host/`、`internal/endpoint/`、`internal/matcher/` _(无法删除：kratos 把这些放在 internal 不公开，next 必须保留拷贝；建议改为 README 注明)_
- [ ] 3.7 评估 `internal/group/`、`internal/context/`、`internal/httputil/`：保留 next 独有的，删除与 kratos 重复的 _(推迟到 Phase 3B)_
- [x] 3.8 创建 `transport/http.go`：实现 `NewHTTPServer(cfg HTTPServerConfig) (*khttp.Server, error)`
- [x] 3.9 创建 `transport/grpc.go`：实现 `NewGRPCServer(cfg GRPCServerConfig) (*kgrpc.Server, error)`
- [x] 3.10 工厂中集成 `middleware.Registry.Build("server", cfg.GetMiddlewares())` 自动装配中间件链
- [x] 3.11 工厂中处理 network/timeout/监听地址等配置翻译；ServerOptions 透传给 kratos 支持业务覆盖
- [x] 3.12 为 `transport/http.go`、`transport/grpc.go` 补齐单测，覆盖：默认值、中间件装配、错误传播、Registry 回退、MustXxx panic

## 4. Phase 4 — Middleware Factory 化

- [x] 4.1 重写 `middleware/registry.go`：定义 `Registry` 接口与默认实现，禁止全局 `globalRegistry`
- [x] 4.2 重写 `middleware/builder.go`：实现 `Build(scope, configs)` 方法，按 scope 过滤、不修改输入
- [x] 4.3 删除旧 `BuildMiddleware` 中对 `_ms[i].Name` 的 in-place 修改逻辑
- [ ] 4.4 创建 `middleware/factories/tracing/`，删除原有 `middleware/tracing/{tracing.go,span.go,statshandler.go,metadata.go,option.go}`，仅保留 ~50 行的 factory 适配 `kratos/middleware/tracing` _(推迟到下一轮：先验证 Registry 切换稳定，再做拷贝代码替换)_
- [ ] 4.5 同样处理 `middleware/factories/recovery/`（适配 `kratos/middleware/recovery`） _(推迟)_
- [ ] 4.6 同样处理 `middleware/factories/metrics/`（适配 `kratos/middleware/metrics`） _(推迟)_
- [ ] 4.7 同样处理 `middleware/factories/metadata/`（适配 `kratos/middleware/metadata`） _(推迟)_
- [ ] 4.8 同样处理 `middleware/factories/circuitbreaker/`（适配 `kratos/middleware/circuitbreaker`） _(推迟)_
- [ ] 4.9 同样处理 `middleware/factories/bbr/`（适配 `kratos contrib` 或自有实现） _(推迟)_
- [x] 4.10 保留 `middleware/logging/`：仅适配为新 `Registry` 注册接口，业务逻辑保持
- [x] 4.11 为 `middleware` 包补齐单测，覆盖：Registry 注册、Build 顺序、scope 过滤、未注册错误、输入不可变性
- [x] 4.12 验证旧测试 `middleware/logging/logging_test.go` 通过

## 5. Phase 5 — App 装饰器与 Options 重设计

- [x] 5.1 重写 `app.go`：定义新 `App` 结构（嵌入 `*kratos.App`、持有 `*Container`、`*Lifecycle`、`MiddlewareRegistry`）
- [x] 5.2 重写 `options.go`：新内部 options 结构，新增 WithComponents/WithContainer/WithMiddlewareRegistry；保留 Loader 兼容 Option
- [x] 5.3 实现 `next.New(opts...)`：构建 Container → 注册 Self/Registry → 添加 Component → 执行 WithContainer → 构造 kratos.App
- [x] 5.4 实现 `App.Run()`：先 `Lifecycle.Start(ctx)`、再 `App.App.Run()`、退出时 `Lifecycle.Stop(ctx)`，组合多错
- [x] 5.5 实现 Container 注入工厂：`WithContainer(func(*Container))` Option
- [x] 5.6 通过 `components/builtin.go` 包装现有 4 个 loader 为 lifecycle.Component（默认接入）
- [ ] 5.7 删除 `runtime/` 包及其旧 `Loader` 接口 _(推迟到 Phase 7：当所有 loader 已被 components 包独立实现替代后)_
- [ ] 5.8 删除 `broker.DefaultBroker`、`registry.DefaultRegistry` 等全局变量 _(broker 包侧已完成；registry.DefaultRegistry 仍被 runtime/loader/registry 与 naxt.go 默认 Registrar 使用，推迟到 Phase 7)_
- [x] 5.9 重命名 `naxt_test.go` 为 `app_test.go`，使用新 API（Container/Lifecycle/MiddlewareRegistry 访问器、WithContainer/WithComponents）
- [x] 5.10 为 `app.go` 补齐集成测试，验证：默认值、Container 注入、middleware Registry 注入、Lifecycle 启动/停止、Component 调用次数

## 6. Phase 6 — Config 类型化与文档

- [x] 6.1 在 `next.New` 中把 `*configv1.Next` 与 `kconfig.Config` 注册到 Container（保留 `nextConfig` 全局变量作 deprecated 兼容入口，Phase 7 清理）
- [x] 6.2 增加 `config/typed.go`：实现 `Bind[T any]` / `MustBind[T]` / `BindWatch[T any]`，含 7 个单测
- [x] 6.3 Nacos 配置源保持在 `adapter/config/nacos`，不动
- [ ] 6.4 重写 `config/helper.go`：移除对 `nextConfig` 全局变量的依赖 _(推迟到 Phase 7：大量 ApplicationConfig() 调用方仍未迁移)_
- [x] 6.5 重写 `README.md`：明确 next 是 kratos 的"装饰扩展层"，新增 5 大设计亮点 + 完整示例
- [x] 6.6 新增 `examples/quickstart/`（HTTP + Component + Container DI 完整示例）；broker / component 示例推迟到下一轮
- [x] 6.7 撰写迁移文档 `docs/MIGRATION.md`：v1 → v2 概念对照表（10 个迁移点）
- [ ] 6.8 更新所有 Go 文件的 package 注释，与 README 风格一致 _(已对 Phase 1-6 新增/重写文件完成；旧文件统一在 Phase 7 一次性扫描)_

## 7. 验证与发布

- [x] 7.1 全仓库执行 `go vet ./...` 与 `go build ./...` 通过 _(全部 0 错误)_
- [x] 7.2 全仓库执行 `go test -race ./...` —— **33/33 包全部通过**（顺手修复 `pkg/threading` 与 `config` 中 v1 遗留 race：threading 测试移除全局 logger 替换；config 测试合并为单测共享 cfg 实例避开 kratos.config Close 后 watcher goroutine 仍打日志的上游 race）
- [x] 7.3 代码量统计：v2 核心包（container/lifecycle/middleware/broker/config/components/transport + 主包）= **9682 行**（不含 testdata/api 生成代码、不含 pkg/* 工具）。完整仓库非测试代码 12939 行。目标 ≤ 7000 行的范围适用于"删除 transport 拷贝代码后"，本轮 Phase 3B 推迟，故该指标推迟评估。
- [x] 7.4 grep `DefaultBroker / DefaultRegistry / DefaultLogger / globalRegistry / nextConfig`：✓ `broker.DefaultBroker` 已清零；其余保留为 deprecated 兼容入口（全部带 `// Deprecated:` 注释，业务可通过 Container 替代）
- [x] 7.5 grep `_ms\[i\]\.Name =` 字符串 mutation：✓ **0 处**（全清）
- [ ] 7.6 检查 `transport/http/`、`transport/grpc/` 与 kratos 上游相似度：**仍 80%+ 相似**（Phase 3B 待办；本轮已通过新工厂 `transport.NewHTTPServer / NewGRPCServer` 提供 v2 入口并行存在）
- [x] 7.7 文档评审：✓ `README.md`（5 大设计亮点 + 完整示例）、`docs/MIGRATION.md`（10 个迁移点）、`examples/quickstart/`（HTTP + Component + DI）
- [ ] 7.8 打 tag `v2.0.0-rc.1` _(用户操作)_
- [ ] 7.9 rc 反馈处理，打 tag `v2.0.0` _(用户操作)_
- [ ] 7.10 GitHub Release Notes 明确 break change _(用户操作)_
