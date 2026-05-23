## 上下文

### 当前状态

`next/middleware/{tracing,recovery,metrics,metadata,circuitbreaker,bbr,logging}` 共 7 个 sub-package，1825 行代码。除 `logging` 外，其余 6 个都与 kratos 上游同名 middleware 存在大量重复逻辑：

```
middleware/tracing/  498 行
├── tracing.go        202  ← 与 kratos.tracing.Server/Client 大量重复
├── span.go           160  ← 与 kratos.tracing.span 几乎相同
├── option.go          51  ← next 特有 b3 propagator option
├── metadata.go        45  ← next 特有 metadata 提取
└── statshandler.go    40  ← grpc stats handler，kratos 已有等价

middleware/recovery/  161 行
├── recovery.go        82  ← 与 kratos.recovery.Recovery 几乎相同
└── option.go          79  ← next 特有 Sentry hook + Logger 集成

middleware/metrics/   170 行 (单文件)
   └── metrics.go         ← 与 kratos.metrics.Server/Client 重复 + next Prometheus 命名约定

middleware/metadata/  168 行 (单文件)
   └── metadata.go        ← 与 kratos.metadata.Server/Client 重复 + next prefix 约定

middleware/circuitbreaker/  143 行 (单文件)
   └── circuitbreaker.go  ← 与 kratos.circuitbreaker 几乎相同（都基于 aegis sre）

middleware/bbr/        63 行 (单文件)
   └── bbr.go             ← 与 kratos.ratelimit (bbr) 完全相同

middleware/logging/   622 行
├── logging.go        395  ← next 业务日志字段约定（必须保留）
├── option.go         102
└── logging_test.go   125
```

### 关键约束

1. **业务方 yaml 配置不能改**。业务现在用：
   ```yaml
   http:
     middlewares:
       - name: server.tracing
         options: { ... }
   ```
   重写后，相同 yaml 必须产生等价行为。

2. **next 业务定制必须保留**：
   - `tracing`: b3 header propagator（next 业务规范）、metadata 提取规则、span 名生成约定
   - `recovery`: Sentry 集成 + 自定义 panic logger
   - `metrics`: Prometheus 标签命名（`http_server_requests_total{method,path,code}`）
   - `metadata`: prefix 规则（`x-md-*` → `md.*`）

3. **kratos middleware 已暴露足够的 Option**：经核查 kratos middleware 上游 API 完整，所有 next 业务定制均可通过 Option 注入。

### 利益相关者

- next 核心维护者：减少维护负担
- 业务方：零感知（yaml 配置不变）
- 第三方扩展作者：sub-package 内部 API 变化，需小幅调整

## 目标 / 非目标

**目标：**

1. **删除约 900 行重复实现**，每个 sub-package 主文件 ≤ 100 行
2. **保留所有 next 业务定制能力**（通过 kratos middleware Option 注入）
3. **业务 yaml 配置零修改**，行为完全等价
4. **Option 翻译层标准化**：每个 sub-package 提供 `optionsFromConfig(cfg) ([]Option, error)`
5. **logging 保持独立**，仅做必要清理（约 -50 行）

**非目标：**

1. 不改变 yaml 配置字段名（向后兼容）
2. 不新增 middleware 能力
3. 不处理 `logging` 主体逻辑（独立维护）
4. 不修改 Registry / Builder 装配机制（已在主重构完成）
5. 不影响 `transport` 子包（与 Phase 3B 并行）

## 决策

### 决策 1：每个 sub-package 统一目录结构

**选择**：所有适配器 sub-package 统一为以下结构：

```
middleware/<name>/
├── option.go        # next 特有 Option 定义
├── adapter.go       # kratos 适配器（≤ 100 行）
├── config.go        # optionsFromConfig 翻译层（≤ 50 行）
├── factory.go       # Registry factory 注册
└── *_test.go        # 单元测试
```

**理由**：
- 一致性：业务方查阅扩展规则有迹可循
- 可测试性：translation layer 独立测试
- 文件职责清晰

**替代方案**：
- 单文件 `<name>.go` → 否决：超过 200 行不便于评审
- 拆得更细（每个 Option 一个文件） → 否决：过度工程

### 决策 2：Option 类型完全代理 kratos

**选择**：`tracing.Option = kratos_tracing.Option` 类型别名（或包装类型）。

**理由**：
- 业务方直接传递 kratos middleware Option 即可
- 减少 wrapper 类型混淆

**示例**：

```go
// middleware/tracing/option.go
package tracing

import ktracing "github.com/go-kratos/kratos/v2/middleware/tracing"

type Option = ktracing.Option

// next 特有 Option 通过组合返回 ktracing.Option
func WithB3Propagator() Option { /* ... */ }
func WithBusinessNameRule() Option { /* ... */ }
```

### 决策 3：Server / Client 适配器对称

**选择**：每个 sub-package 提供：

```go
func Server(opts ...Option) middleware.Middleware
func Client(opts ...Option) middleware.Middleware
```

直接代理到 kratos 实现：

```go
// middleware/tracing/adapter.go
func Server(opts ...Option) middleware.Middleware {
    opts = append(defaultServerOptions(), opts...)
    return ktracing.Server(opts...)
}
func Client(opts ...Option) middleware.Middleware {
    opts = append(defaultClientOptions(), opts...)
    return ktracing.Client(opts...)
}
```

**理由**：保持与现有 sub-package API 一致，迁移成本低。

### 决策 4：config 翻译层独立函数

**选择**：每个 sub-package 暴露：

```go
// optionsFromConfig 从 configv1.Middleware.Options 反序列化 next 配置，返回 Option 列表。
func optionsFromConfig(cfg *configv1.Middleware) ([]Option, error)
```

factory 内部调用：

```go
func ServerFactory(cfg *configv1.Middleware) (middleware.Middleware, error) {
    opts, err := optionsFromConfig(cfg)
    if err != nil {
        return nil, fmt.Errorf("tracing: %w", err)
    }
    return Server(opts...), nil
}
```

**理由**：translation 独立可测；factory 函数纤薄。

### 决策 5：bbr 直接用 kratos.ratelimit

**选择**：

```go
// middleware/bbr/adapter.go
package bbr

import (
    "github.com/go-kratos/kratos/v2/middleware"
    kratelimit "github.com/go-kratos/kratos/v2/middleware/ratelimit"
)

func Server(opts ...kratelimit.Option) middleware.Middleware {
    return kratelimit.Server(opts...)
}
```

**理由**：kratos.ratelimit 默认使用 bbr 限流器，行为完全一致。无需 next 自有实现。

### 决策 6：circuitbreaker 完全代理

**选择**：

```go
// middleware/circuitbreaker/adapter.go
import kbreaker "github.com/go-kratos/kratos/v2/middleware/circuitbreaker"

func Client(opts ...Option) middleware.Middleware {
    return kbreaker.Client(opts...)
}
```

**理由**：next 现有实现就是 kratos 实现的复制版，无任何业务定制需要保留。

### 决策 7：logging 不动主体

**选择**：`middleware/logging` 保留全部 395 行核心实现，仅清理 option.go 中冗余配置项（约 -50 行）。

**理由**：spec 明确"next 业务日志格式独立维护"。

### 决策 8：保留旧 sub-package 测试

**选择**：所有 `*_test.go` 保留，确保新实现通过原有行为测试。

**理由**：测试是 v1 行为契约的最佳证明，不应在重写中丢弃。

## 风险 / 权衡

| 风险 | 缓解措施 |
|-----|---------|
| kratos middleware Option 不够 next 定制需求 | 提交上游 PR；本变更暂保留 next 特有实现作为兜底 |
| 重写后行为微妙不一致（如 span 名生成规则） | 集成测试覆盖每个 sub-package 端到端行为 |
| 第三方扩展直接 import sub-package 内部类型 | 内部 API 变更难免；在 release notes 中说明 |
| Sentry 集成等业务能力丢失 | 通过 Option（HandlerFunc 注入）保留 |
| Prometheus 指标命名变化 | 翻译层强制使用 next 命名约定；e2e 测试断言指标名 |
| logging "轻量清理"实际改动范围扩大 | 严格限定为 option.go 冗余项清理，主逻辑不动 |

## 迁移计划

### 阶段 1：preparing（1 周）

1. 详细对照表：列出每个 sub-package 的 next 特定逻辑 → kratos Option 映射
2. 业务方 yaml 配置全量扫描，确认无未公开字段使用
3. 创建分支 `feat/v2-middleware-adapters`

### 阶段 2：逐个 sub-package 重写（2-3 周）

按从简到繁顺序：

4. `bbr` → 直接代理 kratos.ratelimit
5. `circuitbreaker` → 直接代理 kratos.circuitbreaker
6. `metadata` → kratos.metadata 适配器 + prefix Option
7. `recovery` → kratos.recovery 适配器 + Sentry/logger Option
8. `metrics` → kratos.metrics 适配器 + Prometheus 命名 Option
9. `tracing` → kratos.tracing 适配器 + b3 propagator + metadata 规则 Option

每个 sub-package：
- 实现新结构（option.go / adapter.go / config.go / factory.go）
- 保留原 `*_test.go`，确保通过
- 删除旧 implementation 文件
- benchmark 对比性能

### 阶段 3：logging 轻量清理（3 天）

10. 清理 `option.go` 冗余项
11. 文档化业务日志字段约定到 `docs/`

### 阶段 4：测试与验证（1 周）

12. 全量 `CGO_ENABLED=0 go test -race ./...`
13. 内部业务方端到端验证（next.bff.layout 等）
14. Prometheus 指标对比（v1 vs v2.2.0）
15. Tracing span 对比（v1 vs v2.2.0）

### 阶段 5：发布与归档（1 周）

16. 发布 `v2.2.0`
17. Release Notes 强调"零业务感知"
18. 归档本变更

### 回滚策略

- 每个 sub-package 重写完成后独立打 tag
- 若某 sub-package 行为不一致，可单独 revert 该 sub-package 至 v2.0.0 状态
- 不依赖 `transport-cleanup`，可独立回滚

## 开放问题

1. `recovery` 中的 Sentry 集成是否升级到 sentry-go v0.x 新 API？
2. `tracing` 的 b3 propagator 是否考虑迁移到 OpenTelemetry standard propagator？
3. `metrics` 的 Prometheus 标签是否新增 `service` 维度？（业务方需求）
4. `metadata` prefix 规则是否变更为可配置（当前硬编码 `x-md-*`）？
5. 是否合并发布 `v2.2.0`（含 transport-cleanup），还是分两个 release？
