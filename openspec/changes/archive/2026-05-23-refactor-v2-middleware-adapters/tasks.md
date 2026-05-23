## 1. 准备工作

- [x] 1.1 创建分支 `feat/v2-middleware-adapters`（本地开发）
- [x] 1.2 全量基线测试通过
- [x] 1.3 详细对照表：见 `docs/middleware-adapter-mapping.md`
- [x] 1.4 业务方 yaml 配置 — proto 定义未变，factory 名称不变
- [ ] 1.5 录制基线行为：tracing span 样本、Prometheus metric 样本（可选，发布前）

## 2. middleware/bbr 重写

- [x] 2.1–2.7 全部完成（adapter/option/config/factory，代理 kratos ratelimit）

## 3. middleware/circuitbreaker 重写

- [x] 3.1–3.7 全部完成（adapter + trigger.go 保留 ratio 逻辑）

## 4. middleware/metadata 重写

- [x] 4.1–4.7 全部完成（完全代理 kratos metadata + config_test）

## 5. middleware/recovery 重写

- [x] 5.1–5.7 全部完成（代理 kratos recovery + WithHandler + config_test）

## 6. middleware/metrics 重写

- [x] 6.1–6.4 adapter/option/config/factory 结构化完成
- [x] 6.5 删除旧 metrics.go
- [ ] 6.6–6.7 指标行为契约测试（Prometheus 仍用 contrib，非 kratos OTEL middleware）
- [x] 6.8 adapter.go ≤ 100 行

## 7. middleware/tracing 重写

- [x] 7.1 adapter.go 基于 `ktracing.NewTracer`
- [x] 7.2 option.go：`DefaultPropagator`（b3 + Metadata）、WithDisabled 等
- [x] 7.3 config.go + config_test
- [x] 7.4 factory.go
- [x] 7.5 删除 tracing.go、metadata.go（合并到 adapter/option）
- [x] 7.6 保留 span.go、statshandler.go（next httpconv enrichment）
- [x] 7.7 config_test（propagator/optionsFromConfig）
- [ ] 7.8 集成测试：跨服务 tracing 链路（可选）
- [ ] 7.9 adapter.go 155 行（超出 100 行，已 PR 说明：span 状态记录）

## 8. middleware/logging 轻量清理

- [x] 8.1 修正 option.go 注释（WithDisabled）
- [x] 8.2 文档化：`docs/middleware-logging-fields.md`
- [x] 8.3 现有测试通过
- [x] 8.4 变更 ≤ 50 行

## 9. Spec 同步与文档

- [x] 9.1 同步到 `openspec/specs/middleware/spec.md`
- [x] 9.2 移除"渐进式"过渡条款
- [x] 9.3 适配器约束已写入主 spec
- [x] 9.4 更新 `docs/MIGRATION.md` §7b
- [x] 9.5 `middleware/all/all.go` 无需修改

## 10. 验证与发布

- [x] 10.1 全量 `go test -race ./...` 通过
- [x] 10.6 `openspec-cn validate refactor-v2-middleware-adapters --strict` 通过
- [ ] 10.2–10.5 发布前 review / 业务方验证

## 11. 发布与归档

- [ ] 11.1–11.5 RC 发布与归档
