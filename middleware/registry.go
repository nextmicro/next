// Package middleware 提供基于配置的中间件装配能力。
//
// 设计原则：
//   - Registry 是可实例化的，由 Container 注入，不再使用全局单例
//   - Factory 注册名使用 "<scope>.<name>" 全限定形式（例如 "server.tracing"）
//   - Build 时严格不修改输入 configs（无 in-place mutation）
//   - 各 factory 优先适配 kratos 原生中间件，减少重复实现
package middleware

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/middleware"
	configv1 "github.com/nextmicro/next/api/config/v1"
)

// ErrNotFound 表示请求的中间件名称未注册。
var ErrNotFound = errors.New("middleware: factory not registered")

// Factory 根据配置构造一个 kratos middleware。
type Factory func(*configv1.Middleware) (middleware.Middleware, error)

// Registry 是中间件 Factory 的注册表与装配器。
// 同一进程允许多个 Registry 实例，互不影响。
type Registry interface {
	// Register 注册一个 Factory。
	// name 必须为全限定名："<scope>.<name>"，例如 "server.tracing"、"client.logging"。
	// 重复注册会覆盖。
	Register(name string, factory Factory)

	// Create 根据单个配置创建一个 middleware。
	// 查找时按 name 大小写不敏感匹配；未注册返回 ErrNotFound。
	Create(cfg *configv1.Middleware) (middleware.Middleware, error)

	// Build 根据 scope 和配置列表构造中间件链。
	// scope 通常为 "server" 或 "client"。
	// 入参的 configs 切片不会被修改（无 in-place mutation）。
	// 配置列表中未注册的 name 默认返回错误；通过 BuildWithOptions(WarnOnMissing) 可改为仅记录警告。
	Build(scope string, configs []*configv1.Middleware) ([]middleware.Middleware, error)

	// BuildWithOptions 与 Build 等价，但允许细化行为。
	BuildWithOptions(scope string, configs []*configv1.Middleware, opts ...BuildOption) ([]middleware.Middleware, error)
}

// BuildOption 配置 Build 行为。
type BuildOption func(*buildOptions)

type buildOptions struct {
	warnOnMissing bool
	missingLogger func(name string)
}

// WarnOnMissing 配置 Build：未注册的 middleware 名称仅记录警告并跳过，而非返回错误。
func WarnOnMissing(logger func(name string)) BuildOption {
	return func(o *buildOptions) {
		o.warnOnMissing = true
		o.missingLogger = logger
	}
}

// ============================================================
// 默认实现
// ============================================================

type defaultRegistry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry 创建一个新的 Registry 实例。
func NewRegistry() Registry {
	return &defaultRegistry{
		factories: make(map[string]Factory),
	}
}

func (r *defaultRegistry) Register(name string, factory Factory) {
	key := normalizeName(name)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[key] = factory
}

func (r *defaultRegistry) Create(cfg *configv1.Middleware) (middleware.Middleware, error) {
	if cfg == nil {
		return nil, fmt.Errorf("middleware: nil config")
	}
	key := normalizeName(cfg.GetName())
	r.mu.RLock()
	factory, ok := r.factories[key]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, cfg.GetName())
	}
	return factory(cfg)
}

func (r *defaultRegistry) Build(scope string, configs []*configv1.Middleware) ([]middleware.Middleware, error) {
	return r.BuildWithOptions(scope, configs)
}

// BuildWithOptions 按 scope 装配中间件链：
//  1. 不修改入参 configs 中任一元素
//  2. 为每个 cfg 派生一个 scoped name = "<scope>.<cfg.Name>"，查找对应 factory
//  3. 按 configs 出现顺序返回中间件切片（不再反转）
func (r *defaultRegistry) BuildWithOptions(scope string, configs []*configv1.Middleware, opts ...BuildOption) ([]middleware.Middleware, error) {
	bo := &buildOptions{}
	for _, o := range opts {
		o(bo)
	}

	ms := make([]middleware.Middleware, 0, len(configs))
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		fullName := scope + "." + cfg.GetName()
		key := normalizeName(fullName)

		r.mu.RLock()
		factory, ok := r.factories[key]
		r.mu.RUnlock()
		if !ok {
			err := fmt.Errorf("%w: %q (scope=%q)", ErrNotFound, cfg.GetName(), scope)
			if bo.warnOnMissing {
				if bo.missingLogger != nil {
					bo.missingLogger(fullName)
				}
				continue
			}
			return nil, err
		}

		m, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("middleware %q (scope=%q): %w", cfg.GetName(), scope, err)
		}
		ms = append(ms, m)
	}
	return ms, nil
}

// normalizeName 将名称转为大小写不敏感的查找键。
// 不再添加 "next.middleware." 前缀（旧版的做法），保持名称即所见。
func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
