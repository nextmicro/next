## 需求

### 需求:Component 接口组合能力声明

Lifecycle 必须使用接口组合定义 Component 能力，禁止使用单一胖接口强制实现所有方法。Component 仅需通过实现可选接口声明所需能力。

#### 场景:仅实现 Starter 的 Component 被正确编排

- **当** 一个 Component 只实现 `Name() string` 与 `Starter` 接口，未实现 `Stopper`
- **那么** Lifecycle 启动时必须调用其 `Start(ctx)`，停止时必须跳过该 Component 而不报错

#### 场景:实现多个能力接口

- **当** 一个 Component 同时实现 `Initializer/Starter/Stopper/Watcher/HealthCheck`
- **那么** Lifecycle 必须依次在 Init/Start/Watch/Stop/Health 阶段调用对应方法

### 需求:基于类型反射的依赖声明

Component 必须通过 `DependsOn` 接口的 `Deps() []reflect.Type` 方法声明依赖，禁止使用字符串名称作为依赖标识。

#### 场景:依赖声明驱动启动顺序

- **当** Component A 声明 `Deps() = []reflect.Type{TypeOf[*LoggerComponent]()}`，Component B 无依赖
- **那么** Lifecycle 必须先启动 B（实际上 A 也依赖 B 时 B 先启动），无声明依赖的 A 与 B 同层并行启动

#### 场景:循环依赖在启动前检测

- **当** Component A 依赖 B 且 B 依赖 A
- **那么** `Lifecycle.Start(ctx)` 必须在执行任何 Component 之前返回循环依赖错误，错误信息包含环中所有 Component 名称

### 需求:拓扑排序与同层并行启动

Lifecycle 必须基于依赖图进行拓扑排序，将无依赖关系的 Component 划分到同一层，同层 Component 并行启动以缩短启动时延。

#### 场景:同层并行执行

- **当** Component A 与 B 均依赖 Logger，且彼此无依赖
- **那么** Logger 启动完成后，A 与 B 必须并行执行 `Start(ctx)`，而非串行

#### 场景:跨层严格顺序

- **当** Component A 依赖 B
- **那么** B 的 `Start(ctx)` 必须在 A 的 `Start(ctx)` 开始之前完成（包括其异步资源就绪）

### 需求:启动失败时反向回滚

任意 Component 的 `Start(ctx)` 返回错误时，Lifecycle 必须按相反启动顺序停止已成功启动的所有 Component，禁止留下半启动状态。

#### 场景:中间组件启动失败

- **当** 启动顺序为 A→B→C，B 启动成功，C 启动失败
- **那么** Lifecycle 必须先调用 C 的 Stop（若已实现 Stopper 且部分启动），再调用 B 的 Stop，最后调用 A 的 Stop，并返回 C 的错误

#### 场景:回滚期间发生新错误时聚合返回

- **当** 启动失败后回滚过程中某 Component 的 `Stop(ctx)` 也返回错误
- **那么** Lifecycle 必须聚合所有错误返回（启动失败错误 + 回滚错误），禁止吞掉任意一个

### 需求:反向停止顺序与超时控制

Lifecycle 停止 Component 时必须严格反向于启动顺序，每个 Component 的 `Stop(ctx)` 必须在传入的 context 内完成，超时后强制下一个。

#### 场景:正常反向停止

- **当** 启动顺序为 A→B→C
- **那么** `Lifecycle.Stop(ctx)` 必须按 C→B→A 顺序调用 Stop

#### 场景:单个 Component 停止超时

- **当** B 的 `Stop(ctx)` 超过 ctx 超时仍未返回
- **那么** Lifecycle 必须在超时后记录错误并继续停止 A，禁止无限等待

### 需求:健康检查聚合

Lifecycle 必须提供 `Health(ctx)` 方法聚合所有实现 `HealthCheck` 接口的 Component 的健康状态。

#### 场景:全部健康时返回 OK

- **当** 所有实现 HealthCheck 的 Component `Health(ctx)` 都返回 nil
- **那么** `Lifecycle.Health(ctx)` 必须返回总状态 OK 及每个组件的健康详情

#### 场景:任一 Component 不健康时整体降级

- **当** 任意 HealthCheck 返回错误
- **那么** `Lifecycle.Health(ctx)` 必须返回总状态 DEGRADED，并在详情中列出失败 Component 与错误信息

### 需求:与 kratos.App 的执行顺序契约

Lifecycle 必须在 kratos.App 启动 Server 之前完成所有 Component 的 Start，并在 kratos.App 停止 Server 之后才执行 Component 的 Stop。

#### 场景:基础设施先于 Server 就绪

- **当** next.App.Run() 被调用
- **那么** Logger/Tracing/Registry/Broker 等基础设施 Component 必须全部 Start 完成后，kratos.App 才能开始 Listen 端口

#### 场景:Server 关闭后才释放基础设施

- **当** 收到退出信号
- **那么** kratos.App 必须先完成 Server 优雅退出，Lifecycle 才能开始反向 Stop 基础设施 Component
