# middleware 适配器映射（Phase 4B）

| sub-package | kratos 适配 | next 保留能力 | 文件结构 |
|-------------|------------|--------------|---------|
| `bbr` | `middleware/ratelimit.Server` + `WithBBRLimiter` | BBR yaml 参数翻译 | adapter/option/config/factory |
| `metadata` | `middleware/metadata.Server/Client` | 默认 prefix（server `x-md-` / client `x-md-global-`） | adapter/config/factory |
| `circuitbreaker` | `middleware/circuitbreaker.Client` | `ratioTrigger` / SRE 参数 | adapter/config/factory/trigger |
| `recovery` | `middleware/recovery.Recovery` | next logger + stack 配置 | adapter/config/factory |
| `metrics` | kratos contrib/prometheus 后端 | next 指标名与 label 维度 | adapter/option/config/factory |
| `tracing` | `middleware/tracing.NewTracer` | b3 propagator、httpconv span、trace 响应头 | adapter/option/config/factory + span |
| `logging` | **独立实现** | next 业务 access log 字段 | logging/option + logging.go |

## 业务影响

- factory 名称不变（`server.tracing` 等）
- yaml 字段不变
- metrics 仍使用 Prometheus（非 kratos OTEL metrics middleware）
- tracing span 名仍为 `HTTP Server {operation}` 格式
