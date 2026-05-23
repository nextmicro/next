## 修改需求

### 需求:直接复用 kratos transport 类型

next 必须直接使用 `kratos/transport/http.Server`、`kratos/transport/grpc.Server` 等原生类型，禁止重新定义同名类型或拷贝其内部实现。

#### 场景:返回 kratos 原生 Server 类型

- **当** 业务调用 `transport.NewHTTPServer(c, cfg)`
- **那么** 返回类型必须是 `*kratos/transport/http.Server`，业务可直接使用 kratos 文档中的所有方法

#### 场景:不存在 next/transport/http.Server 类型

- **当** 在 next 包结构中查找 transport.Server 自定义类型
- **那么** `next/transport/http` 与 `next/transport/grpc` 包**必须不导出** `Server` 结构体类型，只导出 next 独有的扩展（如 `transport/httpx` 提供的 Router DSL）
- **过渡条款**：为减小业务方升级压力，在 `v2.1.0` 发布周期内 `transport/http` 包可提供 `type Server = khttp.Server` 类型别名（标注 Deprecated），但**禁止**再保留独立 `Server` 结构体定义与 `NewServer(...)` 构造函数；type alias 必须在 `v3.0.0` 之前删除

## 修改需求

### 需求:删除拷贝代码

next 仓库**必须**不再保留与 kratos 上游高度相似的拷贝代码，**禁止**新增此类拷贝。例外：`internal/{host,endpoint,matcher}/` 因 kratos 未公开公共 API 必须保留为自维护拷贝。

#### 场景:transport/http 拷贝代码清零

- **当** 检查 `next/transport/http/` 目录
- **那么** 必须不存在 `server.go`、`client.go`、`codec.go`、`codec_go1.20.go`、`context.go`、`calloption.go`、`response.go`、`redirect.go`、`resolver.go`、`filter.go`、`transport.go`、`binding/`、`status/` 等从 kratos 拷贝的文件或子包；仅允许保留 `transport/http/deprecated.go`（仅 type alias，1 个 release cycle 后删除）

#### 场景:transport/grpc 拷贝代码清零

- **当** 检查 `next/transport/grpc/` 目录
- **那么** 必须不存在 `server.go`、`client.go`、`codec.go`、`balancer.go`、`interceptor.go`、`transport.go` 等从 kratos 拷贝的文件；`resolver/` 子包是否保留由独立评估决定（不在本变更范围内）

#### 场景:internal 包必须保留

- **当** 检查 `next/internal/{host,endpoint,matcher}/` 目录
- **那么** 这些包**必须保留**为 next 自维护的拷贝。原因：kratos 把它们放在 `kratos/v2/internal/` 不公开 API，外部包 import 会编译失败
- **后续工作**：评估能否上游推动 kratos 公开这些工具包，届时再考虑替换

## 新增需求

### 需求:next 独有 Router DSL v2

next 必须提供独立子包 `transport/httpx` 实现 Router DSL，不持有自定义 Server 类型，而是以 `*khttp.Server` 作为底层依赖。

#### 场景:Router 构造与 kratos Server 解耦

- **当** 业务调用 `httpx.NewRouter(srv *khttp.Server, prefix string, m ...MiddlewareFunc)`
- **那么** 返回的 Router 必须通过 `srv.Route(prefix)` 在 kratos Server 上注册路由组，不依赖任何 next 自维护的 Server 结构体

#### 场景:Router 支持 GET/POST/PUT/DELETE/Group 等方法

- **当** 业务调用 `r.GET(path, handler, middleware...)` 等方法
- **那么** Router 必须正确将 handler 注册到底层 kratos Server，并保证与 v1 nexthttp.Router 等价的行为（路由匹配、子组、middleware 顺序）

#### 场景:Context 接口保留并解耦

- **当** 业务实现 `func(httpx.Context) error` 风格的 handler
- **那么** `httpx.Context` 必须提供 `Request() / Response() / Bind(v) / JSON(code, v) / Vars()` 等方法，且实现不依赖任何 next 自维护的 Server 结构体（基于 `http.ResponseWriter + *http.Request` + `kratos.binding`）

#### 场景:Context 对象池化

- **当** 并发请求触发 Router handler
- **那么** httpx 必须使用 `sync.Pool` 复用 Context 实例，避免每请求堆分配

### 需求:旧 transport/http 包 API 兼容性

为减小业务方升级压力，删除拷贝代码后 `transport/http` 包必须在 1 个 release cycle（v2.1.0）内保留必要的 type alias。

#### 场景:type alias 提供过渡

- **当** v1 业务代码 import `nexthttp "github.com/nextmicro/next/transport/http"` 并使用 `*nexthttp.Server`
- **那么** v2.1.0 必须通过 `type Server = khttp.Server` 等 alias 让其继续编译通过；编译时必须有 Deprecated warning（通过 `// Deprecated:` 注释）

#### 场景:type alias 在 v3.0.0 前删除

- **当** v3.0.0 发布
- **那么** `transport/http/deprecated.go` 必须删除，业务必须完全迁移到 `transport.NewHTTPServer` 与 `transport/httpx`

### 需求:迁移文档完备

本变更必须提供完整的业务方升级文档。

#### 场景:MIGRATION.md 包含 v1→v2 代码对照

- **当** 业务方查阅 `docs/MIGRATION.md`
- **那么** 必须找到"v1 nexthttp.Server → v2 transport.NewHTTPServer + httpx.Router"的完整代码对照示例，包括 Server 创建、路由注册、middleware 应用、handler 签名

#### 场景:批量替换脚本

- **当** 业务方升级 import 路径
- **那么** MIGRATION.md 必须提供 sed/gofmt 风格的批量替换命令示例（如 `transport/http/binding` → `kratos/transport/http/binding`）
