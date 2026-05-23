## 上下文

### 当前状态

`refactor-v2-architecture` 已完成 5 大架构升级，但 `transport/{http,grpc}` 子包内约 3500 行 kratos 拷贝代码仍未删除。

```
transport/http/
├── server.go          422 行  ← 与 kratos.HTTPServer 80%+ 相似
├── client.go          443 行  ← 与 kratos HTTP client 80%+ 相似
├── codec.go + go1.20  178 行  ← 与 kratos codec 100% 相似
├── context.go         196 行  ← 与 kratos Context 几乎相同
├── calloption.go      135 行  ← 与 kratos call option 几乎相同
├── response.go         81 行  ← kratos 复制
├── redirect.go         18 行  ← kratos 复制
├── resolver.go        154 行  ← kratos 复制
├── filter.go           16 行  ← kratos 复制
├── transport.go       112 行  ← kratos 复制
├── router.go          175 行  ← ★ next 独有 mux 路由 DSL
├── binding/                    ← kratos 已有等价物
└── status/                     ← kratos 已有等价物

transport/grpc/
├── server.go          303 行  ← 与 kratos.GRPCServer 80%+ 相似
├── client.go          299 行  ← 与 kratos GRPC client 80%+ 相似
├── codec.go            38 行  ← kratos 复制
├── balancer.go        105 行  ← 与 kratos balancer 几乎相同
├── interceptor.go      88 行  ← kratos 复制
├── transport.go        84 行  ← kratos 复制
└── resolver/                   ← 评估
```

### 关键约束

1. **router.go 是 next 独有差异化能力**，不能简单删除。它提供：
   - `Router` 接口（GET/POST/PUT/DELETE/Group 等）
   - `Context` 接口（Reset/Request/Response/Bind 等）
   - `MiddlewareFunc` / `HandlerFunc` 类型
   - 基于 `gorilla/mux` 的子路由 + handler pool
   - 与 `*nexthttp.Server` 紧耦合（依赖 `s.router *mux.Router`、`s.ene EncodeErrorFunc`）

2. **业务方主入口仍是 `*nexthttp.Server`**：示例 `next.bff.layout` 等业务代码深度依赖这个类型

3. **kratos.HTTPServer 不暴露内部 mux.Router**：next Router DSL 不能直接复用 kratos server 的路由器

### 利益相关者

- next 核心维护者：完成 v2 "装饰扩展层"定位
- 业务方：升级 main.go 创建 Server 的代码
- 第三方插件作者：使用 nexthttp.Context / MiddlewareFunc 的扩展

## 目标 / 非目标

**目标：**

1. **删除 `transport/http`、`transport/grpc` 中所有从 kratos 复制的代码**（约 3500 行）
2. **保留 next Router DSL 能力**，但解耦于自维护 Server，迁移到独立子包 `transport/httpx`
3. **业务方代码改造代价可控**：提供 type alias 与平滑升级路径
4. **transport spec 中"渐进式"条款移除**，达成真正零拷贝
5. **单元测试覆盖 Router DSL v2**

**非目标：**

1. 不重写 next.bff.layout 等业务方代码（业务自行升级）
2. 不引入新 Router 能力（保持现有功能集）
3. 不改动 kratos 上游
4. 不处理 `transport/grpc/resolver/`（独立评估，可能保留）
5. 不处理 `internal/{host,endpoint,matcher}/`（已在主 spec 中确认必须保留）

## 决策

### 决策 1：Router DSL v2 不持有自定义 Server 类型

**选择**：Router DSL v2 接受 `*khttp.Server` 作为依赖参数。

**理由**：
- 消除 next 自维护 Server 的根本动机：next 是"装饰层"，不应有自己的 Server 类型
- kratos.HTTPServer 已暴露 `HandlePrefix / Handle / HandleFunc / Walk` 等方法供路由注册
- 业务方升级路径：`nexthttp.NewServer().Route("/api")` → `khttpSrv := transport.NewHTTPServer(...); router := httpx.NewRouter(khttpSrv, "/api")`

**替代方案考虑**：
- 让 Router 自己内嵌 `*khttp.Server` 包装 → 否决：又引入新的"自定义 Server"类型，违背初衷
- 完全删除 Router DSL，让业务直接用 `srv.Route()` → 否决：丢失 next 现有差异化能力

**实现要点**：
```go
package httpx  // transport/httpx/

type Router interface {
    GET(path string, h HandlerFunc, m ...MiddlewareFunc)
    POST(...)
    // ...
    Group(prefix string, m ...MiddlewareFunc) Router
}

// NewRouter 在 khttp.Server 上构建路由组。
// 通过 mux := srv.RoutePrefix(prefix).Subrouter() 注册路由。
func NewRouter(srv *khttp.Server, prefix string, m ...MiddlewareFunc) Router
```

### 决策 2：Context 抽象保留并迁移到 httpx

**选择**：`nexthttp.Context` 接口完整保留，迁移到 `transport/httpx/context.go`，去除对自维护 Server 的依赖。

**理由**：
- Context 是业务方 handler 函数签名（`func(httpx.Context) error`），删除会让所有 handler 签名失败
- Context 自身实现可基于 `http.ResponseWriter + *http.Request`，无需依赖自定义 Server
- 保留 sync.Pool 复用模式

### 决策 3：tail-cleanup 工具包独立

**选择**：删除前为 `transport/http/codec.go`、`response.go` 等纯 kratos 拷贝代码，提供 type alias 1 个版本：

```go
// transport/http/deprecated.go
package http
// Deprecated: use kratos/transport/http directly.
type Codec = khttp.Codec
// ... 其它 alias
```

下一个版本（v3.0.0）才彻底删除 alias。

**理由**：给业务方一个 release cycle 的迁移窗口。

### 决策 4：client.go 与 server.go 同步处理

**选择**：删除 `transport/http/client.go` 与 `transport/grpc/client.go`，新增 `transport/http_client.go` 与 `transport/grpc_client.go` 作为薄装饰工厂（返回 `*khttp.Client` / `*grpc.ClientConn`）。

**理由**：与 server.go 处理保持对称；client 也涉及 middleware 自动装配（依赖 `middleware.Registry`）。

### 决策 5：binding/ 与 status/ 直接删除

**选择**：不提供 type alias，业务直接换 import 路径。

**理由**：这两个包大小适中，业务方一次搜索替换即可。kratos.binding 与 kratos.errors 完全等价。

### 决策 6：grpc/resolver/{direct,discovery}/ 评估保留

**选择**：本变更**不删除** resolver 包，标注为"待评估"。

**理由**：
- direct/discovery 是 grpc resolver scheme 注册，与 kratos resolver 行为有细微差别（next discovery 集成了 `kratos/registry` 接口的具体实现）
- 删除影响面不清晰；推迟到独立 micro-change

## 风险 / 权衡

| 风险 | 缓解措施 |
|-----|---------|
| 业务方升级失败导致 v2.1.0 发布回滚 | 发布 RC 版本，征集 2 周反馈；在 MIGRATION.md 提供详细代码对比 |
| Router DSL v2 与 v1 行为细微差异 | 单元测试覆盖每个 v1 测试用例；提供 v1→v2 对照表 |
| Context.Bind / Context.JSON 等方法依赖 binding/status 包 | 在 Router v2 中重新基于 kratos.binding 实现 |
| `transport/grpc/client.go` 中 `chain.BuildMiddleware` 引用消失影响业务 | 兼容层 `middleware.BuildMiddleware` 在归档变更中已保留 |
| 第三方插件作者使用 `nexthttp.Server` 类型断言 | 不可避免；发布 break change 通知 |
| 删除 binding/ 后业务方 import 失败大批量 | 提供 `go fix` 风格的 sed 脚本（README 中给出） |

## 迁移计划

### 阶段 1：开发与内测（约 2 周）

1. 实现 `transport/httpx/` 新 Router DSL + Context
2. 在 `transport/http/` 增加 type alias（`type Server = khttp.Server`）+ `nexthttp.NewServer` 改为 deprecated wrapper
3. 单元测试 + benchmark 对比 v1/v2 路由性能
4. 内部业务代码升级试点（next.bff.layout）

### 阶段 2：RC 发布（1 周）

5. 发布 `v2.1.0-rc.1`，明确 breaking change 范围
6. 收集业务方反馈
7. 修复 RC 发现问题

### 阶段 3：删除拷贝代码（1 周）

8. 删除 `transport/http/{server.go,client.go,codec.go,...}` 共 11 个文件
9. 删除 `transport/http/{binding,status}/` 子包
10. 删除 `transport/grpc/{server.go,client.go,codec.go,...}` 共 6 个文件
11. 更新 `transport` capability 主规范（移除渐进式条款）
12. 全量 `go vet / go test -race ./...` 通过

### 阶段 4：发布（1 周）

13. 发布 `v2.1.0` 正式版
14. GitHub Release Notes 明确：本版本删除约 4000 行 kratos 拷贝代码
15. 归档本变更到 `openspec/changes/archive/`

### 回滚策略

- 阶段 1-3 任何阶段发现严重问题，可保留 type alias，延期 4000 行删除到 v3.0.0
- v2.1.0 发布后若业务方升级率 < 50%，则 v2.2.0 仍保留 type alias

## 开放问题

1. `transport/http/router.go` 中的 `r.srv.ene` 错误编码器：v2 是否提供自定义 EncodeErrorFunc Option？
2. `transport/grpc/resolver/discovery/` 是否在本变更评估，还是独立 micro-change？
3. type alias 保留多久？1 个版本 (v2.1) 或 2 个版本 (v2.1, v2.2)？
4. 是否同时清理 `pkg/threading`, `pkg/rescue` 等 v1 工具包中与 kratos 重复的部分？（建议独立变更）
