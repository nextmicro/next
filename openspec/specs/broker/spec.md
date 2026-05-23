## 需求

### 需求:底层 Broker 接口的最小化

底层 `broker.Broker` 接口必须仅暴露连接管理与字节级消息传输，禁止包含 `Init/Options/Address/String` 等暴露内部状态的方法。

#### 场景:接口仅包含核心方法

- **当** 实现一个新 Broker（例如 NATS）
- **那么** 仅需实现 `Connect / Disconnect / Publish / Subscribe` 四个方法，不需要实现 `Address() string` 等元信息方法

#### 场景:配置通过构造函数注入

- **当** 创建 Broker 实例
- **那么** 必须通过构造函数（如 `kafka.New(opts ...Option)`）注入配置，禁止暴露 `Init(opts...)` 二次初始化方法

### 需求:泛型类型安全的 Topic 定义

Broker 必须提供 `typed.Topic[T any]` 类型作为业务层一等公民，将主题名称与负载类型在编译期绑定。

#### 场景:声明 Topic 后获得类型安全引用

- **当** 业务声明 `var OrderCreated = typed.Topic[*pb.Order]{Name: "orders.created", Codec: ProtoCodec[*pb.Order]{}}`
- **那么** 后续 `NewPublisher(broker, OrderCreated)` 返回的 Publisher 必须只接受 `*pb.Order` 类型的消息，传入其他类型在编译期失败

#### 场景:同一 Topic 重复定义不污染状态

- **当** 同一个 Topic 名称在多处声明
- **那么** 编译通过且每个声明独立，禁止内部全局表导致冲突

### 需求:类型安全的 Publisher 与 Subscriber

Broker 必须提供 `typed.Publisher[T]` 与 `typed.Subscriber[T]` 接口，分离发布与订阅能力，避免单一接口耦合。

#### 场景:发布无需手动 marshal

- **当** 业务代码调用 `pub.Publish(ctx, &pb.Order{ID: "123"})`
- **那么** Publisher 必须自动使用 Topic 中声明的 Codec 完成序列化并调用底层 Broker，业务代码全程不接触 `[]byte`

#### 场景:订阅 Handler 直接得到类型化 payload

- **当** 业务调用 `sub.Subscribe(ctx, func(ctx, e Event[*pb.Order]) error { o := e.Payload(); ... })`
- **那么** `e.Payload()` 必须直接返回 `*pb.Order`，禁止业务代码处理 `[]byte` 解码

#### 场景:解码失败的错误处理

- **当** 收到的消息无法通过 Codec 解码为目标类型
- **那么** Subscriber 必须将原始消息 Nack（或转入死信处理），调用 Handler 时返回类型化的解码错误，禁止 panic

### 需求:Subscription 生命周期与底层连接分离

订阅返回的 `Subscription` 关闭时必须仅终止该订阅，禁止影响同一底层 Broker 的其他订阅或发布。

#### 场景:取消订阅不影响其他订阅

- **当** 业务对同一 Broker 创建了 3 个不同 topic 的订阅，关闭其中一个
- **那么** 剩余 2 个订阅必须继续正常接收消息

#### 场景:取消订阅不关闭底层连接

- **当** 业务关闭一个订阅
- **那么** 底层 Broker 的 `Connect` 状态必须保持，后续 `Publish` 必须仍然可用

### 需求:Codec 抽象与可扩展性

Broker 必须定义 `Codec[T]` 接口，允许业务为不同 Topic 使用不同编码（Proto/JSON/MsgPack 等）。框架必须内置 Proto 与 JSON 两种实现。

#### 场景:使用 Proto Codec

- **当** Topic 声明使用 `typed.ProtoCodec[*pb.Order]{}`
- **那么** Publisher 必须调用 `proto.Marshal` 序列化，Subscriber 必须调用 `proto.Unmarshal` 反序列化

#### 场景:自定义 Codec 接入

- **当** 业务实现 `Codec[T]` 接口并在 Topic 中使用
- **那么** Publisher 与 Subscriber 必须使用该自定义 Codec，框架不得强制使用内置实现

### 需求:消息头与上下文传递

`typed.Event[T]` 必须暴露消息头（Header）以支持 trace propagation、metadata 透传等横切关注点。

#### 场景:发布消息时传递 Header

- **当** 调用 `pub.Publish(ctx, msg, WithHeader("trace-id", "abc"))`
- **那么** 订阅端 `e.Header().Get("trace-id")` 必须返回 `"abc"`

#### 场景:Header 与 Payload 编码分离

- **当** 消息 Payload 使用 ProtoBuf 编码
- **那么** Header 必须独立于 Payload 编码，使用底层 broker 的原生 header 机制（kafka 的 record headers / memory broker 的 map）

### 需求:Wrapper 模式保留

底层 `Broker` 必须支持 `Wrapper func(Broker) Broker` 装饰器，允许在不修改实现的前提下叠加 logging/metrics/tracing 横切能力。Wrapper 必须作用于底层 `[]byte` 层，对上层 `Publisher[T]/Subscriber[T]` 透明。

#### 场景:叠加 logging wrapper

- **当** 业务通过 `broker.Wrap(LoggingWrapper(), MetricsWrapper())` 构造 Broker
- **那么** 任意 `Publish/Subscribe` 调用必须经过 wrapper 链，记录日志与指标，不影响泛型层使用

#### 场景:Wrapper 不影响类型安全

- **当** Wrapper 在底层操作 `*Message{[]byte}`
- **那么** 上层 `Publisher[*pb.Order]` 仍必须保证编译期类型约束，wrapper 不引入类型擦除

### 需求:Container 集成

Broker 必须通过 Container 注入，禁止暴露 `DefaultBroker` 全局变量供业务直接访问。

#### 场景:通过 Container 解析 Broker

- **当** 业务需要发布消息
- **那么** 必须通过 `container.Resolve[broker.Broker](c)` 获取实例，框架不得提供 `broker.DefaultBroker` 包级变量
