package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nextmicro/next/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Init(t *testing.T) {
	data := []byte(`{"database":{"datasource":"user:password@tcp(localhost:port)/db?charset=utf8mb4\u0026parseTime=True\u0026loc=Local","host":"localhost","password":"password"},"foo":"bar"}`)
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

	// env
	//os.Setenv("NEXT_DATABASE_HOST", "localhost")
	//os.Setenv("NEXT_DATABASE_PASSWORD", "password")
	//os.Setenv("NEXT_DATABASE_DATASOURCE", "user:password@tcp(localhost:port)/db?charset=utf8mb4&parseTime=True&loc=Local")

	cfg, err := config.Init(path)
	assert.NoError(t, err)

	res, err := cfg.Value("database").String()
	t.Log(res)
	assert.NoError(t, err)
	assert.Equal(t, string(data), res)
}
