package config

import (
	"github.com/go-kratos/kratos/v2/config"
	"github.com/nextmicro/next/pkg/env"
)

var (
	// DefaultConfig is a default config.
	DefaultConfig config.Config
)

// BizConfFile returns the biz config filename.
func BizConfFile() string {
	return env.DeployEnvironment() + ".yaml"
}

// Load loads config from config source.
func Load() error {
	return DefaultConfig.Load()
}

// Scan scans config into v.
func Scan(v interface{}) error {
	return DefaultConfig.Scan(v)
}

// Value gets the config value by key.
func Value(key string) config.Value {
	return DefaultConfig.Value(key)
}

// Watch watches config changes.
func Watch(key string, o config.Observer) error {
	return DefaultConfig.Watch(key, o)
}

// Close closes the config source.
func Close() error {
	return DefaultConfig.Close()
}
