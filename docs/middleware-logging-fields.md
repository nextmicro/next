# next 业务日志字段约定（middleware/logging）

`middleware/logging` 为 next 自有实现，不通过 kratos 适配器。以下为 access log 结构化字段约定。

## Server 侧字段

| 字段 | 说明 |
|------|------|
| `start` | 请求开始时间（可配置 `time_format`） |
| `kind` | 固定 `server` |
| `component` | transport 类型（HTTP / GRPC） |
| `route` | operation / 路由模板 |
| `duration` | 耗时（人类可读） |
| `caller_service` | 来自 metadata `x-md-local-caller` |
| `request` | 请求体摘要（`dump_request` 或 error 时） |
| `response` | 响应体摘要（`dump_response` 或 error 时） |
| `code` / `reason` / `error` | kratos 错误信息 |
| metadata 映射字段 | 由 yaml `metadata` 规则注入 |

## Client 侧字段

| 字段 | 说明 |
|------|------|
| `callee_service` | 目标服务名 |
| `callee.address` | 目标节点地址 |
| 其余 | 与 server 类似，`kind=client` |

## 日志级别

- 正常：`Debug`
- 慢请求：超过 `slow_threshold` 时 `Info` / `Error`（含 error 时 Error）
- 失败：`Error`

## yaml 配置

见 `api/middleware/logging/v1`：`disabled`、`time_format`、`slow_threshold`、`ignored_routes`、`access_level`、`dump_request`、`dump_response`、`metadata`。
