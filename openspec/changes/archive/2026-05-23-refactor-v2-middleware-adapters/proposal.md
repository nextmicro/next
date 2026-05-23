## 为什么

`refactor-v2-architecture` 完成了 middleware **装配层**重构（消除字符串 mutation + 全局单例，新 `Registry` 与 `Builder`），但 7 个 middleware sub-package（`tracing/recovery/metrics/metadata/circuitbreaker/bbr/logging`，共约 **1825 行**）内部仍是 **next 自有实现**，与 kratos 上游同名中间件存在大量重复逻辑。

主重构变更的 spec 已标注："改写为 kratos 纯适配器形态，需逐个 middleware 评估保留哪些定制能力，由后续独立变更承担。"本变更即承担该工作。

### 当前重复度对比

| 子包 | 行数 | 与 kratos 上游差异 | 改造方向 |
|-----|-----|------------------|---------|
| `tracing` | 498 | 包含 next 特定 propagator (b3)、metadata 提取规则、span 命名约定 | 抽取 next 业务定制为 Option，主体改为 kratos.tracing 适配器 |
| `recovery` | 161 | next 自有 panic 捕获 + Sentry/logger 集成 | 改为 kratos.recovery 适配器 + 保留 Sentry hook 作为 Option |
| `metrics` | 170 | next 自有 Prometheus 标签命名（method/path/code） | 改为 kratos.metrics 适配器 + 保留 next 标签约定作为 Option |
| `metadata` | 168 | next 自有 metadata key 前缀规则 | 改为 kratos.metadata 适配器 + 保留 prefix Option |
| `circuitbreaker` | 143 | 基于 aegis sre.NewBreaker，与 kratos.circuitbreaker 几乎相同 | 直接换用 kratos.circuitbreaker + 保留 aegis 配置参数翻译 |
| `bbr` | 63 | 基于 aegis ratelimit，与 kratos.ratelimit/bbr 完全相同 | 直接换用 kratos.ratelimit (bbr 限流器) |
| `logging` | 622 | **next 业务日志字段约定**（business field names） | **保留独立实现**（spec 明确要求） |

### 目标量化

- 删除约 **800-1000 行**重复实现
- 每个适配器 sub-package 主文件 ≤ 100 行（符合 spec 指引）
- `logging` 保持独立，约 600 行不变
- 配置层 100% 向后兼容（业务 yaml 不需要改）

## 变更内容

### 重写以下 6 个 middleware sub-package 为 kratos 适配器形态

- **BREAKING (内部 API)**：sub-package 内部类型与函数签名变更（如 `tracing.Server(...)` → `tracing.Server(opts ...Option)`），但通过 Registry factory 访问的业务代码无感知

| sub-package | 改造动作 |
|------------|---------|
| `middleware/tracing` | 主体改为 `kratos/middleware/tracing` 适配器；将 next 特有的 b3 propagator、metadata 提取规则抽取为 Option 注入 |
| `middleware/recovery` | 主体改为 `kratos/middleware/recovery` 适配器；Sentry 集成作为 HandlerFunc Option |
| `middleware/metrics` | 主体改为 `kratos/middleware/metrics` 适配器；next Prometheus 命名约定作为 Option |
| `middleware/metadata` | 主体改为 `kratos/middleware/metadata` 适配器；prefix 配置作为 Option |
| `middleware/circuitbreaker` | 主体改为 `kratos/middleware/circuitbreaker` 适配器；aegis 参数翻译为 Option |
| `middleware/bbr` | 主体改为 `kratos/middleware/ratelimit` 适配器（kratos.ratelimit 默认使用 bbr） |

### 保留 `middleware/logging` 独立实现

- spec 明确要求 next 业务日志格式独立维护
- 仅做必要重构：清理重复 Option，统一日志字段约定文档

### Factory 注册保持不变

- 业务方使用的 `server.tracing`、`client.tracing` 等 factory 名称不变
- 业务方 yaml 配置不需要任何修改

### 完善 Option 翻译层

- 每个 sub-package 提供 `optionsFromConfig(cfg *configv1.Middleware) ([]Option, error)` 函数
- 集中处理 `anypb.Any` 反序列化与字段映射

## 功能 (Capabilities)

### 新增功能

无新增 capability。

### 修改功能

- `middleware`: 删除"渐进式"过渡条款，要求 7 个 sub-package 中除 logging 外**必须**作为 kratos 适配器实现；新增"适配器代码量约束"为强制要求；新增"Option 翻译层"要求

## 影响

### 受影响代码

| 范围 | 变更类型 | 估算 |
|-----|--------|------|
| `middleware/tracing/*.go`（5 文件，498 行） | 重写 | -400 行 |
| `middleware/recovery/*.go`（2 文件，161 行） | 重写 | -100 行 |
| `middleware/metrics/*.go`（1 文件，170 行） | 重写 | -120 行 |
| `middleware/metadata/*.go`（1 文件，168 行） | 重写 | -120 行 |
| `middleware/circuitbreaker/*.go`（1 文件，143 行） | 重写 | -100 行 |
| `middleware/bbr/*.go`（1 文件，63 行） | 重写 | -40 行 |
| `middleware/logging/*.go`（3 文件，622 行） | 保留并轻量清理 | -50 行 |

预计净减少约 **900 行**。

### API 破坏面

- **业务层**：零破坏。所有 factory 名称、yaml 配置字段、Registry.Build 行为不变
- **内部包 API**：`tracing.Server(...)` / `recovery.Server(...)` 等导出函数签名可能变化。直接 import sub-package 的第三方扩展需小幅调整

### 依赖

- 上游：直接依赖 `kratos/v2/middleware/{tracing,recovery,metrics,metadata,circuitbreaker,ratelimit}`，已在 go.mod
- 副作用：消除对 `aegis` 部分包的直接依赖（kratos 已封装），减小依赖面

### 测试

- 每个适配器 sub-package 新增"翻译层测试"：验证 yaml 配置正确翻译为 Option
- 保留并扩展集成测试：验证最终行为与 v1 一致（追踪 span 名、metrics 标签、recovery 行为等）

### 兼容性策略

- 本变更建议在 `v2.2.0` 发布
- 与 `refactor-v2-transport-cleanup` (Phase 3B) 不互相依赖，可并行开发
- 业务方零感知升级
