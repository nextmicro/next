## 为什么

`refactor-v2-architecture` 主重构变更已完成 5 大核心架构升级（container / lifecycle / typed broker / middleware registry / config typed），但 **transport/{http,grpc} 子包下约 3500 行从 kratos 复制的代码**未在该变更范围内删除。

不删除的原因（已记录在归档变更的 tasks.md 中）：
- `transport/http/router.go` 是 next 独有的 mux 路由 DSL（Router 接口 + Context 抽象 + Group/GET/POST 等方法），强耦合自维护的 `*nexthttp.Server` 类型，**删除前需要重新设计**基于 kratos.HTTPServer 的替代方案
- 旧 Server 类型是 v1 业务方主入口，硬删会让所有 v1 main.go 编译失败

本变更专门处理这一遗留问题，使 v2 真正达成"装饰扩展层"定位 —— 仓库内零拷贝 kratos 代码（`internal/{host,endpoint,matcher}` 除外，因 kratos 未公开公共 API，已在主 spec 中注明）。

## 变更内容

### **BREAKING**：删除 next 自维护的 `transport/http.Server` 与 `transport/grpc.Server` 类型

- 删除 `transport/http/{server.go, client.go, codec.go, codec_go1.20.go, context.go, calloption.go, response.go, redirect.go, resolver.go, filter.go, transport.go}` 共约 2350 行
- 删除 `transport/http/{binding,status}/` 子包（kratos 已有公开等价物）
- 删除 `transport/grpc/{server.go, client.go, codec.go, balancer.go, interceptor.go, transport.go}` 共约 1130 行
- 评估 `transport/grpc/resolver/{direct,discovery}/`：与 kratos resolver 对比，重复则删除

### 设计并实现 next Router DSL v2

- 现有 Router DSL（`GET/POST/Group/Handle` + `Context` 接口 + `MiddlewareFunc` 适配）改造为基于 `*khttp.Server` 的 wrapper
- 新位置：`transport/httpx/router.go`（与 `transport/http.go` 工厂同包族）
- 保持业务方迁移代价低：`router.Router`、`router.Handler`、`router.HandlerFunc` 等接口签名不变

### 删除旧 Server 后的兼容入口

- 在 `transport/http/` 与 `transport/grpc/` 顶层提供 type alias：`type Server = khttp.Server`（暂时），引导业务方 1-2 个版本内迁移到 `transport.NewHTTPServer`
- 业务方升级路径文档化到 `docs/MIGRATION.md` 的新章节

### 同步更新 `transport` capability 主规范

- 移除 spec 中"渐进式"条款，回归 spec 设计初衷（"不存在 next/transport/http.Server 类型"）
- 标注 Phase 3B 已完成

## 功能 (Capabilities)

### 新增功能

无新增 capability（本变更专注清理 + 已有 capability 完善）。

### 修改功能

- `transport`: 移除"渐进式"过渡条款，要求不再导出 `transport/http.Server` 与 `transport/grpc.Server` 自定义类型；新增 next Router DSL v2 规范条款

## 影响

### 受影响代码

| 范围 | 变更类型 | 估算 |
|-----|--------|------|
| `transport/http/*.go`（11 个文件） | 删除 | ~2350 行 |
| `transport/http/{binding,status}/` | 删除 | ~580 行 |
| `transport/grpc/*.go`（6 个文件） | 删除 | ~1130 行 |
| `transport/grpc/resolver/` | 评估，可能删除 | ~400 行 |
| `transport/httpx/router.go`（新 Router DSL） | 新增 | ~250 行 |
| 业务方代码 | **必须改造** | 各业务自评 |

预计删除 ~4000 行 + 新增 ~250 行 = **净减少约 3750 行**。

### API 破坏面

- `*nexthttp.Server` 类型不再存在 → 业务必须改用 `*khttp.Server`（kratos 原生）
- `nexthttp.NewServer(...)` 不再存在 → 业务必须改用 `transport.NewHTTPServer(transport.HTTPServerConfig{...})`
- `nexthttp.Context` / `nexthttp.MiddlewareFunc` / `nexthttp.Router` 等仍提供（在新位置 `transport/httpx/`，包名不同）
- `transport/http/binding`、`transport/http/status` 删除 → 业务改用 `kratos/transport/http/binding`、`kratos/errors`

### 依赖

- 上游：`kratos/v2/transport/http` 与 `kratos/v2/transport/grpc` 已被新工厂使用，无新增依赖
- 下游：所有依赖 `nexthttp.Server` 类型的业务必须升级

### 系统

- 编译期：业务方升级前编译失败（这是预期的 breaking change）
- 运行时：行为等价（kratos 原生 Server 与原 nexthttp.Server 在路由、middleware 装配等方面一致）
- 测试：删除大量旧测试文件；新 Router DSL 补齐单测

### 兼容性策略

- 本变更建议在 `v2.1.0` 或 `v3.0.0` 发布
- 发布前与业务方协调升级窗口
- 在 `docs/MIGRATION.md` 提供完整的"v1 nexthttp → v2 transport.NewHTTPServer"代码示例
