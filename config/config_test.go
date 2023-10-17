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

func TestConfig_Init(t *testing.T) {
	data := []byte(`{"database":{"datasource":"user:password@tcp(localhost:port)/db?charset=utf8mb4\u0026parseTime=True\u0026loc=Local","host":"localhost","password":"password"},"foo":"bar"}`)
	var raw db
	err := json.Unmarshal(data, &raw)
	assert.NoError(t, err)

	path := filepath.Join(os.TempDir(), "dev.yaml")
	fh, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		fh.Close()
		os.Remove(path)
	}()
	_, err = fh.Write(data)
	if err != nil {
		t.Error(err)
	}

	cfg, err := config.Init(path)
	assert.NoError(t, err)

	var db db
	err = cfg.Scan(&db)
	assert.NoError(t, err)
	assert.Equal(t, raw, db)
}

func TestConfig_InitA(t *testing.T) {
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
	assert.NoError(t, err)

	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	_, err = fh.Write([]byte(data))
	assert.NoError(t, err)

	cfg, err := config.Init(path)
	assert.NoError(t, err)

	var db db
	err = cfg.Scan(&db)
	assert.NoError(t, err)

	assert.Equal(t, "127.0.0.1:6379", db.Redis.Address)
	assert.Equal(t, "user:password@tcp(localhost:port)/db?charset=utf8mb4&parseTime=True&loc=Local", db.Database.Datasource)
}

func TestBizConfPath(t *testing.T) {
	path := config.BizConfPath()
	t.Log(path)
}
