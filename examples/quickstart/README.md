# quickstart

最小可运行 next v2 应用示例。

## 运行

```bash
go run .
```

输出类似：

```
[health-printer] started
greeting prepared: Hello from next v2, next!
lifecycle status: OK, components: 1
INFO[server] starting...
INFO[server] listening on :8080
```

## 验证

```bash
curl http://127.0.0.1:8080/hello?name=cursor
```

预期：

```json
{"message":"hello, cursor"}
```

## 重点

| 行为 | 实现位置 |
|-----|---------|
| HTTP server 创建 + middleware 装配 | `transport.NewHTTPServer` |
| 业务服务注入 DI | `next.WithContainer(...)` + `container.Register` |
| 自定义生命周期组件 | `next.WithComponents(&healthComponent{})` |
| 类型安全依赖解析 | `container.Resolve[*GreeterService](app.Container())` |
| 健康检查聚合 | `app.Lifecycle().Health(ctx)` |
| 信号驱动启动/退出 | `app.Run()`（内部委托 kratos.App） |

## 停止

`Ctrl+C` 触发 SIGINT，App 会：
1. 收到信号，kratos.App 退出
2. `Lifecycle.Stop` 反向停止 `healthComponent`
3. `app.Run()` 返回
