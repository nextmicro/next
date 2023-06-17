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
	Foo string `json:"foo"`
}

func TestConfig_Init(t *testing.T) {
	data := []byte(`{"database":{"datasource":"user:password@tcp(localhost:port)/db?charset=utf8mb4\u0026parseTime=True\u0026loc=Local","host":"localhost","password":"password"},"foo":"bar"}`)
	var raw db
	err := json.Unmarshal(data, &raw)
	assert.NoError(t, err)

	path := filepath.Join(os.TempDir(), "application-dev.json")
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
