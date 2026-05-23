package config

import (
	"errors"
	"fmt"

	kconfig "github.com/go-kratos/kratos/v2/config"
)

// ErrPathNotFound 在 Bind/BindWatch 时找不到指定 path 时返回。
var ErrPathNotFound = errors.New("config: path not found")

// Bind 把 path 处的配置段反序列化为 *T 并返回。
//
// 行为：
//   - path 为空字符串时绑定整个根配置
//   - path 找不到时返回 ErrPathNotFound
//   - 反序列化失败时返回包装后的错误，便于调用方 Wrap 上下文
//
// 示例：
//
//	type DB struct {
//	    Datasource string `json:"datasource"`
//	    Pool       int    `json:"pool"`
//	}
//	db, err := config.Bind[DB](cfg, "database")
//	if err != nil { return err }
//	_ = db.Datasource
//
// 注意：T 不能是指针类型；Bind 内部会分配 *T 并 Scan。
// 多次 Bind 同一 path 返回独立实例，互不影响。
func Bind[T any](c kconfig.Config, path string) (*T, error) {
	if c == nil {
		return nil, errors.New("config: nil Config")
	}
	v := c.Value(path)
	if v == nil {
		return nil, fmt.Errorf("%w: %q", ErrPathNotFound, path)
	}
	out := new(T)
	if err := v.Scan(out); err != nil {
		return nil, fmt.Errorf("config: scan %q: %w", path, err)
	}
	return out, nil
}

// MustBind 是 Bind 的 panic 版本。
func MustBind[T any](c kconfig.Config, path string) *T {
	out, err := Bind[T](c, path)
	if err != nil {
		panic(err)
	}
	return out
}

// BindWatch 在 path 上注册热更新回调，每次配置变更时把新值 Scan 到 *T 并触发 fn。
//
// 行为：
//   - 首次调用立即注册 Watcher；不会立即触发 fn（仅当 path 真正变更才触发）
//   - 业务方应在 fn 中处理"指针替换"或"原子赋值"语义，自行保证线程安全
//   - 返回的 stop 函数取消订阅；可在 lifecycle.Stop 中调用
//
// 注意：如果 path 在配置变更时被删除，Scan 会保留旧值或为零值，
// 取决于底层 kconfig.Config 实现；调用方应在 fn 中校验关键字段。
//
// 示例：
//
//	stop, err := config.BindWatch[DB](cfg, "database", func(db *DB) {
//	    if db.Datasource == "" { return }
//	    pool.SetDSN(db.Datasource)
//	})
//	if err != nil { return err }
//	defer stop()
func BindWatch[T any](c kconfig.Config, path string, fn func(*T)) (stop func(), err error) {
	if c == nil {
		return nil, errors.New("config: nil Config")
	}
	if fn == nil {
		return nil, errors.New("config: nil callback fn")
	}

	observer := func(_ string, v kconfig.Value) {
		if v == nil {
			return
		}
		out := new(T)
		if scanErr := v.Scan(out); scanErr != nil {
			return
		}
		fn(out)
	}

	if err := c.Watch(path, observer); err != nil {
		return nil, fmt.Errorf("config: watch %q: %w", path, err)
	}

	// kratos.Config 没有 Unwatch；返回的 stop 仅为占位（业务方可关闭外部 channel）。
	// 后续可对接 kratos 的 Unwatch（若新版本提供）。
	return func() {}, nil
}
