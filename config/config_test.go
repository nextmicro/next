package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nextmicro/next/config"
	"github.com/stretchr/testify/assert"
)

type db struct {
	Database struct {
		Datasource string `json:"datasource"`
		Host       string `json:"host"`
		Password   string `json:"password"`
	} `json:"database"`
	Foo   string `json:"foo"`
	Redis struct {
		Address string `json:"address"`
	} `json:"redis"`
}

// TestConfig_Init 通过两个子用例覆盖原 TestConfig_Init / TestConfig_InitA。
//
// 合并为单测原因：kratos.config 在 Close 后 watcher goroutine 仍会异步打日志（log.Infof），
// 与下一个测试中 SetLogger 写入 kratos 全局 logger 形成 race（kratos 上游问题）。
// 共享一个 config 实例 + 顺序执行子用例后再 Close 可消除此 race，
// 且保留了对 file source + Scan 行为的覆盖。
func TestConfig_Init(t *testing.T) {
	data := `{
  "database": {
    "datasource": "user:password@tcp(localhost:port)/db?charset=utf8mb4&parseTime=True&loc=Local",
    "host": "localhost",
    "password": "password"
  },
  "foo": "bar",
  "redis": {
    "address": "127.0.0.1:6379"
  }
}`

	path := filepath.Join(os.TempDir(), "dev.yaml")
	fh, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fh.Close()
		_ = os.Remove(path)
	})

	if _, err := fh.Write([]byte(data)); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Init(path)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = cfg.Close() })

	t.Run("Scan", func(t *testing.T) {
		var want db
		if err := json.Unmarshal([]byte(data), &want); err != nil {
			t.Fatal(err)
		}
		var got db
		err = cfg.Scan(&got)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("RedisAddress", func(t *testing.T) {
		var got db
		err := cfg.Scan(&got)
		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1:6379", got.Redis.Address)
		assert.Equal(t,
			"user:password@tcp(localhost:port)/db?charset=utf8mb4&parseTime=True&loc=Local",
			got.Database.Datasource)
	})
}

func TestBizConfPath(t *testing.T) {
	path := config.BizConfFile()
	t.Log(path)
}
