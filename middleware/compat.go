package middleware

import (
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	log "github.com/nextmicro/logger"
	configv1 "github.com/nextmicro/next/api/config/v1"
)

// DefaultRegistry 是包级默认 Registry，供 sub-package 的 init() 注册 factory 使用。
//
// Deprecated: 计划在 Phase 5 移除。新代码应通过 container.Resolve[middleware.Registry](c)
// 获取由 next.App 持有的 Registry 实例，并显式注册 factory。
// 此处保留仅为让 `middleware/{tracing,recovery,...}` 等子包的 init() 自注册
// 在过渡期继续工作。
var DefaultRegistry = NewRegistry()

// Register 注册一个 factory 到 DefaultRegistry。
//
// Deprecated: 使用 container.Resolve[middleware.Registry](c).Register(name, factory) 代替。
func Register(name string, factory Factory) {
	DefaultRegistry.Register(name, factory)
}

// Create 从 DefaultRegistry 创建一个 middleware。
//
// Deprecated: 使用 container.Resolve[middleware.Registry](c).Create(cfg) 代替。
func Create(cfg *configv1.Middleware) (middleware.Middleware, error) {
	return DefaultRegistry.Create(cfg)
}

// BuildMiddleware 兼容旧调用形态：按 "<transport>.<role>" 的复合 kind 构建中间件链。
//
// 新实现相对于旧版本的改动：
//   - 不再修改入参 _ms 中任何元素的 Name 字段（无 in-place mutation）
//   - 不再要求注册名前缀 "next.middleware."
//   - scope 从 kind 中自动解析：去掉 "http." 或 "grpc." 后剩下的部分（通常为 "server" 或 "client"）
//   - 未注册的 middleware 仅记录 Warn 日志而非 Error，以保持向后兼容
//
// Deprecated: 使用 container.Resolve[middleware.Registry](c).Build(scope, configs) 代替。
func BuildMiddleware(kind string, configs []*configv1.Middleware) ([]middleware.Middleware, error) {
	scope := scopeFromKind(kind)
	ms, err := DefaultRegistry.BuildWithOptions(scope, configs, WarnOnMissing(func(name string) {
		log.Warnf("middleware: skip not registered %q (kind=%q)", name, kind)
	}))
	if err != nil {
		return nil, err
	}
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		log.Infof("middleware: registered %q (kind=%q)", cfg.GetName(), kind)
	}
	return ms, nil
}

// scopeFromKind 从形如 "http.server" / "grpc.client" 的 kind 中提取 scope。
// 已知 transport 前缀（http./grpc.）被剔除；其余原样返回。
func scopeFromKind(kind string) string {
	s := strings.ToLower(kind)
	s = strings.TrimPrefix(s, "http.")
	s = strings.TrimPrefix(s, "grpc.")
	return s
}
