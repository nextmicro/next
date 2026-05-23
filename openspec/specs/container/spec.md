## 需求

### 需求:类型安全的实例注册

Container 必须支持以 Go 类型为键注册任意实例，禁止使用字符串作为唯一标识。注册接口必须利用 Go 泛型在编译期约束类型一致性。

#### 场景:按具体类型注册并解析

- **当** 调用方使用 `container.Register[*UserService](c, svc)` 注册一个 `*UserService` 实例
- **那么** 调用 `container.Resolve[*UserService](c)` 必须返回同一个实例且不发生类型断言错误

#### 场景:按接口类型注册并解析

- **当** 调用方使用 `container.RegisterAs[broker.Broker](c, &kafkaBroker{})` 显式以接口类型注册
- **那么** 调用 `container.Resolve[broker.Broker](c)` 必须返回该实现，且不需要二次类型转换

#### 场景:重复注册同一类型

- **当** 同一具体类型被注册两次且未声明覆盖
- **那么** 第二次注册必须返回错误，禁止静默覆盖

### 需求:命名实例注册

Container 必须支持同一类型的多实例命名注册，用于多数据源、多 broker、多 registry 等场景。

#### 场景:命名注册不同实例

- **当** 调用方依次执行 `RegisterNamed[*sql.DB](c, "primary", db1)` 与 `RegisterNamed[*sql.DB](c, "replica", db2)`
- **那么** `ResolveNamed[*sql.DB](c, "primary")` 必须返回 `db1`，`ResolveNamed[*sql.DB](c, "replica")` 必须返回 `db2`

#### 场景:命名实例与默认实例隔离

- **当** 同一类型同时存在默认注册和命名注册
- **那么** `Resolve[T]` 必须返回默认实例，`ResolveNamed[T]` 必须返回命名实例，两者互不影响

### 需求:解析失败的可观察错误

Container 必须为未注册类型的解析返回明确错误，禁止 panic（除 `MustResolve` 等显式声明 panic 的便捷 API 外）。

#### 场景:解析未注册类型

- **当** 调用方对未注册的类型调用 `Resolve[T](c)`
- **那么** 必须返回包含类型名称的错误（例如 `container: type *foo.Bar not registered`），且第二返回值为该类型的零值

#### 场景:MustResolve 在未注册时 panic

- **当** 调用方对未注册的类型调用 `MustResolve[T](c)`
- **那么** 必须 panic，且 panic 信息包含类型名称

### 需求:启动期一次性注册保证

Container 必须在启动期完成所有注册，运行期热路径（Resolve）禁止包含写操作。读操作必须线程安全且无锁等待开销。

#### 场景:并发解析无竞争

- **当** 启动完成后，1000 个 goroutine 并发调用 `Resolve[T](c)` 解析相同类型
- **那么** 所有调用必须成功返回相同实例，且无数据竞争（`go test -race` 通过）

#### 场景:启动后写入触发警告

- **当** 启动完成后再次调用 `Register[T](c, v)`（即 `c.Sealed()` 之后）
- **那么** 必须返回错误（建议禁止运行期注册），调用方需在启动期注册完毕

### 需求:启动前依赖完整性校验

Container 必须提供 `Validate()` 方法，由 Lifecycle 在启动前调用，校验所有声明依赖的 Component 都能从 Container 中解析。

#### 场景:依赖完整时校验通过

- **当** 所有 Component 声明的 `Deps()` 中类型均已在 Container 注册
- **那么** `Validate()` 返回 nil

#### 场景:依赖缺失时校验失败

- **当** 某 Component 声明依赖 `broker.Broker` 但 Container 中未注册该类型
- **那么** `Validate()` 必须返回错误，错误信息必须包含缺失类型与依赖的 Component 名称
