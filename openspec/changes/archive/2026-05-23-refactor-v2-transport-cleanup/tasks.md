## 1. 准备与基线

- [x] 1.1 创建分支 `feat/v2-transport-cleanup`（本地开发，未 push）
- [x] 1.2 全量基线测试：`CGO_ENABLED=0 go test -race ./... 2>&1 | tee baseline_test.log`
- [x] 1.3 统计待删除代码行数：`wc -l transport/http/*.go transport/grpc/*.go > cleanup_baseline.txt`（5314 行）
- [x] 1.4 列出业务方依赖的 `nexthttp.Server` 字段、方法 — 本仓库仅 `middleware/tracing` 依赖 `Transporter`，已迁移至 `khttp.Transporter`

## 2. 设计与新 Router DSL 实现（transport/httpx）

- [x] 2.1 创建 `transport/httpx/` 子包目录
- [x] 2.2 实现 `httpx/types.go`：定义 `Router`、`HandlerFunc`、`MiddlewareFunc`、`Context` 接口
- [x] 2.3 实现 `httpx/context.go`：基于 `http.ResponseWriter + *http.Request + kratos/transport/http/binding` 的 Context 实现，含 `sync.Pool` 复用
- [x] 2.4 实现 `httpx/router.go`：`NewRouter(srv *khttp.Server, prefix string, m ...MiddlewareFunc) Router`
- [x] 2.5 实现 GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS 方法
- [x] 2.6 实现 `Group(prefix, m ...) Router` 子路由
- [x] 2.7 实现 `Use(m ...MiddlewareFunc)` 链式 middleware 注入
- [x] 2.8 实现 `httpx/error.go` + `httpx/codec.go`：默认 EncodeErrorFunc 与 EncodeResponseFunc（CustomResponse 信封）
- [x] 2.9 单元测试：路由匹配、子组、middleware 顺序、Context 池化、并发安全
- [ ] 2.10 benchmark：与 v1 nexthttp.Router 对比 routing 与 handler 调度性能（v1 已删除，跳过）

## 3. transport/http type alias 过渡层

- [x] 3.1 在 `transport/http/` 新建 `deprecated.go`，提供 Server 薄包装 + type alias + Route 过渡 API
- [x] 3.2 删除 `transport/http/server.go`、`client.go`、`codec.go`、`codec_go1.20.go`、`context.go`、`calloption.go`、`response.go`、`redirect.go`、`resolver.go`、`filter.go`、`transport.go`
- [x] 3.3 删除 `transport/http/router.go`（功能已迁移到 `transport/httpx/`）
- [x] 3.4 删除 `transport/http/binding/` 子包
- [x] 3.5 删除 `transport/http/status/` 子包
- [x] 3.6 删除 `transport/http/*_test.go` 中针对已删除代码的测试

## 4. transport/grpc type alias 过渡层

- [x] 4.1 在 `transport/grpc/` 新建 `deprecated.go`，提供 Server/Client type alias 与 kratos 选项 re-export
- [x] 4.2 删除 `transport/grpc/server.go`、`client.go`、`codec.go`、`balancer.go`、`interceptor.go`、`transport.go`
- [x] 4.3 评估并保留 `transport/grpc/resolver/{direct,discovery}/`（独立 micro-change 处理；本变更不动）
- [x] 4.4 删除 `transport/grpc/*_test.go` 中针对已删除代码的测试

## 5. 上层引用修复

- [x] 5.1 全仓 grep `transport/http/binding` 与 `transport/http/status` — 本仓库无引用，已删除子包
- [x] 5.2 全仓 grep `transport/http/router` — 无引用；tracing 已改用 `khttp.Transporter`
- [x] 5.3 修复 `transport/factory_test.go` 中可能因为重命名失败的引用 — 无需修改
- [x] 5.4 全仓 `go vet ./...` 通过
- [x] 5.5 全仓 `CGO_ENABLED=0 go build ./...` 通过

## 6. 内部业务试点升级（next.bff.layout 示例）

- [ ] 6.1–6.6 外部业务仓库不在本 monorepo，由业务方自行升级（见 docs/MIGRATION.md §7）

## 7. Spec 修订与文档

- [x] 7.1 将本变更 `specs/transport/spec.md` 同步到 `openspec/specs/transport/spec.md`
- [x] 7.2 移除主 spec 中的"渐进式"过渡条款
- [x] 7.3 更新 `docs/MIGRATION.md`：新增 "v1 nexthttp → v2 transport.NewHTTPServer + httpx" 章节
- [x] 7.4 提供 sed 批量替换脚本示例（import 路径）
- [x] 7.5 更新 `examples/quickstart/main.go`：演示 httpx.Router 用法
- [x] 7.6 在 `README.md` 标注 transport 子包已清理

## 8. 验证与发布前 review

- [x] 8.1 全量回归：`CGO_ENABLED=0 go test -race ./...` 通过
- [x] 8.2 对比 baseline 与 final 测试结果 — 全部通过，0 race
- [x] 8.3 统计实际删除行数：`75 files changed, +1088 −9193`（净减少 **8105 行**，超过 3500 目标）
- [x] 8.4 运行 `openspec-cn validate refactor-v2-transport-cleanup --strict` 通过
- [ ] 8.5 发布 RC：`v2.1.0-rc.1`，发出业务方升级公告
- [ ] 8.6 RC 反馈窗口 2 周

## 9. 正式发布与归档

- [ ] 9.1 RC 反馈处理完毕后发布 `v2.1.0` 正式版
- [ ] 9.2 GitHub Release Notes 明确"删除约 4000 行 kratos 拷贝代码"
- [ ] 9.3 `openspec-cn archive refactor-v2-transport-cleanup`
- [ ] 9.4 记录后续工作：v3.0.0 删除 `transport/{http,grpc}/deprecated.go` 中所有 type alias
